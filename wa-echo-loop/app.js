const { Client, LocalAuth } = require('whatsapp-web.js');
const qrcode = require('qrcode-terminal');
const axios = require('axios');

// Go Gateway Service URL
const GO_GATEWAY_URL = 'http://127.0.0.1:8080/api/v1/chat';

// Model definitions
const TEXT_MODEL = 'qwen3:8b';
const VISION_MODEL = 'gemma4:e2b';

// Initialize WhatsApp Client with local session saving
const client = new Client({
    authStrategy: new LocalAuth({ dataPath: './.wwebjs_auth' }),
    puppeteer: {
        args: ['--no-sandbox', '--disable-setuid-sandbox']
    }
});

// Display QR Code in Terminal for setup
client.on('qr', (qr) => {
    console.log('⚡ Scan this QR Code with WhatsApp on your phone:');
    qrcode.generate(qr, { small: true });
});

// Log readiness state
client.on('ready', () => {
    console.log('✅ WhatsApp Bot is connected and ready!');
    console.log(`🤖 Default Text Model: ${TEXT_MODEL}`);
    console.log(`👁️ Default Vision Model: ${VISION_MODEL}`);
});

// Main Message Event Handler
async function handleIncomingMessage(msg) {
    // Ignore status updates, group notifications, or broadcasts
    if (msg.isStatus || msg.from === 'status@broadcast') return;

    let targetModel = TEXT_MODEL;
    let base64Images = [];
    let promptText = msg.body ? msg.body.trim() : '';
    console.log(`message received from ${msg.from}. `);
     if(msg.from.includes("@g.us")){
            
            return; // Ignore group messages
        }

        if(!msg.body.startsWith("Jambu::")){
            return; // Ignore group messages    
        }

        if(!msg.from.includes("919361315379@c.us")){
            return; // Ignore group messages    
        }
    try {
        // Check if message contains an image media attachment
        if (msg.hasMedia) {
            const media = await msg.downloadMedia();

            // Only process image media
            if (media && media.mimetype.startsWith('image/')) {
                base64Images.push(media.data); // Pure base64 string
                targetModel = VISION_MODEL;    // Switch to Gemma 4 for vision

                if (!promptText) {
                    promptText = "Describe what you see in this image in detail.";
                }

                console.log(`📸 Image received from ${msg.from}. Forwarding to ${VISION_MODEL}...`);
            }
        } else {
            console.log(`💬 Text message received from ${msg.from}. Forwarding to ${TEXT_MODEL}...`);
        }


        // Do not process empty messages (e.g., non-image media like audio/documents)
        if (!promptText && base64Images.length === 0) return;

        // Construct Ollama API Chat request body expected by Go Gateway
        const payload = {
            model: targetModel,
            messages: [
                {
                    role: "user",
                    content: promptText,
                    images: base64Images
                }
            ],
            stream: false
        };

        // Call your Go Gateway endpoint
        const response = await axios.post(GO_GATEWAY_URL, payload, {
            headers: { 'Content-Type': 'application/json' },
            timeout: 180000 // 3 minute timeout for model generation
        });

        // Extract model reply
        if (response.data && response.data.message && response.data.message.content) {
            const replyText = response.data.message.content;
            await msg.reply(replyText);
            console.log(`✨ Replied to ${msg.from} via ${targetModel}`);
        } else {
            await msg.reply("⚠️ Received empty response from model gateway.");
        }

    } catch (error) {
        console.error(`❌ Error processing message from ${msg.from}:`, error.message);
        
        if (error.response) {
            console.error('Gateway Error Status:', error.response.status);
            console.error('Gateway Error Data:', error.response.data);
        }

        await msg.reply("Sorry, I encountered an error trying to process your request.");
    }
};
client.on("message_create", handleIncomingMessage);
// client.on("message", handleIncomingMessage);
// Start WhatsApp Client
client.initialize();