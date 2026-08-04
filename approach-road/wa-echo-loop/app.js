const { Client, LocalAuth } = require('whatsapp-web.js');
const qrcode = require('qrcode-terminal');
const axios = require('axios'); // Still used for image downloading & location
const amqp = require('amqplib');
const crypto = require('crypto');
const PersonaConfig = require('./personaConfig');

// Load persona configuration
const personaName = process.env.PERSONA_NAME || 'default';
const persona = PersonaConfig.loadByName(personaName);
console.log(`📝 ${persona.toString()}`);

// ============= RABBITMQ CLIENT =============
class RabbitMQClient {
    constructor() {
        this.conn = null;
        this.ch = null;
        this.connected = false;
        this.responseCache = new Map(); // Cache responses by correlationId
    }

    async connect() {
        const rabbitmqURL = process.env.RABBITMQ_URL || 'amqp://indieclaw:secretpass@localhost:5672/';
        try {
            this.conn = await amqp.connect(rabbitmqURL);
            this.ch = await this.conn.createChannel();

            // Declare queues
            await this.ch.assertQueue('orchestrator.requests', { durable: true });
            await this.ch.assertQueue('orchestrator.responses', { durable: true, expires: 3600000 });

            this.connected = true;
            console.log('🐰 Connected to RabbitMQ');

            // Start consuming responses
            this.consumeResponses();
        } catch (err) {
            console.error('❌ RabbitMQ connection failed:', err.message);
            this.connected = false;
        }
    }

    async publishRequest(req) {
        if (!this.connected || !this.ch) {
            throw new Error('RabbitMQ not connected');
        }

        this.ch.sendToQueue('orchestrator.requests', Buffer.from(JSON.stringify(req)), {
            persistent: true,
            contentType: 'application/json',
            correlationId: req.correlationId
        });
    }

    async consumeResponses() {
        if (!this.connected || !this.ch) return;

        try {
            await this.ch.consume('orchestrator.responses', (msg) => {
                if (msg) {
                    const resp = JSON.parse(msg.content.toString());
                    this.responseCache.set(resp.correlationId, resp);
                    console.log(`📬 Response cached: ${resp.correlationId}`);
                    this.ch.ack(msg);
                }
            }, { noAck: false });
        } catch (err) {
            console.error('❌ Failed to consume responses:', err.message);
        }
    }

    getResponse(correlationId) {
        return this.responseCache.get(correlationId);
    }

    clearResponse(correlationId) {
        this.responseCache.delete(correlationId);
    }
}

const rabbitmq = new RabbitMQClient();

const waClient = new Client({
    authStrategy: new LocalAuth({ dataPath: './.wwebjs_auth' }),
    puppeteer: { args: ['--no-sandbox', '--disable-setuid-sandbox'] },
    webVersionCache: {
        type: 'remote',
        remotePath: 'https://raw.githubusercontent.com/wppconnect-team/wa-version/main/html/{version}.html'
    },
    webVersionCache: {
    type: 'remote',
    remotePath: 'https://raw.githubusercontent.com/wppconnect-team/wa-version/main/html/2.2412.54.html'
}
});


waClient.on('qr', (qr) => {
    console.log('⚡ Scan this QR Code with WhatsApp:');
    qrcode.generate(qr, { small: true });
});
waClient.on('message', (msg) => {
    console.log('DEBUG: Message received from:', msg.from, 'Body:', msg.body);
    handleIncomingMessage(msg); // Pass it to your logic
});

waClient.on('ready', async () => {
    console.log('✅ WhatsApp Bot ready!');
    // Connect to RabbitMQ when WhatsApp is ready
    await rabbitmq.connect();
});
let LOCATION = "Unknown Location";

async function fetchLocation() {
    try {
        const { data } = await axios.get("https://ipwho.is/");

        LOCATION = [
            data.city,
            data.region,
            data.country
        ].filter(Boolean).join(", ");

        console.log("📍 Current Location:", LOCATION);
    } catch (err) {
        console.error("Unable to determine location:", err.message);
        LOCATION = "Unknown Location";
    }
}

