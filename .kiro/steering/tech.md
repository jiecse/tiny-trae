# Technology Stack

## Language & Runtime
- **Go 1.24.1**: Primary programming language
- **Module**: `tiny-trae` (Go modules for dependency management)

## Key Dependencies
- **Anthropic SDK**: `github.com/anthropics/anthropic-sdk-go` - Claude API integration
- **Bubbletea**: `github.com/charmbracelet/bubbletea` - Terminal UI framework
- **Charm Libraries**: 
  - `bubbles` - UI components
  - `lipgloss` - Styling
  - `glamour` - Markdown rendering
- **JSON Schema**: `github.com/invopop/jsonschema` - Tool input validation

## External Tools
- **ripgrep**: Required for text search functionality (`brew install ripgrep` on macOS)

## Build & Development Commands

### Building
```bash
# Install dependencies
go mod tidy

# Build binary
go build -o tiny-trae

# Run from source
go run main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./internal/tools
```

### Environment Setup
```bash
# Required: Anthropic API key
export ANTHROPIC_API_KEY="your-api-key"

# Optional: Custom API endpoint
export ANTHROPIC_BASE_URL="custom-endpoint"

# For CloseAI integration
export CLOSEAI_API_KEY="sk-your-key"
export CLOSEAI_MODEL="claude-sonnet-4-20250514"
```

## API Integration
- **Primary**: Anthropic Claude API
- **Alternative**: CloseAI proxy for Claude access
- **Proxy Support**: OpenRouter via anthropic-proxy

## Architecture Pattern
- **Concurrent**: Agent core runs in separate goroutine
- **Message Passing**: Structured communication between components
- **Interface-based**: Frontend abstraction for extensibility