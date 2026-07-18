package vision

import (
	"context"
	"strings"

	"github.com/Protocol-Lattice/go-agent/src/models"
	ollama "github.com/ollama/ollama/api" // <- correct import
	// Import the package where OllamaLLM is defined
)

// VisionFixedLLM wraps the original OllamaLLM
type VisionFixedLLM struct {
	*models.OllamaLLM // Embedding the original struct
}

// GenerateWithFiles overrides the method to fix the bug
func (v *VisionFixedLLM) GenerateWithFiles(ctx context.Context, prompt string, files []models.File) (any, error) {
	// 1. Manually prepare the Ollama request
	var images []ollama.ImageData
	for _, f := range files {
		// Just append the bytes. The SDK will handle the base64 conversion correctly
		// if you send the bytes directly.
		images = append(images, ollama.ImageData(f.Data))
	}

	req := &ollama.GenerateRequest{
		Model:  v.Model,
		Prompt: prompt,
		Images: images,
	}

	// 2. Call the Ollama client directly, bypassing the broken library method
	var response strings.Builder
	err := v.Client.Generate(ctx, req, func(gr ollama.GenerateResponse) error {
		response.WriteString(gr.Response)
		return nil
	})

	return struct{ Text string }{Text: response.String()}, err
}
