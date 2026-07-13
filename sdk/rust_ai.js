const fs = require('fs');

class RustAIClient {

    constructor(baseUrl = "http://localhost:8080") {
        this.baseUrl = baseUrl;
        // Native Ollama endpoint fallback to bypass proxy for vision matrix layers
        this.ollamaUrl = "http://localhost:11434"; 
    }

    /**
     * Extracts pure Base64 strings from either file paths or WhatsApp media URIs.
     */
    _encodeImage(imageInput) {
        if (imageInput.startsWith('data:')) {
            return imageInput.replace(/^data:image\/\w+;base64,/, "");
        }
        
        if (!fs.existsSync(imageInput)) {
            throw new Error(`Image file not found at path: ${imageInput}`);
        }
        
        const fileBuffer = fs.readFileSync(imageInput);
        return fileBuffer.toString('base64');
    }

    async chat(messages, options = {}) {
        let hasImage = false;

        // Process message payloads
        const processedMessages = messages.map(msg => {
            if (msg.image) {
                hasImage = true;
                const base64Str = this._encodeImage(msg.image);
                return {
                    role: msg.role,
                    content: msg.content,
                    images: [base64Str] // Native Ollama specification array
                };
            }
            return msg;
        });

        let targetUrl;
        let selectedModel;
        let chatOptions;

        if (hasImage) {
            // Direct to Ollama to ensure vision parameters aren't stripped
            targetUrl = `${this.ollamaUrl}/api/chat`;
            selectedModel = "gemma4:e2b"; // Enforce gemma for all media/vision tasks
            chatOptions = {
                num_ctx: options.context ?? 8192,
                think: false // Disabled thinking steps to avoid gemma-vision processing loops
            };
        } else {
            // Standard Text: Route through your Rust proxy gateway flow
            targetUrl = `${this.baseUrl}/api/v1/chat`;
            selectedModel = options.model || "qwen3:8b"; // Fallback to your primary text model
            chatOptions = {
                num_ctx: options.context ?? 40960,
                think: options.think ?? true // Retain tool-use thinking features for text agents
            };
        }

        const payload = {
            model: selectedModel,
            messages: processedMessages,
            stream: false,
            options: chatOptions
        };

        const response = await fetch(
            targetUrl,
            {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify(payload)
            }
        );

        if (!response.ok) {
            throw new Error(`Target endpoint returned status: ${response.status}`);
        }

        const data = await response.json();

        // Standardize output response schema blocks
        if (hasImage) {
            return {
                message: {
                    content: data.message?.content || data.response || JSON.stringify(data)
                }
            };
        }

        return data;
    }
}

module.exports = RustAIClient;
