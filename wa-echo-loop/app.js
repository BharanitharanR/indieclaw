const { Client, LocalAuth } = require("whatsapp-web.js");
const qrcode = require("qrcode-terminal");

const RustAIClient = require("../sdk/rust_ai.js");
const ALLOWED_NUMBER = "213103828537359@lid";
const ai = new RustAIClient();

const client = new Client({
    authStrategy: new LocalAuth()
});

client.on("qr", qr => {

    console.log("Scan QR");

    qrcode.generate(qr, {
        small: true
    });

});

client.on("ready", () => {

    console.log("WhatsApp Ready");

});

client.on("authenticated", () => {
    console.log("Authenticated");
});

client.on("auth_failure", msg => {
    console.log("Auth failure:", msg);
});

client.on("disconnected", reason => {
    console.log("Disconnected:", reason);
});

client.on("change_state", state => {
    console.log("State:", state);
});

client.on("loading_screen", (percent, message) => {
    console.log(percent, message);
});

client.on("message", async msg => {

    console.log("Received message:", msg.from);
    console.log("Received BODY:", msg.body);
    // Ignore everyone except this number
    if (msg.from !== ALLOWED_NUMBER)
        return;

    // Ignore groups
    if (msg.from.includes("@g.us"))
        return;

    // Ignore status updates
    if (msg.isStatus)
        return;

    console.log("User:", msg.body);

    try {

        const response = await ai.chat([
            {
                role: "user",
                content: msg.body
            }
        ]);

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

        console.error(err);

        await msg.reply("Rust Gateway unavailable.");

    }

});

client.initialize();