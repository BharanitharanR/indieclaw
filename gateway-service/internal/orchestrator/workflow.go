package orchestrator

import (
	"context"
	"encoding/base64"
	"fmt"
	"gateway-service/internal/pkg/vision"
	"log"
	"net/http"
	"reflect"

	"github.com/Protocol-Lattice/go-agent"
	"github.com/Protocol-Lattice/go-agent/src/memory"
	"github.com/Protocol-Lattice/go-agent/src/models"
	// Add this import
)

type AgentWorkflow struct {
	textAgent   *agent.Agent
	visionAgent *agent.Agent
}

func NewAgentWorkflow(ctx context.Context, textModelName, visionModelName string) (*AgentWorkflow, error) {
	memBank := memory.NewMemoryBankWithStore(memory.NewInMemoryStore())
	mem := memory.NewSessionMemory(memBank, 8)

	// 1. Text Model initialization
	textModel, err := models.NewOllamaLLM(textModelName, "")
	if err != nil {
		return nil, err
	}

	// 2. Vision Model initialization (Manual creation + Wrapper)
	originalLLM, err := models.NewOllamaLLM(visionModelName, "")
	if err != nil {
		return nil, err
	}
	// Wrap it with your fixed logic
	fixedLLM := &vision.VisionFixedLLM{OllamaLLM: originalLLM}

	// 3. Text Agent
	t, err := agent.New(agent.Options{
		SystemPrompt: "You are an expert orchestrator assistant.",
		Model:        textModel,
		Memory:       mem,
	})
	if err != nil {
		return nil, err
	}

	// 4. Vision Agent (Using the fixedLLM wrapper)
	v, err := agent.New(agent.Options{
		SystemPrompt: "You are an expert vision assistant.",
		Model:        fixedLLM, // <--- Using the wrapped provider here
		Memory:       mem,
	})
	if err != nil {
		return nil, err
	}

	return &AgentWorkflow{textAgent: t, visionAgent: v}, nil
}

func (w *AgentWorkflow) Run(ctx context.Context, sessionID string, input string, isVision bool, imageBase64 string) (string, error) {
	var selectedAgent *agent.Agent
	var response interface{}
	var err error
	if isVision {
		// 1. CRITICAL: Decode the Base64 string into RAW BYTES
		imgBytes, err := base64.StdEncoding.DecodeString(imageBase64)
		// Add this:
		mimeType := http.DetectContentType(imgBytes)
		log.Printf("DEBUG: Detected MIME for file: %s", mimeType)
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %v", err)
		}

		// 2. Prepare the file structure with RAW BYTES
		files := []models.File{
			{
				Name: "input.jpg",
				MIME: "image/jpeg",
				Data: imgBytes, // The library expects raw binary data here
			},
		}

		// 3. Invoke the library method
		return w.visionAgent.GenerateWithFiles(ctx, sessionID, input, files)
	} else {
		selectedAgent = w.textAgent
		response, err = selectedAgent.Generate(ctx, sessionID, input)
	}

	// --- DEBUG START ---
	if response != nil {
		v := reflect.ValueOf(response)

		log.Printf("DEBUG: Type: %T, Kind: %v, ID: %s , IInput: %s", response, v.Kind(), sessionID, input)

		// If it's a struct/pointer to struct, print all fields
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				log.Printf("DEBUG: Field[%s]: %v", v.Type().Field(i).Name, v.Field(i).Interface())
			}
		}
	}
	if err != nil {
		log.Printf("DEBUG: Type: %T, Kind: %v", response, err.Error())
		return "", err
	}

	// 1. If it's already a string, return it immediately
	if str, ok := response.(string); ok {
		return str, nil
	}

	// 2. Reflective extraction (The "Magic" part)
	v := reflect.ValueOf(response)

	// If it's a pointer to a struct, get the element
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// If it's a struct, look for "Text"
	if v.Kind() == reflect.Struct {
		f := v.FieldByName("Text")
		if f.IsValid() && f.Kind() == reflect.String {
			return f.String(), nil
		}
	}

	// 3. Last ditch effort: If we can't find a "Text" field,
	// convert the whole thing to a string so the bot gets SOMETHING
	return fmt.Sprintf("%+v", response), nil
}
