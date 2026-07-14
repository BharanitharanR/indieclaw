package main

import (
	"log"
	"net/http"

	"gateway-service/internal/service"
)

func main() {
	ollamaURL := "http://127.0.0.1:11434"

	// Default to gemma4 (or qwen2.5)
	defaultModel := "gemma4:12b"

	ollamaSvc := service.NewOllamaService(ollamaURL, defaultModel)

	mux := http.NewServeMux()
	ollamaSvc.RegisterRoutes(mux)

	log.Println("Gateway running on http://127.0.0.1:8080")
	if err := http.ListenAndServe("127.0.0.1:8080", mux); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
