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
