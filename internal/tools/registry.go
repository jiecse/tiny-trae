package tools

import (
	"encoding/json"
	"github.com/anthropics/anthropic-sdk-go"
)

// GetAllTools returns all available tool definitions.
func GetAllTools() []ToolDefinition {
	return []ToolDefinition{
		ReadFileDefinition,
		ListFilesDefinition,
		EditFileDefinition,
		RipgrepDefinition,
		BashDefinition,
		CreateCodebaseSearchTool(),
	}
}

// GetMinimalTools returns a minimal set of tools for basic tasks.
func GetMinimalTools() []ToolDefinition {
	return []ToolDefinition{
		ReadFileDefinition,
		ListFilesDefinition,
		EditFileDefinition,
		CreateCodebaseSearchTool(),
	}
}

// CreateCodebaseSearchTool creates a tool definition for the codebase search agent
func CreateCodebaseSearchTool() ToolDefinition {
	return ToolDefinition{
		Name:        "codebase_search",
		Description: "Intelligent code search and discovery agent. Use this tool to find code based on functionality, concepts, or complex search requirements. It can discover relationships between code parts and perform contextual searches that go beyond simple keyword matching.",
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query or concept to find (e.g., 'authentication logic', 'error handling patterns', 'file upload functionality')",
				},
				"search_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of search to perform",
					"enum":        []string{"function", "concept", "pattern", "related", "architecture"},
				},
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "Optional: specific directory to search in (e.g., 'internal/auth', 'src/components')",
				},
				"file_types": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Optional: file extensions to focus on (e.g., ['.go', '.js', '.py'])",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "Optional: additional context to help with the search",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Optional: maximum number of results to return (default: 10)",
					"default":     10,
				},
			},
			Required: []string{"query"},
		},
		Function: func(input json.RawMessage) (string, error) {
			// This function will be handled by the main agent's executeSubAgentTool method
			// We return a placeholder here since the actual execution happens in the agent
			return "codebase_search_placeholder", nil
		},
	}
}