async function handleIncomingMessage(msg) {
    if (msg.isStatus || msg.from === 'status@broadcast') return;

    // Reject group messages
    if (msg.from.includes("@g.us")) return;

    // Extract phone number from sender JID (format: "919361315379@c.us" -> "919361315379")
    const senderPhone = msg.from.split('@')[0];

    // Validate phone number against persona's allowed list
    if (persona.allowedPhoneNumbers.length > 0) {
        if (!persona.isPhoneAllowed(senderPhone)) {
            console.log(`⛔ Rejected message from unauthorized number: ${senderPhone}`);
            return;
        }
    } else {
        console.log(`⚠️  No phone number validation configured. Accepting from: ${senderPhone}`);
    }

    // Check for trigger prefix (default: "Self" if none configured)
    const triggerPrefix = process.env.TRIGGER_PREFIX || 'Self';
    if (!msg.body.startsWith(triggerPrefix)) return;

    let targetModel = persona.getModelForType(msg.hasMedia);
    let base64Images = [];
    let promptText = msg.body.replace(new RegExp(`^${triggerPrefix}\\s*`), '').trim();

    try {
        if (msg.hasMedia) {
            console.log("📥 Media detected in message - attempting download with retry...");
            let media = null;
            let downloadAttempts = 0;
            const MAX_DOWNLOAD_ATTEMPTS = 3;

            // Retry logic with exponential backoff
            while (!media && downloadAttempts < MAX_DOWNLOAD_ATTEMPTS) {
                downloadAttempts++;
                try {
                    console.log(`   [Attempt ${downloadAttempts}/${MAX_DOWNLOAD_ATTEMPTS}] Downloading media...`);

                    // Add timeout to download operation (30 seconds)
                    const downloadPromise = msg.downloadMedia();
                    const timeoutPromise = new Promise((_, reject) =>
                        setTimeout(() => reject(new Error('Download timeout after 30s')), 30000)
                    );

                    media = await Promise.race([downloadPromise, timeoutPromise]);

                    if (!media) {
                        console.warn(`   ⚠️  Attempt ${downloadAttempts}: Download returned null/undefined`);
                        if (downloadAttempts < MAX_DOWNLOAD_ATTEMPTS) {
                            const backoffMs = Math.pow(2, downloadAttempts - 1) * 1000; // 1s, 2s, 4s
                            console.log(`   ⏳ Retrying in ${backoffMs}ms...`);
                            await new Promise(resolve => setTimeout(resolve, backoffMs));
                        }
                    }
                } catch (mediaError) {
                    // Extract error message from various error types
                    let errorMsg = 'Unknown error';
                    if (mediaError instanceof Error) {
                        errorMsg = mediaError.message;
                    } else if (typeof mediaError === 'string') {
                        errorMsg = mediaError;
                    } else if (mediaError && mediaError.toString) {
                        errorMsg = mediaError.toString();
                    } else {
                        errorMsg = JSON.stringify(mediaError);
                    }
                    console.warn(`   ❌ Attempt ${downloadAttempts}: ${errorMsg}`);
                    if (downloadAttempts < MAX_DOWNLOAD_ATTEMPTS) {
                        const backoffMs = Math.pow(2, downloadAttempts - 1) * 1000;
                        console.log(`   ⏳ Retrying in ${backoffMs}ms...`);
                        await new Promise(resolve => setTimeout(resolve, backoffMs));
                    }
                }
            }

            if (media) {
                const mediaSize = media.data ? media.data.length : 0;
                console.log(`✅ Media downloaded successfully on attempt ${downloadAttempts} - Type: ${media.mimetype}, Size: ${mediaSize} bytes`);

                // Validate media size (max 25MB)
                const MAX_MEDIA_SIZE = 25 * 1024 * 1024;
                if (mediaSize > MAX_MEDIA_SIZE) {
                    console.error(`❌ Media too large: ${mediaSize} bytes (max: ${MAX_MEDIA_SIZE} bytes)`);
                    await msg.reply("🖼️ Image is too large. Please send a smaller file.");
                    return;
                }

                if (media.mimetype && media.mimetype.startsWith('image/')) {
                    if (!media.data) {
                        console.error("❌ Media has no data field (structure error)");
                        await msg.reply("❌ Image download failed (no data).");
                        return;
                    } else {
                        base64Images.push(media.data);
                        targetModel = persona.visionModel;
                        console.log(`✅ Image added to request (${base64Images.length} total images)`);
                        if (!promptText) promptText = "Describe what you see in this image in detail.";
                    }
                } else {
                    console.log(`⚠️  Media is not an image (MIME: ${media.mimetype})`);
                    await msg.reply("ℹ️ Please send an image. Video/audio/documents are not supported yet.");
                    return;
                }
            } else {
                console.error(`❌ Failed to download media after ${MAX_DOWNLOAD_ATTEMPTS} attempts`);
                console.error(`ℹ️ Known issue: whatsapp-web.js #201828 - media download fails with newer WhatsApp Web versions`);
                console.error(`   Workaround: Update whatsapp-web.js or apply patch from https://github.com/wwebjs/whatsapp-web.js/issues/201828`);
                await msg.reply("❌ Failed to download image.\n\n📝 This is a known issue with whatsapp-web.js library.\n\n🔧 Try:\n1. Update the library: npm update whatsapp-web.js\n2. Restart the bot\n3. Try sending the image again");
                return;
            }
        }

        if (!promptText && base64Images.length === 0) {
            console.log("⚠️  No text prompt and no images, skipping request");
            return;
        }

        // Generate unique correlation ID for request-response tracking
        const correlationId = `req_${Date.now()}_${crypto.randomBytes(4).toString('hex')}`;
        const idempotencyKey = `msg_${senderPhone}_${Date.now()}`;

        // Construct queue request
        const queueRequest = {
            correlationId,
            phoneNumber: senderPhone,
            message: promptText,
            timestamp: Date.now(),
            idempotencyKey,
            images: base64Images
        };

        console.log(`📤 Enqueuing request - ID: ${correlationId}, Text: ${promptText.length} chars, Images: ${base64Images.length}`);

        try {
            // Enqueue the request (non-blocking)
            await rabbitmq.publishRequest(queueRequest);

            // Acknowledge immediately to user
            console.log(`✓ Request enqueued: ${correlationId}`);

            // Poll for response with timeout (15 minutes max wait)
            const startTime = Date.now();
            const maxWaitMs = 15 * 60 * 1000;
            const pollIntervalMs = 500;

            const pollResponse = setInterval(async () => {
                const elapsed = Date.now() - startTime;
                const response = rabbitmq.getResponse(correlationId);

                if (response) {
                    clearInterval(pollResponse);
                    rabbitmq.clearResponse(correlationId);

                    if (response.status === 'success') {
                        await msg.reply(response.result);
                        console.log(`✨ Sent response to ${senderPhone} (${response.processingTimeMs}ms processing)`);
                    } else {
                        await msg.reply(`❌ Error: ${response.error}`);
                        console.log(`⚠️  Error response for ${correlationId}: ${response.error}`);
                    }
                } else if (elapsed > maxWaitMs) {
                    clearInterval(pollResponse);
                    await msg.reply(`⏱️ Request timed out after 15 minutes. Please try again.`);
                    console.log(`❌ Response timeout for ${correlationId}`);
                }
            }, pollIntervalMs);
        } catch (err) {
            console.error("❌ Failed to enqueue request:", err.message);
            await msg.reply("⚠️ Failed to queue your request. Please try again.");
        }

    } catch (error) {
        console.error("❌ Error:", error);
        await msg.reply("Sorry, I encountered an error.");
    }
}

waClient.on("message_create", handleIncomingMessage);
(async () => {
   // await fetchLocation();
    waClient.initialize();
})();