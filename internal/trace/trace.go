package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// TraceData represents the data to be recorded for each inference
type TraceData struct {
	Timestamp   time.Time                     `json:"timestamp"`
	RequestID   string                        `json:"request_id"`
	Model       string                        `json:"model"`
	MaxTokens   int64                         `json:"max_tokens"`
	Messages    []anthropic.MessageParam      `json:"messages"`
	Tools       []anthropic.ToolUnionParam    `json:"tools"`
	SystemPrompt string                       `json:"system_prompt"`
}

// ResponseData represents the response data to be recorded
type ResponseData struct {
	Timestamp time.Time           `json:"timestamp"`
	RequestID string              `json:"request_id"`
	Message   *anthropic.Message  `json:"message,omitempty"`
	Error     string              `json:"error,omitempty"`
}

// Tracer handles recording inference traces
type Tracer struct {
	baseDir string
	enabled bool
}

// NewTracer creates a new Tracer instance
func NewTracer(baseDir string) *Tracer {
	return &Tracer{
		baseDir: baseDir,
		enabled: true,
	}
}

// SetEnabled enables or disables tracing
func (t *Tracer) SetEnabled(enabled bool) {
	t.enabled = enabled
}

// IsEnabled returns whether tracing is enabled
func (t *Tracer) IsEnabled() bool {
	return t.enabled
}

// RecordInference records a complete inference trace
func (t *Tracer) RecordInference(
	ctx context.Context,
	model anthropic.Model,
	maxTokens int64,
	messages []anthropic.MessageParam,
	tools []anthropic.ToolUnionParam,
	systemPrompt string,
	response *anthropic.Message,
	err error,
) error {
	if !t.enabled {
		return nil
	}

	// Generate unique request ID based on timestamp
	now := time.Now()
	requestID := fmt.Sprintf("%s", now.Format("20060102-150405-000000"))

	// Create trace directory
	traceDir := filepath.Join(t.baseDir, requestID)
	if err := os.MkdirAll(traceDir, 0755); err != nil {
		return fmt.Errorf("failed to create trace directory: %w", err)
	}

	// Record request
	requestData := TraceData{
		Timestamp:    now,
		RequestID:    requestID,
		Model:        string(model),
		MaxTokens:    maxTokens,
		Messages:     messages,
		Tools:        tools,
		SystemPrompt: systemPrompt,
	}

	if err := t.writeJSON(filepath.Join(traceDir, "request.json"), requestData); err != nil {
		return fmt.Errorf("failed to write request.json: %w", err)
	}

	// Record response
	responseData := ResponseData{
		Timestamp: now,
		RequestID: requestID,
		Message:   response,
	}
	if err != nil {
		responseData.Error = err.Error()
	}

	if err := t.writeJSON(filepath.Join(traceDir, "response.json"), responseData); err != nil {
		return fmt.Errorf("failed to write response.json: %w", err)
	}

	// Generate and write prompt.md
	promptContent := t.generatePromptMarkdown(requestData, responseData)
	if err := t.writeFile(filepath.Join(traceDir, "prompt.md"), promptContent); err != nil {
		return fmt.Errorf("failed to write prompt.md: %w", err)
	}

	return nil
}

// writeJSON writes data as JSON to the specified file
func (t *Tracer) writeJSON(filePath string, data interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// writeFile writes content to the specified file
func (t *Tracer) writeFile(filePath, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// generatePromptMarkdown generates a human-readable markdown representation
func (t *Tracer) generatePromptMarkdown(request TraceData, response ResponseData) string {
	var builder strings.Builder

	// Header
	builder.WriteString(fmt.Sprintf("# Inference Trace: %s\n\n", request.RequestID))
	builder.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", request.Timestamp.Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("**Model:** %s\n\n", request.Model))
	builder.WriteString(fmt.Sprintf("**Max Tokens:** %d\n\n", request.MaxTokens))

	// System Prompt
	builder.WriteString("## System Prompt\n\n")
	builder.WriteString("```\n")
	builder.WriteString(request.SystemPrompt)
	builder.WriteString("\n```\n\n")

	// Messages
	builder.WriteString("## Conversation\n\n")
	for i, msg := range request.Messages {
		builder.WriteString(fmt.Sprintf("### Message %d (%s)\n\n", i+1, msg.Role))
		
		// Handle different content types
		if msg.Content != nil {
			// For simplicity, convert content to string representation
			contentBytes, err := json.MarshalIndent(msg.Content, "", "  ")
			if err == nil {
				builder.WriteString(string(contentBytes))
				builder.WriteString("\n\n")
			} else {
				builder.WriteString("[Content could not be displayed]\n\n")
			}
		}
	}

	// Tools
	if len(request.Tools) > 0 {
		builder.WriteString("## Available Tools\n\n")
		for i, tool := range request.Tools {
			if tool.OfTool != nil {
				builder.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, tool.OfTool.Name))
				// Check if description is present by converting to JSON and checking
				if descBytes, err := json.Marshal(tool.OfTool.Description); err == nil && string(descBytes) != "null" {
					var desc string
					if err := json.Unmarshal(descBytes, &desc); err == nil {
						builder.WriteString(fmt.Sprintf("**Description:** %s\n\n", desc))
					}
				}
			}
		}
	}

	// Response
	builder.WriteString("## Response\n\n")
	if response.Error != "" {
		builder.WriteString(fmt.Sprintf("**Error:** %s\n\n", response.Error))
	} else if response.Message != nil {
		builder.WriteString(fmt.Sprintf("**Usage:** Input: %d tokens, Output: %d tokens\n\n", 
			response.Message.Usage.InputTokens, response.Message.Usage.OutputTokens))
		
		for i, content := range response.Message.Content {
			builder.WriteString(fmt.Sprintf("### Content %d (%s)\n\n", i+1, content.Type))
			switch content.Type {
			case "text":
				builder.WriteString(content.Text)
				builder.WriteString("\n\n")
			case "tool_use":
				builder.WriteString(fmt.Sprintf("**Tool:** %s\n\n", content.Name))
				builder.WriteString(fmt.Sprintf("**ID:** %s\n\n", content.ID))
				builder.WriteString("**Input:**\n\n")
				builder.WriteString("```json\n")
				inputBytes, _ := json.MarshalIndent(content.Input, "", "  ")
				builder.WriteString(string(inputBytes))
				builder.WriteString("\n```\n\n")
			}
		}
	}

	return builder.String()
}