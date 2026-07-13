const { Client, LocalAuth } = require("whatsapp-web.js");
const qrcode = require("qrcode-terminal");

const RustAIClient = require("../sdk/rust_ai.js");
const ai = new RustAIClient();

const client = new Client({
    authStrategy: new LocalAuth()
});

client.on("qr", qr => {
    console.log("Scan QR");
    qrcode.generate(qr, { small: true });
});

client.on("ready", () => {
    console.log("WhatsApp Ready");
});

async function handleIncomingMessage(msg) {
    if (msg.from.includes("@g.us")) return;
    if (msg.isStatus) return;

    // Read message body or fallback to image caption text
    let promptText = msg.body || msg.caption || "";
    
    // Explicit security whitelist restriction validation 
    if (!msg.from.includes("919361315379@c.us")) return;
    if (!promptText.startsWith("Jambu::")) return;

    console.log(`Processing valid user query from: ${msg.from}`);

    // Clean trigger prefix so the model doesn't get confused by "Jambu::"
    promptText = promptText.replace("Jambu::", "").trim();
    
    // If no specific text prompt remains alongside media, use a definitive instruction
    if (promptText === "" && msg.hasMedia) {
        promptText = "Describe this image in detail.";
    }

    try {
        const userMessage = {
            role: "user",
            content: promptText
        };

        if (msg.hasMedia) {
            const media = await msg.downloadMedia();
            if (media && media.mimetype.startsWith("image/")) {
                userMessage.image = `data:${media.mimetype};base64,${media.data}`;
                console.log("Image media successfully attached to active payload structure.");
            }
        }

        // Invoke client without hardcoded vision parameters — SDK resolves routing rules internally
        const response = await ai.chat([userMessage]);

        let answer = "";
        if (response.message) {
            answer = response.message.content;
        } else if (response.response) {
            answer = response.response;
        } else {
            answer = JSON.stringify(response);
        }

        await msg.reply(answer);

    } catch (err) {
        console.error("Processing Fail:", err);
        await msg.reply("Rust Gateway unavailable.");
    }
}

client.on("message_create", handleIncomingMessage);
client.on("message", handleIncomingMessage);

client.initialize();
