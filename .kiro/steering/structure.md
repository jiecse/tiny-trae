# Project Structure

## Root Level
- `main.go` - Application entry point with CLI flag handling
- `go.mod/go.sum` - Go module dependencies
- `README.md` - Project documentation
- `ARCHITECTURE.md` - Detailed architecture documentation
- `tiny-trae` - Compiled binary (generated)

## Core Directories

### `/internal` - Private application code
- **`/agent`** - Core agent logic and message system
  - `agent.go` - Main agent implementation with conversation loop
  - `message.go` - Message types and Frontend interface definition
- **`/tools`** - Tool implementations and registry
  - `registry.go` - Tool collection and organization
  - `*_tool.go` - Individual tool implementations (bash, edit_file, etc.)
  - `*_test.go` - Tool-specific tests
- **`/profile`** - Configuration profiles for different use cases
  - `profile.go` - Profile definitions (default, minimal, closeai)
- **`/frontend`** - UI implementations
  - `tui.go` - Terminal user interface using bubbletea
- **`/prompt`** - System prompt management
- **`/subagent`** - Specialized sub-agents (e.g., codebase search)
- **`/trace`** - Request/response tracing functionality

### `/traces` - Runtime trace files (generated)
- Timestamped directories containing request.json, response.json, prompt.md

### `/.kiro` - Kiro IDE configuration
- `/steering` - AI assistant guidance documents

### `/.trae` - Project-specific rules
- `/rules/project_rules.md` - Development conventions

## Code Organization Patterns

### Tool Structure
- Each tool has a `ToolDefinition` struct with Name, Description, InputSchema, Function
- Input schemas use JSON schema generation via reflection
- Tools are registered in `registry.go` and grouped by profile

### Message Flow
- Frontend interface defines `SendMessage()` and `GetUserInput()` methods
- Message types: UserInput, Assistant, ToolCall, ToolResult, Error, SystemInfo
- Agent core communicates with frontend through structured messages

### Profile System
- Profiles combine model settings, tool sets, and system prompts
- Switchable via `--profile` flag (default, minimal, closeai)
- Environment-based configuration for API keys and models

## Naming Conventions
- Go standard: PascalCase for exported, camelCase for unexported
- Files: snake_case with descriptive names
- Packages: lowercase, single word when possible
- Tool names: lowercase with underscores (e.g., `edit_file`, `codebase_search`)