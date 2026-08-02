const { Client, LocalAuth } = require('whatsapp-web.js');
const qrcode = require('qrcode-terminal');
const client = require('./grpcClent'); // Import the gRPC client created in the previous step
const axios = require('axios'); // Still used for image downloading & location
// Model definitions
const TEXT_MODEL = 'qwen3:8b';
const VISION_MODEL = 'gemma4:e2b';

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

waClient.on('ready', () => console.log('✅ WhatsApp Bot ready!'));
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
    if (msg.isStatus || msg.from === 'status@broadcast' )  return;
    if (msg.from.includes("@g.us") || !msg.body.startsWith("Self") || !msg.from.includes("919361315379@c.us")) return;

    let targetModel = TEXT_MODEL;
    let base64Images = [];
    let promptText = msg.body.replace(/^Self\s*/, '').trim();

    try {
        if (msg.hasMedia) {
            const media = await msg.downloadMedia();
            if (media && media.mimetype.startsWith('image/')) {
                base64Images.push(media.data);
                targetModel = VISION_MODEL;
                if (!promptText) promptText = "Describe what you see in this image in detail.";
            }
        }

        if (!promptText && base64Images.length === 0) return;

        // Contextual Prompting

        const location = LOCATION;
        const finalPrompt = `Current Location: ${location}. \nInstruction: ${promptText}`;
        console.log(finalPrompt)
        // Construct gRPC Request
        const request = {
            messages: [{
                role: "user",
                content: finalPrompt,
                images: base64Images
            }]
        };

        // gRPC Call
        client.chat(request, async (err, response) => {
            if (err) {
                console.error("❌ gRPC Error:", err);
                await msg.reply("⚠️ Error communicating with the backend.");
                return;
            }
            
            if (response.message && response.message.content) {
                await msg.reply(response.message.content);
                console.log(`✨ Replied via ${targetModel}`);
            }
        });

    } catch (error) {
        console.error("❌ Error:", error);
        await msg.reply("Sorry, I encountered an error.");
    }
}

waClient.on("message_create", handleIncomingMessage);
(async () => {
    await fetchLocation();
    waClient.initialize();
})();