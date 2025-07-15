package subagent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"tiny-trae/internal/tools"
)

// ToolDefinition is an alias for tools.ToolDefinition
type ToolDefinition = tools.ToolDefinition

// SubAgentRequest represents a request to a sub-agent
type SubAgentRequest struct {
	Task       string                 `json:"task"`
	Context    string                 `json:"context"`
	Parameters map[string]interface{} `json:"parameters"`
}

// SubAgentResponse represents the response from a sub-agent
type SubAgentResponse struct {
	Success bool   `json:"success"`
	Result  string `json:"result"`
	Error   string `json:"error,omitempty"`
}

// SubAgent interface defines the contract for sub-agents
type SubAgent interface {
	GetName() string
	GetDescription() string
	Execute(ctx context.Context, request SubAgentRequest) (SubAgentResponse, error)
}

// BaseSubAgent provides common functionality for sub-agents
type BaseSubAgent struct {
	client      anthropic.Client
	tools       []ToolDefinition
	name        string
	description string
}

// NewBaseSubAgent creates a new base sub-agent
func NewBaseSubAgent(client anthropic.Client, tools []ToolDefinition, name, description string) *BaseSubAgent {
	return &BaseSubAgent{
		client:      client,
		tools:       tools,
		name:        name,
		description: description,
	}
}

// GetName returns the sub-agent's name
func (b *BaseSubAgent) GetName() string {
	return b.name
}

// GetDescription returns the sub-agent's description
func (b *BaseSubAgent) GetDescription() string {
	return b.description
}

// executeWithLLM executes a sub-agent task using the LLM
func (b *BaseSubAgent) executeWithLLM(ctx context.Context, systemPrompt, userPrompt string) (SubAgentResponse, error) {
	// Convert tools to anthropic format
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range b.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	// Create conversation
	conversation := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
	}

	// Call LLM
	message, err := b.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_0,
		MaxTokens: 2048,
		Messages:  conversation,
		Tools:     anthropicTools,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
	})
	if err != nil {
		return SubAgentResponse{
			Success: false,
			Error:   fmt.Sprintf("LLM request failed: %v", err),
		}, nil
	}

	// Process response and execute tools
	var textResponse string
	for _, content := range message.Content {
		switch content.Type {
		case "text":
			textResponse += content.Text
		case "tool_use":
			// Execute tool
			toolResult, err := b.executeTool(content.Name, content.Input)
			if err != nil {
				return SubAgentResponse{
					Success: false,
					Error:   fmt.Sprintf("Tool execution failed: %v", err),
				}, nil
			}
			textResponse += "\n\nTool Result: " + toolResult
		}
	}

	return SubAgentResponse{
		Success: true,
		Result:  textResponse,
	}, nil
}

// executeTool executes a tool with the given name and input
func (b *BaseSubAgent) executeTool(name string, input json.RawMessage) (string, error) {
	for _, tool := range b.tools {
		if tool.Name == name {
			return tool.Function(input)
		}
	}
	return "", fmt.Errorf("tool not found: %s", name)
}

// CodebaseSearchRequest represents a request to search the codebase
type CodebaseSearchRequest struct {
	Query       string            `json:"query"`       // The search query or concept to find
	SearchType  string            `json:"search_type"` // Type of search: "function", "concept", "pattern", "related"
	Directory   string            `json:"directory"`   // Optional: specific directory to search in
	FileTypes   []string          `json:"file_types"`  // Optional: file extensions to focus on
	Context     string            `json:"context"`     // Optional: additional context for the search
	MaxResults  int               `json:"max_results"` // Optional: maximum number of results to return
	Parameters  map[string]string `json:"parameters"`  // Optional: additional search parameters
}