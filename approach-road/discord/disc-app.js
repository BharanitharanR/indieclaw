require("dotenv").config();
const { Client, GatewayIntentBits } = require('discord.js');
const axios = require('axios'); // Still used for image downloading & location
const grpcClient = require('./grpcClent'); // Import the gRPC client

const DISCORD_TOKEN = process.env.DISCORD_TOKEN;
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

    // 1. Triggers
    const isMention = message.mentions.has(client.user);
    const startsWithJambu = message.content.trim().startsWith("Jambu::");
    const hasImage = message.attachments.size > 0;

    if (!isMention && !startsWithJambu && !hasImage) return;

    // 2. Prepare Prompt
    let promptText = message.content
        .replace(/^Jambu::\s*/i, '')
        .replace(new RegExp(`<@!?${client.user.id}>\\s*`, 'i'), '')
        .trim();

    // 3. Process Images
    let base64Images = [];
    if (hasImage) {
        for (const [id, attachment] of message.attachments) {
            if (attachment.contentType?.startsWith('image/')) {
                try {
                    const response = await axios.get(attachment.url, { responseType: 'arraybuffer' });
                    base64Images.push(Buffer.from(response.data, 'binary').toString('base64'));
                } catch (err) { console.error("Image download failed:", err); }
            }
        }
    }

    if (!promptText && hasImage) promptText = "Describe this image in detail.";
    if (!promptText && !hasImage) return;

    const location = await getCurrentLocation();
    const finalPrompt = `Current Location: ${location}. \nInstruction: ${promptText}`;

    // 4. Send via gRPC
    await message.channel.sendTyping();
    
    const request = {
        messages: [{ 
            role: "user", 
            content: finalPrompt, 
            images: base64Images 
        }]
    };

    grpcClient.chat(request, async (err, response) => {
        if (err) {
            console.error("gRPC Error:", err);
            await message.reply("⚠️ Error communicating with the backend.");
        } else {
            
           //  await message.reply(response.message?.content || "No reply.");
            await sendLongMessage(message, response.message?.content || "No reply.");
        }
    });
});

async function loginWithRetry(retries = 5, delayMs = 5000) {
    for (let i = 0; i < retries; i++) {
        try {
            await client.login(DISCORD_TOKEN);
            return;
        } catch (err) {
            console.error(`Login failed: ${err.message}`);
            await new Promise(r => setTimeout(r, delayMs));
        }
    }
    process.exit(1);
}
async function sendLongMessage(message, content) {
    if (content.length <= 2000) {
        return message.reply(content);
    }

    // Split into chunks of 1900 to ensure we don't hit the limit
    const chunks = content.match(/.{1,1900}(\n|$)/gs) || [content];
    
    for (const chunk of chunks) {
        await message.channel.send(chunk);
    }
}
loginWithRetry();