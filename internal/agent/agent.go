package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"tiny-trae/internal/subagent"
	"tiny-trae/internal/tools"
	"tiny-trae/internal/trace"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/invopop/jsonschema"
)

// ToolDefinition struct defines a tool that the agent can use.
type ToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
}

// Profile represents a configuration that combines model settings, tools, and system prompt.
type Profile struct {
	Name         string
	Model        anthropic.Model
	MaxTokens    int64
	Tools        []ToolDefinition
	SystemPrompt string
}

// Agent struct represents the core of the AI agent.
type Agent struct {
	client    anthropic.Client
	profile   *Profile
	frontend  Frontend
	tracer    *trace.Tracer
	subAgents map[string]subagent.SubAgent
}

// NewAgent creates a new Agent instance with a profile and frontend.
func NewAgent(
	client anthropic.Client,
	profile *Profile,
	frontend Frontend,
	tracer *trace.Tracer,
) *Agent {
	agent := &Agent{
		client:    client,
		profile:   profile,
		frontend:  frontend,
		tracer:    tracer,
		subAgents: make(map[string]subagent.SubAgent),
	}

	// Initialize sub-agents
	agent.InitializeSubAgents()

	return agent
}

// NewAgentWithDefaults creates a new Agent instance with individual parameters (legacy).
// Deprecated: Use NewAgent with a Profile instead.
func NewAgentWithDefaults(
	client anthropic.Client,
	tools []ToolDefinition,
	systemPrompt string,
	frontend Frontend,
	tracer *trace.Tracer,
) *Agent {
	profile := &Profile{
		Name:         "legacy",
		Model:        anthropic.ModelClaudeSonnet4_0,
		MaxTokens:    1024,
		Tools:        tools,
		SystemPrompt: systemPrompt,
	}
	agent := NewAgent(client, profile, frontend, tracer)

	// Initialize sub-agents
	agent.InitializeSubAgents()

	return agent
}

// NewClientWithOptions creates a new Anthropic client with the given options.
func NewClientWithOptions(options ...option.RequestOption) anthropic.Client {
	return anthropic.NewClient(options...)
}

// Run starts the agent's main loop in a separate goroutine.
// It continuously processes user input and model responses, communicating with the frontend
// through the Frontend interface. The core logic runs independently from the UI.
func (a *Agent) Run(ctx context.Context, initialMessage string) error {
	// Send initial system message
	if initialMessage == "" {
		a.frontend.SendMessage(Message{
			Type:    MessageTypeSystemInfo,
			Content: "Chat with Tiny Trae (use CTRL+C to exit)",
		})
	}

	// Start the core agent loop in a goroutine
	errorChan := make(chan error, 1)
	go func() {
		errorChan <- a.runCore(ctx, initialMessage)
	}()

	// Wait for completion or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errorChan:
		return err
	}
}

// runCore contains the main agent logic that runs in a separate goroutine
func (a *Agent) runCore(ctx context.Context, initialMessage string) error {
	conversation := []anthropic.MessageParam{}

	if initialMessage != "" {
		userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(initialMessage))
		conversation = append(conversation, userMessage)
		// Send user input message to frontend
		a.frontend.SendMessage(Message{
			Type:    MessageTypeUserInput,
			Content: initialMessage,
		})
	}

	readUserInput := initialMessage == ""
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if readUserInput {
			userInput, ok := a.frontend.GetUserInput()
			if !ok {
				break
			}

			userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(userInput))
			conversation = append(conversation, userMessage)

			// Send user input message to frontend
			a.frontend.SendMessage(Message{
				Type:    MessageTypeUserInput,
				Content: userInput,
			})
		}

		message, err := a.runInference(ctx, conversation)
		if err != nil {
			a.frontend.SendMessage(Message{
				Type:    MessageTypeError,
				Content: fmt.Sprintf("LLM request failed: %v", err),
			})

			// In interactive mode, continue the loop to allow user to try again
			if a.frontend.IsInteractive() {
				readUserInput = true
				continue
			} else {
				// In non-interactive mode, return error to exit
				return err
			}
		}
		conversation = append(conversation, message.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, content := range message.Content {
			switch content.Type {
			case "text":
				// Send assistant message to frontend
				// Always show assistant messages to ensure tool feedback is displayed
				a.frontend.SendMessage(Message{
					Type:    MessageTypeAssistant,
					Content: content.Text,
				})
			case "tool_use":
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}

		if len(toolResults) == 0 {
			// If no tools were used, check if we should continue reading input based on interactive mode
			if a.frontend.IsInteractive() {
				// In interactive mode, continue to read user input
				readUserInput = true
				continue
			} else {
				// In non-interactive mode, exit after processing the message
				return nil
			}
		}

		// After tool execution, add tool results to conversation and continue inference
		conversation = append(conversation, anthropic.NewUserMessage(toolResults...))

		// Continue the inference loop to get model's response to tool results
		// Don't read user input in the next iteration, let the model respond to tool results first
		readUserInput = false
		continue
	}

	return nil
}

