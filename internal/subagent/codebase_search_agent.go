package subagent

import (
	"context"
	"fmt"

	"tiny-trae/internal/tools"
	"tiny-trae/internal/trace"

	"github.com/anthropics/anthropic-sdk-go"
)

// CodebaseSearchAgent is a specialized sub-agent for intelligent code search and discovery
type CodebaseSearchAgent struct {
	*BaseSubAgent
}

// NewCodebaseSearchAgent creates a new codebase search agent
func NewCodebaseSearchAgent(client anthropic.Client, tracer *trace.Tracer, frontend Frontend) *CodebaseSearchAgent {
	// Define tools available to this sub-agent
	subAgentTools := []ToolDefinition{
		{
			Name:        "list_files",
			Description: "List files and directories",
			InputSchema: tools.ListFilesDefinition.InputSchema,
			Function:    tools.ListFilesDefinition.Function,
		},
		{
			Name:        "ripgrep",
			Description: "Search for patterns in files using regular expressions",
			InputSchema: tools.RipgrepDefinition.InputSchema,
			Function:    tools.RipgrepDefinition.Function,
		},
		{
			Name:        "read_file",
			Description: "Read the contents of a file",
			InputSchema: tools.ReadFileDefinition.InputSchema,
			Function:    tools.ReadFileDefinition.Function,
		},
	}

	baseAgent := NewBaseSubAgent(
		client,
		subAgentTools,
		"codebase_search",
		"Intelligent code search and discovery agent that can find code based on functionality and concepts",
		tracer,
		frontend,
	)

	return &CodebaseSearchAgent{
		BaseSubAgent: baseAgent,
	}
}

// Execute performs intelligent code search based on the request
func (c *CodebaseSearchAgent) Execute(ctx context.Context, request SubAgentRequest) (SubAgentResponse, error) {
	systemPrompt := `You are a specialized codebase search agent. Your role is to intelligently search and discover code based on functionality, concepts, and relationships.

Core responsibilities:
- Perform intelligent code search using multiple techniques
- Find code based on functionality and concepts, not just keywords
- Discover connections between different parts of the codebase
- Provide contextual filtering for keyword searches

Available tools:
- list_files: List files and directories in the codebase
- ripgrep: Search for patterns in files using regular expressions
- read_file: Read specific files to understand their content

Search strategies you should use:
1. Start with broad exploration using list_files to understand codebase structure
2. Use ripgrep for pattern-based searches with relevant keywords
3. Read key files to understand context and relationships
4. Combine multiple search approaches for comprehensive results
5. Filter and prioritize results based on relevance

When searching:
- Consider variations of terms (e.g., "auth" vs "authentication" vs "login")
- Look for related concepts (e.g., when searching for "error handling", also look for "exception", "try-catch", "panic", etc.)
- Examine file names, function names, comments, and code structure
- Identify patterns and architectural relationships

Always provide:
- Clear explanation of your search strategy
- Relevant code locations with file paths and line numbers
- Brief description of what each result contains
- Relationships between different parts if applicable`

	userPrompt := fmt.Sprintf("Task: %s\n\nContext: %s\n\nParameters: %v",
		request.Task, request.Context, request.Parameters)

	return c.executeWithLLM(ctx, systemPrompt, userPrompt)
}

// GetSearchSuggestions provides search suggestions based on common patterns
func GetSearchSuggestions() map[string][]string {
	return map[string][]string{
		"authentication": {"auth", "login", "session", "token", "jwt", "oauth", "credentials"},
		"error_handling": {"error", "exception", "try", "catch", "panic", "recover", "throw"},
		"database":       {"db", "sql", "query", "transaction", "migration", "schema", "model"},
		"api":            {"rest", "endpoint", "route", "handler", "controller", "middleware"},
		"configuration":  {"config", "env", "settings", "options", "params", "flags"},
		"validation":     {"validate", "sanitize", "check", "verify", "constraint"},
		"logging":        {"log", "logger", "debug", "info", "warn", "error", "trace"},
		"testing":        {"test", "spec", "mock", "stub", "assert", "expect"},
		"security":       {"secure", "encrypt", "decrypt", "hash", "salt", "csrf", "xss"},
		"caching":        {"cache", "redis", "memcache", "store", "ttl", "expire"},
	}
}
