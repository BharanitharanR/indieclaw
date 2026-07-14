package chat

type ChatMessage struct {
	Role    string   `json:"role"`             // "user", "assistant", "system"
	Content string   `json:"content"`          // Text prompt or caption
	Images  []string `json:"images,omitempty"` // Base64 strings for Vision models (Gemma 4 / Qwen-VL)
}

type ChatRequest struct {
	Model    string        `json:"model"` // e.g., "gemma4:12b", "qwen2.5:7b"
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"` // Set to false for standard HTTP response
}

type ChatResponse struct {
	Model     string      `json:"model"`
	CreatedAt string      `json:"created_at"`
	Message   ChatMessage `json:"message"`
	Done      bool        `json:"done"`
}
