require("dotenv").config();

const DISCORD_TOKEN = process.env.DISCORD_TOKEN;
const { Client, GatewayIntentBits } = require('discord.js');
const axios = require('axios');

// Configuration
const GO_GATEWAY_URL = 'http://127.0.0.1:8080/api/v1/chat';
const TEXT_MODEL = 'qwen3:8b';


// Initialize Discord Client with required Intents
const client = new Client({
    intents: [
        GatewayIntentBits.Guilds,
        GatewayIntentBits.GuildMessages,
        GatewayIntentBits.MessageContent
    ]
});

// Helper to fetch location
async function getCurrentLocation() {
    try {
        const response = await axios.get('http://ip-api.com/json/');
        return `${response.data.city}, ${response.data.regionName}, ${response.data.country}`;
    } catch (e) {
        return "Unknown Location";
    }
}

client.once('ready', () => {
    console.log(`✅ Logged in as ${client.user.tag}!`);
});
client.on('messageCreate', async (message) => {
    if (message.author.bot) return;

    // 1. Check if we should ignore this message entirely
    // We only process if it starts with Jambu:: OR if it has an image
    const hasTrigger = message.content.trim().startsWith("Jambu::");
    const hasImage = message.attachments.size > 0;

    if (!hasTrigger && !hasImage) return;

    console.log(`✅ Message accepted. Trigger: ${hasTrigger}, Images: ${message.attachments.size}`);

    // 2. Extract prompt (remove Jambu:: if it exists)
    let promptText = message.content.replace(/^Jambu::\s*/i, '').trim();
    
    // 3. Process Images
    let base64Images = [];
    if (hasImage) {
        for (const [id, attachment] of message.attachments) {
            if (attachment.contentType?.startsWith('image/')) {
                try {
                    console.log(`Downloading: ${attachment.url}`);
                    const response = await axios.get(attachment.url, { responseType: 'arraybuffer' });
                    base64Images.push(Buffer.from(response.data, 'binary').toString('base64'));
                } catch (err) {
                    console.error("Download failed:", err);
                }
            }
        }
    }

    // 4. Default Prompting
    if (!promptText && hasImage) {
        promptText = "Describe this image in detail.";
    } else if (!promptText && !hasImage) {
        return; // Nothing to do
    }

    // 5. Send to Gateway
    await message.channel.sendTyping();
    const payload = {
        model: base64Images.length > 0 ? "gemma4:e2b" : "qwen3:8b",
        messages: [{ role: "user", content: `Instruction: ${promptText}`, images: base64Images }],
        stream: false
    };

    try {
        const response = await axios.post(GO_GATEWAY_URL, payload, { timeout: 180000 });
        base64Images = [];
        await message.reply(response.data?.message?.content || "No reply.");
    } catch (error) {
        console.error("Gateway Error:", error.message);
        base64Images = [];
        await message.reply("Error processing your request.");
    }
});

client.login(DISCORD_TOKEN);