// InitializeSubAgents initializes all sub-agents
func (a *Agent) InitializeSubAgents() {
	// Initialize codebase search agent
	codebaseSearchAgent := subagent.NewCodebaseSearchAgent(a.client)
	a.subAgents[codebaseSearchAgent.GetName()] = codebaseSearchAgent
}

// GetSubAgent returns a sub-agent by name
func (a *Agent) GetSubAgent(name string) (subagent.SubAgent, bool) {
	subAgent, exists := a.subAgents[name]
	return subAgent, exists
}

// ListSubAgents returns all available sub-agents
func (a *Agent) ListSubAgents() map[string]subagent.SubAgent {
	return a.subAgents
}

// executeSubAgentTool executes a sub-agent tool call
func (a *Agent) executeSubAgentTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	// Find the sub-agent
	subAgent, exists := a.subAgents[name]
	if !exists {
		errorMsg := fmt.Sprintf("sub-agent '%s' not found", name)
		// Send tool result message to frontend
		toolResultData := ToolResultData{
			ToolName: name,
			ToolID:   id,
			Result:   errorMsg,
			IsError:  true,
		}
		data, err := json.Marshal(toolResultData)
		if err != nil {
			a.frontend.SendMessage(Message{
				Type:    MessageTypeToolResult,
				Content: errorMsg,
			})
		} else {
			a.frontend.SendMessage(Message{
				Type:    MessageTypeToolResult,
				Content: "",
				Data:    data,
			})
		}
		return anthropic.NewToolResultBlock(id, errorMsg, true)
	}

	// Parse the input based on the sub-agent type
	var request subagent.SubAgentRequest
	if name == "codebase_search" {
		var searchRequest subagent.CodebaseSearchRequest
		if err := json.Unmarshal(input, &searchRequest); err != nil {
			errorMsg := fmt.Sprintf("failed to parse input: %v", err)
			toolResultData := ToolResultData{
				ToolName: name,
				ToolID:   id,
				Result:   errorMsg,
				IsError:  true,
			}
			data, marshalErr := json.Marshal(toolResultData)
			if marshalErr != nil {
				a.frontend.SendMessage(Message{
					Type:    MessageTypeToolResult,
					Content: errorMsg,
				})
			} else {
				a.frontend.SendMessage(Message{
					Type:    MessageTypeToolResult,
					Content: "",
					Data:    data,
				})
			}
			return anthropic.NewToolResultBlock(id, errorMsg, true)
		}
		request = subagent.SubAgentRequest{
			Task:    fmt.Sprintf("Search for: %s (type: %s)", searchRequest.Query, searchRequest.SearchType),
			Context: searchRequest.Context,
			Parameters: map[string]interface{}{
				"query":       searchRequest.Query,
				"search_type": searchRequest.SearchType,
				"directory":   searchRequest.Directory,
				"file_types":  searchRequest.FileTypes,
				"max_results": searchRequest.MaxResults,
				"parameters":  searchRequest.Parameters,
			},
		}
	} else {
		errorMsg := fmt.Sprintf("unknown sub-agent type: %s", name)
		toolResultData := ToolResultData{
			ToolName: name,
			ToolID:   id,
			Result:   errorMsg,
			IsError:  true,
		}
		data, err := json.Marshal(toolResultData)
		if err != nil {
			a.frontend.SendMessage(Message{
				Type:    MessageTypeToolResult,
				Content: errorMsg,
			})
		} else {
			a.frontend.SendMessage(Message{
				Type:    MessageTypeToolResult,
				Content: "",
				Data:    data,
			})
		}
		return anthropic.NewToolResultBlock(id, errorMsg, true)
	}

	// Send tool call message to frontend
	toolCallData := ToolCallData{
		ToolName: name,
		ToolID:   id,
		Input:    input,
	}
	data, err := json.Marshal(toolCallData)
	if err != nil {
		a.frontend.SendMessage(Message{
			Type:    MessageTypeToolCall,
			Content: fmt.Sprintf("Executing sub-agent: %s", name),
		})
	} else {
		a.frontend.SendMessage(Message{
			Type:    MessageTypeToolCall,
			Content: fmt.Sprintf("Executing sub-agent: %s", name),
			Data:    data,
		})
	}

	// Execute the sub-agent
	ctx := context.Background()
	response, err := subAgent.Execute(ctx, request)
	if err != nil {
		errorMsg := fmt.Sprintf("sub-agent execution failed: %v", err)
		toolResultData := ToolResultData{
			ToolName: name,
			ToolID:   id,
			Result:   errorMsg,
			IsError:  true,
		}
		data, marshalErr := json.Marshal(toolResultData)
		if marshalErr != nil {
			a.frontend.SendMessage(Message{
				Type:    MessageTypeToolResult,
				Content: errorMsg,
			})
		} else {
			a.frontend.SendMessage(Message{
				Type:    MessageTypeToolResult,
				Content: "",
				Data:    data,
			})
		}
		return anthropic.NewToolResultBlock(id, errorMsg, true)
	}

	// Send successful result to frontend
	toolResultData := ToolResultData{
		ToolName: name,
		ToolID:   id,
		Result:   response.Result,
		IsError:  false,
	}
	data, err = json.Marshal(toolResultData)
	if err != nil {
		a.frontend.SendMessage(Message{
			Type:    MessageTypeToolResult,
			Content: response.Result,
		})
	} else {
		a.frontend.SendMessage(Message{
			Type:    MessageTypeToolResult,
			Content: response.Result,
			Data:    data,
		})
	}

	return anthropic.NewToolResultBlock(id, response.Result, false)
}

