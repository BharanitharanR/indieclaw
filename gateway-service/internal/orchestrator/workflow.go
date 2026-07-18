package orchestrator

import (
	"context"
	"fmt"
	"log"
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
	// Create the LLM provider objects instead of passing strings
	// Replace "ollama" with your actual provider (e.g., "openai", "gemini") if needed

	memBank := memory.NewMemoryBankWithStore(memory.NewInMemoryStore())
	mem := memory.NewSessionMemory(memBank, 8)
	textModel, err := models.NewLLMProvider(ctx, "ollama", textModelName, "")
	if err != nil {
		return nil, err
	}

	visionModel, err := models.NewLLMProvider(ctx, "ollama", visionModelName, "")
	if err != nil {
		return nil, err
	}

	t, err := agent.New(agent.Options{
		SystemPrompt: "You are an expert orchestrator assistant.",
		Model:        textModel, // Now passing the provider object
		Memory:       mem,
	})
	if err != nil {
		return nil, err
	}

	v, err := agent.New(agent.Options{
		SystemPrompt: "You are an expert vision assistant.",
		Model:        visionModel, // Now passing the provider object
		Memory:       mem,
	})
	if err != nil {
		return nil, err
	}

	return &AgentWorkflow{textAgent: t, visionAgent: v}, nil
}
func (w *AgentWorkflow) Run(ctx context.Context, sessionID string, input string, isVision bool) (string, error) {
	var selectedAgent *agent.Agent
	if isVision {
		selectedAgent = w.visionAgent
	} else {
		selectedAgent = w.textAgent
	}

	response, err := selectedAgent.Generate(ctx, sessionID, input)
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
