package subagent

import (
	"context"
	"encoding/json"
	"fmt"

	"tiny-trae/internal/tools"
	"tiny-trae/internal/trace"

	"github.com/anthropics/anthropic-sdk-go"
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

// Frontend interface for sub-agents - simplified interface to avoid circular imports
type Frontend interface {
	SendMessage(msg Message)
}

// Message types for sub-agent frontend communication
type MessageType string

const (
	MessageTypeAssistant  MessageType = "assistant"
	MessageTypeToolCall   MessageType = "tool_call"
	MessageTypeToolResult MessageType = "tool_result"
)

type Message struct {
	Type    MessageType     `json:"type"`
	Content string          `json:"content"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type ToolCallData struct {
	ToolName string          `json:"tool_name"`
	ToolID   string          `json:"tool_id"`
	Input    json.RawMessage `json:"input"`
}

type ToolResultData struct {
	ToolName string `json:"tool_name"`
	ToolID   string `json:"tool_id"`
	Result   string `json:"result"`
	IsError  bool   `json:"is_error"`
}

// BaseSubAgent provides common functionality for sub-agents
type BaseSubAgent struct {
	client      anthropic.Client
	tools       []ToolDefinition
	name        string
	description string
	tracer      *trace.Tracer
	frontend    Frontend
}

// NewBaseSubAgent creates a new base sub-agent
func NewBaseSubAgent(client anthropic.Client, tools []ToolDefinition, name, description string, tracer *trace.Tracer, frontend Frontend) *BaseSubAgent {
	return &BaseSubAgent{
		client:      client,
		tools:       tools,
		name:        name,
		description: description,
		tracer:      tracer,
		frontend:    frontend,
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

// executeWithLLM executes a sub-agent task using the LLM with multi-turn conversation
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

	// Initialize conversation with user request
	conversation := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
	}

	var finalResponse string
	var allTextContent []string // Collect all text responses
	maxIterations := 10         // Prevent infinite loops

	for iteration := 0; iteration < maxIterations; iteration++ {
		// Call LLM
		message, err := b.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeSonnet4_0,
			MaxTokens: 2048,
			Messages:  conversation,
			Tools:     anthropicTools,
			System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		})

		// Record trace if tracer is available
		if b.tracer != nil {
			traceErr := b.tracer.RecordInference(
				ctx,
				anthropic.ModelClaudeSonnet4_0,
				2048,
				conversation,
				anthropicTools,
				systemPrompt,
				message,
				err,
			)
			if traceErr != nil {
				// Log trace error but don't fail the main operation
				fmt.Printf("Warning: Failed to record subagent trace: %v\n", traceErr)
			}
		}

		if err != nil {
			return SubAgentResponse{
				Success: false,
				Error:   fmt.Sprintf("LLM request failed: %v", err),
			}, nil
		}

		// Add assistant message to conversation
		conversation = append(conversation, message.ToParam())

		// Process response and execute tools
		var textContent string
		toolResults := []anthropic.ContentBlockParamUnion{}
		hasTools := false

		for _, content := range message.Content {
			switch content.Type {
			case "text":
				textContent += content.Text
				// Collect all text content for potential fallback
				if content.Text != "" {
					allTextContent = append(allTextContent, content.Text)
				}
				// Send assistant message to frontend if there's text content
				if b.frontend != nil && content.Text != "" {
					b.frontend.SendMessage(Message{
						Type:    MessageTypeAssistant,
						Content: fmt.Sprintf("[%s] %s", b.name, content.Text),
					})
				}
			case "tool_use":
				hasTools = true

				// Send tool call message to frontend
				if b.frontend != nil {
					toolCallData := ToolCallData{
						ToolName: content.Name,
						ToolID:   content.ID,
						Input:    content.Input,
					}
					data, err := json.Marshal(toolCallData)
					if err != nil {
						b.frontend.SendMessage(Message{
							Type:    MessageTypeToolCall,
							Content: fmt.Sprintf("[%s] Executing tool: %s", b.name, content.Name),
						})
					} else {
						b.frontend.SendMessage(Message{
							Type:    MessageTypeToolCall,
							Content: fmt.Sprintf("[%s] Executing tool: %s", b.name, content.Name),
							Data:    data,
						})
					}
				}

				// Execute tool
				toolResult, err := b.executeTool(content.Name, content.Input)

				// Send tool result message to frontend
				if b.frontend != nil {
					toolResultData := ToolResultData{
						ToolName: content.Name,
						ToolID:   content.ID,
						Result:   toolResult,
						IsError:  err != nil,
					}
					if err != nil {
						toolResultData.Result = err.Error()
					}
					data, marshalErr := json.Marshal(toolResultData)
					if marshalErr != nil {
						b.frontend.SendMessage(Message{
							Type:    MessageTypeToolResult,
							Content: fmt.Sprintf("[%s] Tool result: %s", b.name, toolResultData.Result),
						})
					} else {
						b.frontend.SendMessage(Message{
							Type:    MessageTypeToolResult,
							Content: fmt.Sprintf("[%s] Tool result: %s", b.name, toolResultData.Result),
							Data:    data,
						})
					}
				}

				if err != nil {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(content.ID, fmt.Sprintf("Error: %v", err), true))
				} else {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(content.ID, toolResult, false))
				}
			}
		}

		// If no tools were used, this is the final response
		if !hasTools {
			finalResponse = textContent
			break
		}

		// Add tool results to conversation and continue
		if len(toolResults) > 0 {
			conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
		}
	}

	// Handle case where we reached max iterations without a final response
	if finalResponse == "" {
		// Try to generate a summary response based on collected information
		if len(allTextContent) > 0 {
			// Send a message to frontend indicating we're generating a summary
			if b.frontend != nil {
				b.frontend.SendMessage(Message{
					Type:    MessageTypeAssistant,
					Content: fmt.Sprintf("[%s] Reached iteration limit, generating summary from collected information...", b.name),
				})
			}

			// Attempt one final call to get a summary
			summaryPrompt := fmt.Sprintf("Based on the work I've done so far, please provide a concise summary of the findings. Here's what I've discovered:\n\n%s\n\nPlease summarize the key findings and provide a helpful response.",
				fmt.Sprintf("Previous responses: %s", fmt.Sprintf("%v", allTextContent)))

			summaryMessage, err := b.client.Messages.New(ctx, anthropic.MessageNewParams{
				Model:     anthropic.ModelClaudeSonnet4_0,
				MaxTokens: 1024,
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(anthropic.NewTextBlock(summaryPrompt)),
				},
				System: []anthropic.TextBlockParam{{Text: "You are a helpful assistant that provides concise summaries. Do not use any tools, just provide a direct text response."}},
			})

			if err == nil && len(summaryMessage.Content) > 0 {
				for _, content := range summaryMessage.Content {
					if content.Type == "text" && content.Text != "" {
						finalResponse = content.Text
						// Send summary to frontend
						if b.frontend != nil {
							b.frontend.SendMessage(Message{
								Type:    MessageTypeAssistant,
								Content: fmt.Sprintf("[%s] %s", b.name, content.Text),
							})
						}
						break
					}
				}
			}
		}

		// If we still don't have a response, create a fallback response
		if finalResponse == "" {
			if len(allTextContent) > 0 {
				// Use the last meaningful text content as fallback
				finalResponse = fmt.Sprintf("I've been working on your request and gathered some information, but reached the iteration limit before completing the full analysis. Here's what I found:\n\n%s\n\nPlease try a more specific query if you need additional details.", allTextContent[len(allTextContent)-1])
			} else {
				finalResponse = "I reached the maximum number of iterations while processing your request. The task may be too complex or require a more specific approach. Please try breaking down your request into smaller, more focused queries."
			}

			// Send fallback message to frontend
			if b.frontend != nil {
				b.frontend.SendMessage(Message{
					Type:    MessageTypeAssistant,
					Content: fmt.Sprintf("[%s] %s", b.name, finalResponse),
				})
			}
		}

		return SubAgentResponse{
			Success: true, // Still return success with partial results
			Result:  finalResponse,
		}, nil
	}

	return SubAgentResponse{
		Success: true,
		Result:  finalResponse,
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
	Query      string            `json:"query"`       // The search query or concept to find
	SearchType string            `json:"search_type"` // Type of search: "function", "concept", "pattern", "related"
	Directory  string            `json:"directory"`   // Optional: specific directory to search in
	FileTypes  []string          `json:"file_types"`  // Optional: file extensions to focus on
	Context    string            `json:"context"`     // Optional: additional context for the search
	MaxResults int               `json:"max_results"` // Optional: maximum number of results to return
	Parameters map[string]string `json:"parameters"`  // Optional: additional search parameters
}
