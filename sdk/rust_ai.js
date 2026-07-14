class RustAIClient {

    constructor(baseUrl = "http://localhost:8080") {
        this.baseUrl = baseUrl;
    }

    async chat(messages, options = {}) {

        const payload = {
            model: options.model || "qwen3:8b",
            messages,
            stream: false,
            options: {
                num_ctx: options.context ?? 40960,
                think: options.think ?? true
            }
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
            throw new Error(
                `Gateway returned ${response.status}`
            );
        }

        return await response.json();
    }

}

module.exports = RustAIClient;