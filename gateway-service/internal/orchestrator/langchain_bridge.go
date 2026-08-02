package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/tools"
)

type LangChainToolBridge struct {
	def *ToolDefinition
}

func NewLangChainToolBridge(def *ToolDefinition) *LangChainToolBridge {
	return &LangChainToolBridge{def: def}
}

func (t *LangChainToolBridge) Name() string {
	return t.def.Name
}

func (t *LangChainToolBridge) Description() string {
	return t.def.Description
}

func (t *LangChainToolBridge) Call(ctx context.Context, inputStr string) (string, error) {
	var args map[string]interface{}

	if err := json.Unmarshal([]byte(inputStr), &args); err != nil {
		fallbackKey := "query"
		for k := range t.def.InputSchema {
			if k != "type" && k != "properties" && k != "required" {
				fallbackKey = k
				break
			}
		}
		args = map[string]interface{}{fallbackKey: inputStr}
	}

	result, err := t.def.Handler(ctx, args)
	if err != nil {
		return fmt.Sprintf("Error executing tool: %v", err), nil
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("%v", result), nil
	}

	return string(resultBytes), nil
}

// GetLangChainTools returns a strictly typed []tools.Tool slice
func (tr *ToolRegistry) GetLangChainTools() []tools.Tool {
	var lcTools []tools.Tool
	for _, toolDef := range tr.tools {
		lcTools = append(lcTools, NewLangChainToolBridge(toolDef))
	}
	return lcTools
}
