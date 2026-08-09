const { Client, LocalAuth } = require('whatsapp-web.js');
const qrcode = require('qrcode-terminal');
const axios = require('axios');
const crypto = require('crypto');
const PersonaConfig = require('./personaConfig');
const WhitelistManager = require('./whitelist');

// Load persona configuration
const personaName = process.env.PERSONA_NAME || 'default';
const persona = PersonaConfig.loadByName(personaName);
console.log(`📝 ${persona.toString()}`);

// Initialize whitelist manager
const whitelistConfig = persona.getWhitelistConfig();
const whitelist = new WhitelistManager(whitelistConfig);
console.log(`\n[Whitelist] Status: ${JSON.stringify(whitelist.getStatus(), null, 2)}\n`);

// ============= OLLAMA CLIENT (Direct LLM) =============
class OllamaClient {
    constructor() {
        this.baseUrl = process.env.OLLAMA_URL || 'http://localhost:11434';
        this.model = process.env.LLM_MODEL || 'qwen3:8b-16k';
        this.connected = false;
    }

    async generateResponse(prompt, systemPrompt = '') {
        try {
            const response = await axios.post(`${this.baseUrl}/api/generate`, {
                model: this.model,
                prompt: prompt,
                system: systemPrompt,
                stream: false,
                temperature: 0.7,
                top_p: 0.95,
            }, {
                timeout: 60000
            });

            if (response.data.response) {
                return response.data.response.trim();
            } else {
                throw new Error('Empty response from LLM');
            }
        } catch (err) {
            console.error(`❌ LLM Error: ${err.message}`);
            throw err;
        }
    }