// runInference sends the conversation to the Anthropic API and gets the model's response.
// It constructs a list of tools available for the model to use and includes them in the API request.
// The function returns the model's response message or an error if the API call fails.
func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range a.profile.Tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     a.profile.Model,
		MaxTokens: a.profile.MaxTokens,
		Messages:  conversation,
		Tools:     anthropicTools,
		System:    []anthropic.TextBlockParam{{Text: a.profile.SystemPrompt}},
	})

	// Record trace if tracer is available
	if a.tracer != nil {
		traceErr := a.tracer.RecordInference(
			ctx,
			a.profile.Model,
			a.profile.MaxTokens,
			conversation,
			anthropicTools,
			a.profile.SystemPrompt,
			message,
			err,
		)
		if traceErr != nil {
			// Log trace error but don't fail the main operation
			fmt.Printf("Warning: Failed to record trace: %v\n", traceErr)
		}
	}

	return message, err
}

// executeTool executes a tool with the given name and input.
// It finds the corresponding tool definition, calls its associated function with the provided input,
// and returns the result as a tool result block. If the tool is not found or an error occurs
// during execution, it returns an error message in the tool result block.
func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	// Check if this is a sub-agent tool call
	if name == "codebase_search" {
		return a.executeSubAgentTool(id, name, input)
	}

	var toolDef ToolDefinition
	var found bool
	for _, tool := range a.profile.Tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}
	if !found {
		// Send tool result message to frontend
		toolResultData := ToolResultData{
			ToolName: name,
			ToolID:   id,
			Result:   "tool not found",
			IsError:  true,
		}
		data, err := json.Marshal(toolResultData)
		if err != nil {
			// Fallback to sending message without data if marshaling fails
			a.frontend.SendMessage(Message{
				Type:    MessageTypeToolResult,
				Content: "tool not found",
			})
		} else {
			a.frontend.SendMessage(Message{
				Type:    MessageTypeToolResult,
				Content: "",
				Data:    data,
			})
		}
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	// Send tool call message to frontend
	toolCallData := ToolCallData{
		ToolName: name,
		ToolID:   id,
		Input:    input,
	}
	data, err := json.Marshal(toolCallData)
	if err != nil {
		// Fallback to sending message without data if marshaling fails
		a.frontend.SendMessage(Message{
			Type:    MessageTypeToolCall,
			Content: fmt.Sprintf("Executing tool: %s", name),
		})
	} else {
		a.frontend.SendMessage(Message{
			Type:    MessageTypeToolCall,
			Content: fmt.Sprintf("Executing tool: %s", name),
			Data:    data,
		})
	}

	response, err := toolDef.Function(input)
	isError := err != nil
	result := response
	if err != nil {
		result = err.Error()
	}

	// Send tool result message to frontend
	toolResultData := ToolResultData{
		ToolName: name,
		ToolID:   id,
		Result:   result,
		IsError:  isError,
	}
	data, err = json.Marshal(toolResultData)
	if err != nil {
		// Fallback to sending message without data if marshaling fails
		a.frontend.SendMessage(Message{
			Type:    MessageTypeToolResult,
			Content: result,
		})
	} else {
		a.frontend.SendMessage(Message{
			Type:    MessageTypeToolResult,
			Content: result,
			Data:    data,
		})
	}

	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}
	return anthropic.NewToolResultBlock(id, response, false)
}

// GenerateSchema generates a JSON schema for a given type.
func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var v T
	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Type:       "object",
		Properties: schema.Properties,
	}
}

// ConvertToolsToAgentTools converts tools.ToolDefinition slice to agent.ToolDefinition slice
func ConvertToolsToAgentTools(toolsDefinitions []tools.ToolDefinition) []ToolDefinition {
	agentTools := make([]ToolDefinition, len(toolsDefinitions))
	for i, tool := range toolsDefinitions {
		agentTools[i] = ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
			Function:    tool.Function,
		}
	}
	return agentTools
}
