package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema"
)

// ToolDefinition represents a tool that can be used by agents
type ToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
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

// BashDefinition defines the 'bash' tool.
var BashDefinition = ToolDefinition{
	Name:        "bash",
	Description: "Execute a bash command.",
	InputSchema: BashInputSchema,
	Function:    Bash,
}

// BashInput defines the input schema for the 'bash' tool.
type BashInput struct {
	Command string `json:"command" jsonschema:"description=The command to execute"`
}

// BashInputSchema is the JSON schema for the 'bash' tool's input.
var BashInputSchema = GenerateSchema[BashInput]()

// Bash implements the 'bash' tool.
func Bash(input json.RawMessage) (string, error) {
	bashInput := BashInput{}
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("bash", "-c", bashInput.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command execution error: %v - %s", err, string(output))
	}

	return string(output), nil
}