    async coachingResponse(userMessage, userContext = {}) {
        const systemPrompt = `You are an Executive Coach specializing in logical thinking and decision-making.

COACHING RULES:
1. Respond warmly and personally (not as a consultant)
2. Provide 2-3 specific, actionable steps (not frameworks)
3. Ask ONE probing question at the end
4. Connect advice to their goals
5. Ignore generic frameworks - find novel insights

${userContext.previousSessions ? `User's Background: ${userContext.previousSessions} previous coaching sessions` : ''}
${userContext.focus ? `Their Focus: ${userContext.focus}` : ''}
${userContext.level ? `Their Level: ${userContext.level}` : ''}`;

        try {
            const response = await this.generateResponse(userMessage, systemPrompt);
            return response;
        } catch (err) {
            return `I'd like to help, but I'm having trouble connecting to my reasoning right now. Could you try again in a moment?`;
        }
    }

    async isConnected() {
        try {
            const response = await axios.get(`${this.baseUrl}/api/tags`, { timeout: 5000 });
            this.connected = !!response.data.models;
            return this.connected;
        } catch (err) {
            this.connected = false;
            return false;
        }
    }
}

const ollama = new OllamaClient();

// ============= QDRANT CLIENT (Context Storage) =============
class QdrantClient {
    constructor() {
        this.baseUrl = process.env.QDRANT_URL || 'http://localhost:6333';
        this.collection = 'coaching_history';
    }

    async storeMessage(userId, userMessage, response, metadata = {}) {
        try {
            // Generate simple embedding (or use external service)
            const embedding = this.hashToVector(userMessage);

            await axios.post(`${this.baseUrl}/collections/${this.collection}/points`, {
                points: [{
                    id: crypto.randomUUID(),
                    vector: embedding,
                    payload: {
                        user_id: userId,
                        user_message: userMessage,
                        response: response,
                        timestamp: new Date().toISOString(),
                        ...metadata
                    }
                }]
            });

            console.log(`💾 Stored coaching interaction for ${userId}`);
        } catch (err) {
            console.warn(`⚠️  Failed to store in Qdrant: ${err.message}`);
        }
    }

    async getContext(userId, limit = 3) {
        try {
            // Simplified retrieval - in production use proper semantic search
            const response = await axios.post(`${this.baseUrl}/collections/${this.collection}/points/search`, {
                vector: [0.1, 0.2, 0.3],  // Placeholder
                limit: limit,
                filter: {
                    must: [{
                        key: 'user_id',
                        match: { value: userId }
                    }]
                }
            });

            return response.data.result || [];
        } catch (err) {
            console.warn(`⚠️  Failed to retrieve context: ${err.message}`);
            return [];
        }
    }

    hashToVector(text) {
        // Simple hash to vector for demo (use real embeddings in production)
        const hash = crypto.createHash('sha256').update(text).digest('hex');
        const vector = [];
        for (let i = 0; i < 384; i++) {
            vector.push(parseFloat('0.' + hash.substr(i * 2, 2)) / 256);
        }
        return vector;
    }
}

const qdrant = new QdrantClient();

const waClient = new Client({
    authStrategy: new LocalAuth({ dataPath: './.wwebjs_auth_fresh' }),
    puppeteer: { args: ['--no-sandbox', '--disable-setuid-sandbox'] },
    webVersionCache: {
        type: 'remote',
        remotePath: 'https://raw.githubusercontent.com/wppconnect-team/wa-version/main/html/2.2412.54.html'
    }
});

let currentQRText = null;

waClient.on('qr', (qr) => {
    currentQRText = qr;
    console.log('⚡ QR Code updated - open http://localhost:8002 to scan');
});

waClient.on('message', (msg) => {
    console.log('DEBUG: Message received from:', msg.from, 'Body:', msg.body);
    handleIncomingMessage(msg);
});

waClient.on('ready', async () => {
    console.log('✅ WhatsApp Bot ready!');
    const ollmaConnected = await ollama.isConnected();
    if (!ollmaConnected) {
        console.warn('⚠️  WARNING: Ollama not accessible at', ollama.baseUrl);
        console.warn('   Make sure Ollama is running: ollama serve');
    } else {
        console.log(`✅ Connected to Ollama (Model: ${ollama.model})`);
    }
});

let LOCATION = "Unknown Location";

// ============= REGISTRATION FILE MANAGEMENT =============
const fs = require('fs');
const path = require('path');

const registrationFile = path.join(__dirname, 'registered_users.txt');

function saveToRegistrationFile(contactName, lid) {
    try {
        const entry = `${contactName}|${lid}|${new Date().toISOString()}\n`;
        fs.appendFileSync(registrationFile, entry);
        console.log(`💾 Saved to registration file: ${contactName}`);
    } catch (err) {
        console.error(`❌ Failed to save to registration file: ${err.message}`);
    }
}

function removeFromRegistrationFile(contactName) {
    try {
        if (!fs.existsSync(registrationFile)) return;

        const lines = fs.readFileSync(registrationFile, 'utf-8').split('\n');
        const filtered = lines.filter(line => !line.startsWith(contactName + '|'));
        fs.writeFileSync(registrationFile, filtered.join('\n'));
        console.log(`🗑️  Removed from registration file: ${contactName}`);
    } catch (err) {
        console.error(`❌ Failed to remove from registration file: ${err.message}`);
    }
}

async function fetchLocation() {
    try {
        const { data } = await axios.get("https://ipwho.is/");
        LOCATION = [data.city, data.region, data.country].filter(Boolean).join(", ");
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

    let contact;
    let contactName = null;
    try {
        contact = await msg.getContact();
        contactName = contact.pushname || contact.name || contact.number;
        console.log(`📩 Message from: ${contactName} (${contact.number})`);
    } catch (err) {
        console.warn(`⚠️  Could not retrieve contact info: ${err.message}`);
        return;
    }

    const lid = msg.from.split('@')[0];  // Line ID from WhatsApp

    // ============= REGISTRATION =============
    if (msg.body.toLowerCase().includes("register me")) {
        const result = whitelist.add(contactName);
        if (result.success) {
            // Also save to static file: contact_name | lid
            saveToRegistrationFile(contactName, lid);
            await msg.reply(`✅ Registered: ${contactName}`);
            console.log(`[Registration] ✅ Registered ${contactName} (LID: ${lid})`);
        } else {
            await msg.reply(`⚠️  ${result.message}`);
        }
        return;
    }

    // ============= UNREGISTRATION =============
    if (msg.body.toLowerCase().includes("unregister me")) {
        const result = whitelist.remove(contactName);
        if (result.success) {
            // Also remove from static file
            removeFromRegistrationFile(contactName);
            await msg.reply(`✅ Unregistered: ${contactName}`);
            console.log(`[Unregistration] ✅ Unregistered ${contactName}`);
        } else {
            await msg.reply(`⚠️  ${result.message}`);
        }
        return;
    }

    // ============= WHITELIST CHECK =============
    const isWhitelisted = whitelist.isWhitelisted(contactName);
    whitelist.logRoutingDecision(contactName, isWhitelisted, isWhitelisted ? 'coaching' : 'unregistered');

    // If not registered, ignore message
    if (!isWhitelisted) {
        console.log(`[Whitelist] ℹ️  Message from unregistered user ${contactName}, ignoring`);
        return;
    }

    // ============= DIRECT LLM COACHING =============
    console.log(`⏱️  Starting direct LLM coaching response...`);
    const startTime = Date.now();

    try {
        // Acknowledge receipt
        await msg.react('👀');

        // Get user context from Qdrant
        const userContext = await qdrant.getContext(contactName, 3);
        const contextSummary = userContext.length > 0
            ? `Previous topics: ${userContext.map(c => c.payload?.user_message).slice(0, 2).join(', ')}`
            : 'First interaction';

        console.log(`📚 User context: ${contextSummary}`);

        // Generate coaching response directly from LLM
        const coachingResponse = await ollama.coachingResponse(msg.body, {
            focus: 'executive coaching',
            level: 'intermediate',
            previousSessions: userContext.length
        });

        // Send response (split into chunks if needed)
        const maxLength = 4096;
        if (coachingResponse.length > maxLength) {
            const chunks = [];
            let currentChunk = '';

            const sentences = coachingResponse.match(/[^.!?]+[.!?]+/g) || [coachingResponse];

            for (const sentence of sentences) {
                if ((currentChunk + sentence).length > maxLength) {
                    chunks.push(currentChunk.trim());
                    currentChunk = sentence;
                } else {
                    currentChunk += sentence;
                }
            }
            if (currentChunk.trim()) chunks.push(currentChunk.trim());

            // Send chunks
            for (let i = 0; i < chunks.length; i++) {
                await msg.reply(chunks[i]);
                if (i < chunks.length - 1) {
                    await new Promise(resolve => setTimeout(resolve, 500));  // Rate limit
                }
            }
        } else {
            await msg.reply(coachingResponse);
        }

        // Store interaction in Qdrant
        await qdrant.storeMessage(contactName, msg.body, coachingResponse, {
            responseTime: Date.now() - startTime,
            messageType: 'coaching'
        });

        const responseTime = ((Date.now() - startTime) / 1000).toFixed(1);
        console.log(`✅ Response sent in ${responseTime}s`);
        await msg.react('✅');

    } catch (err) {
        console.error(`❌ Error handling message: ${err.message}`);
        await msg.reply(`Sorry, I encountered an error: ${err.message}. Please try again.`);
        await msg.react('❌');
    }
}

waClient.on("message_create", handleIncomingMessage);

// ============= HTTP SERVERS =============
const http = require('http');

// QR Code UI Server (Port 8002)
const qrServer = http.createServer((req, res) => {
    if (req.url === '/') {
        const html = `
<!DOCTYPE html>
<html>
<head>
    <title>Adiyan - WhatsApp QR Code</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .container {
            background: white;
            padding: 40px;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            text-align: center;
        }
        h1 {
            margin: 0 0 10px 0;
            color: #333;
        }
        .status {
            font-size: 14px;
            color: #666;
            margin-bottom: 30px;
        }
        .qr-container {
            background: #f5f5f5;
            padding: 20px;
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 300px;
            min-width: 300px;
        }
        .qr-container img {
            max-width: 100%;
            height: auto;
        }
        .qr-container.empty {
            color: #999;
            font-size: 18px;
        }
        .refresh {
            margin-top: 20px;
            font-size: 12px;
            color: #999;
        }
        .ready {
            color: #4CAF50;
            font-weight: bold;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎯 Adiyan WhatsApp Bot</h1>
        <div class="status">Scan to connect WhatsApp</div>
        <div id="qr-container" class="qr-container empty">
            Waiting for QR code...
        </div>
        <div id="ready-status"></div>
        <div class="refresh">Page auto-refreshes every 5 seconds</div>
    </div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js"></script>
    <script>
        let qrInstance = null;

        async function updateQR() {
            try {
                const response = await fetch('/api/qr');
                if (!response.ok) throw new Error('Failed to fetch QR');

                const data = await response.json();
                const container = document.getElementById('qr-container');

                if (data.qrText) {
                    container.innerHTML = '';
                    container.classList.remove('empty');

                    // Generate QR code using QRCode.js library
                    qrInstance = new QRCode(container, {
                        text: data.qrText,
                        width: 256,
                        height: 256,
                        colorDark: '#000000',
                        colorLight: '#ffffff',
                        correctLevel: QRCode.CorrectLevel.H
                    });
                } else {
                    container.innerHTML = 'Waiting for QR code...';
                    container.classList.add('empty');
                }

                const statusDiv = document.getElementById('ready-status');
                if (data.whatsappReady) {
                    statusDiv.innerHTML = '<div class="ready">✅ WhatsApp Connected!</div>';
                } else {
                    statusDiv.innerHTML = '';
                }
            } catch (err) {
                console.error('Failed to update QR:', err);
                const container = document.getElementById('qr-container');
                container.innerHTML = 'Error loading QR code';
                container.classList.add('empty');
            }
        }

        updateQR();
        setInterval(updateQR, 5000);
    </script>
</body>
</html>`;
        res.writeHead(200, { 'Content-Type': 'text/html' });
        res.end(html);
        return;
    }

    // API endpoint for QR code
    if (req.url === '/api/qr') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({
            qrText: currentQRText || null,
            whatsappReady: waClient.info ? true : false
        }));
        return;
    }

    res.writeHead(404);
    res.end('Not found');
});

// Admin API Server (Port 8003)
const adminServer = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.setHeader('Access-Control-Allow-Origin', '*');
    res.setHeader('Access-Control-Allow-Methods', 'GET, POST, DELETE, OPTIONS');
    res.setHeader('Access-Control-Allow-Headers', 'Content-Type');

    if (req.method === 'OPTIONS') {
        res.writeHead(200);
        res.end();
        return;
    }

    const url = new URL(req.url, `http://${req.headers.host}`);
    const pathname = url.pathname;

    // GET /admin/whitelist
    if (req.method === 'GET' && pathname === '/admin/whitelist') {
        const status = whitelist.getStatus();
        res.writeHead(200);
        res.end(JSON.stringify(status, null, 2));
        return;
    }

    // GET /admin/health
    if (req.method === 'GET' && pathname === '/admin/health') {
        res.writeHead(200);
        res.end(JSON.stringify({
            status: 'ok',
            whitelist_count: whitelist.getCount(),
            whatsapp_ready: waClient.info ? true : false,
            ollama_connected: ollama.connected,
        }, null, 2));
        return;
    }

    res.writeHead(404);
    res.end(JSON.stringify({ error: 'Not Found' }, null, 2));
});

const qrPort = process.env.QR_PORT || 8002;
const adminPort = process.env.ADMIN_PORT || 8003;

qrServer.listen(qrPort, () => {
    console.log(`\n📱 QR Code UI: http://localhost:${qrPort}`);
});

adminServer.listen(adminPort, () => {
    console.log(`🔧 Admin API: http://localhost:${adminPort}/admin/health\n`);
});

(async () => {
    await fetchLocation();
    waClient.initialize();
})();
