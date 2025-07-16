# Tiny Trae

A minimal AI coding agent powered by Anthropic's Claude with a modular frontend architecture.

🌐 **Landing Page**: https://lldong.github.io/tiny-trae

## Architecture

Tiny Trae has been designed with a clean separation between the core agent logic and the frontend interface. This allows for easy extension with different types of user interfaces.

### Core Components

- **Agent Core**: Handles AI conversation logic and tool execution in a separate goroutine
- **Frontend Interface**: Defines how different UIs can interact with the agent
- **Message System**: Structured communication between core and frontend
- **TUI Frontend**: Terminal user interface implementation using bubbletea

### Frontend

- **TUI**: Terminal user interface with rich interface using bubbletea

For detailed architecture information, see [ARCHITECTURE.md](ARCHITECTURE.md).

This project is a simple AI coding agent implemented in Go. It uses the Anthropic API to interact with a large language model (Claude) to help with software engineering tasks. The agent can execute a predefined set of tools based on the model's response.

## Features

- **Interactive Chat:** Chat with the agent from your terminal.
- **Non-interactive mode:** Provide input directly from the command line.
- **Sub-Agent System:** Specialized agents for complex tasks like intelligent code search.
- **Tool Execution:** The agent can execute the following tools:
    - `read_file`: Read the contents of a file.
    - `list_files`: List files and directories.
    - `edit_file`: Modify files by searching and replacing text.
    - `ripgrep`: Search for text patterns within files.
    - `bash`: Execute shell commands.
    - `codebase_search`: Intelligent code search and discovery using specialized sub-agent.
- **Complete Tracing:** Full tracing support for both main agent and sub-agent interactions.
- **Robust Error Handling:** Smart iteration limits with graceful fallback mechanisms.
- **Extensible:** Easily add new tools and sub-agents to the system.

## Prerequisites

- Go 1.x
- An [Anthropic API key](https://console.anthropic.com/dashboard)
- **ripgrep**: This tool is used by the `ripgrep` command. You can install it by following the instructions in the [ripgrep repository](https://github.com/BurntSushi/ripgrep#installation). For example, on macOS you can use Homebrew: `brew install ripgrep`

## Getting Started

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd tiny-trae
    ```

2.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Set up your Anthropic API key:**
    ```bash
    export ANTHROPIC_API_KEY="your-api-key"
    ```
    You can also set `ANTHROPIC_BASE_URL` if you are using a proxy.

4.  **Run the agent:**
    ```bash
    go run main.go
    ```
    or
    ```bash
    go build -o tiny-trae
    ```

## Usage

### Building the Project

1. Clone the repository:
    ```bash
    git clone <repository-url>
    cd tiny-trae
    ```

2. Build the project:
    ```bash
    go build -o tiny-trae
    ```

### Command Line Options

- `--profile <name>`: Select a configuration profile (default, minimal, closeai)
- `--list-profiles`: List all available profiles
- `-p "<prompt>"`: Run in non-interactive mode with a single prompt
- `--trace`: Enable tracing of model interactions
- `--trace-dir <directory>`: Directory to store trace files (default: ./traces)

### Interactive Mode

To run the agent in interactive mode, simply run the executable:

```bash
./tiny-trae
```

The agent will prompt you for input.

#### Navigation in Interactive Mode

- **Type messages**: Use the input box at the bottom to type your messages
- **Send messages**: Press `Enter` to send your message
- **Scroll up/down**: Use `↑`/`↓` arrow keys or `k`/`j` (vim-style) to scroll through message history
- **Page navigation**: Use `Page Up` and `Page Down` for faster scrolling
- **Jump to top/bottom**: Use `Home` to go to the beginning, `End` to go to the latest messages
- **Quit**: Press `q` or `Ctrl+C` to exit

### Non-interactive Mode

To run the agent in non-interactive mode, use the `-p` flag to provide a prompt:

```bash
./tiny-trae -p "your prompt here"
```

The agent will process the prompt and exit.

### Examples

```bash
# Interactive mode with default profile
./tiny-trae

# Non-interactive mode
./tiny-trae -p "Write a Python function to calculate fibonacci numbers"

# Using closeai profile
CLOSEAI_API_KEY=sk-your-key ./tiny-trae --profile closeai

# Using custom model with closeai
CLOSEAI_API_KEY=sk-your-key CLOSEAI_MODEL=claude-3-opus-20240229 ./tiny-trae --profile closeai

# Enable tracing
./tiny-trae --trace --profile default

# Custom trace directory
./tiny-trae --trace --trace-dir /path/to/traces

# List available profiles
./tiny-trae --list-profiles
```

## Using with OpenRouter

You can use this agent with [OpenRouter](https://openrouter.ai/) by using `anthropic-proxy`.

1.  **Start the proxy:**
    ```bash
    OPENROUTER_API_KEY=your-api-key COMPLETION_MODEL="anthropic/claude-sonnet-4" npx anthropic-proxy
    ```

2.  **Run the agent:**
    In a separate terminal, run the following command:
    ```bash
    ANTHROPIC_BASE_URL=http://0.0.0.0:3000 ./tiny-trae
    ```

## Using with CloseAI

You can also use this agent with CloseAI, which provides access to Claude models through their API.

1.  **Set your CloseAI API key:**
    ```bash
    export CLOSEAI_API_KEY=sk-your-closeai-api-key
    ```

2.  **Optionally configure the model (default: claude-sonnet-4-20250514):**
    ```bash
    export CLOSEAI_MODEL=claude-sonnet-4-20250514
    # Or use other available models like:
    # export CLOSEAI_MODEL=claude-3-opus-20240229
    # export CLOSEAI_MODEL=claude-3-haiku-20240307
    ```

3.  **Run the agent with closeai profile:**
    ```bash
    ./tiny-trae --profile closeai
    ```

    Or with a direct prompt:
    ```bash
    CLOSEAI_API_KEY=sk-your-closeai-api-key ./tiny-trae --profile closeai -p "Your prompt here"
    ```

    With custom model:
    ```bash
    CLOSEAI_API_KEY=sk-your-closeai-api-key CLOSEAI_MODEL=claude-3-opus-20240229 ./tiny-trae --profile closeai
    ```

The closeai profile supports configurable models via the `CLOSEAI_MODEL` environment variable and is optimized for CloseAI's API endpoint.

## How it works

The agent starts a conversation with the user. The user's message is sent to the Anthropic API, and the model can either respond with text or a request to use a tool. If it's a tool-use request, the agent executes the tool and sends the result back to the model. This loop continues until the user exits the program.

### Sub-Agent System

Tiny Trae features a sophisticated sub-agent system for handling complex, specialized tasks:

#### Codebase Search Agent

The `codebase_search` tool is powered by a specialized sub-agent that provides intelligent code discovery:

- **Multi-step Analysis**: Performs comprehensive code exploration using multiple search strategies
- **Tool Integration**: Uses `list_files`, `ripgrep`, and `read_file` tools in combination
- **Contextual Search**: Goes beyond simple keyword matching to understand code relationships
- **Smart Iteration**: Handles complex queries with up to 10 rounds of analysis
- **Graceful Fallback**: Provides partial results even when reaching iteration limits

**Example Usage:**
```bash
./tiny-trae -p "Use codebase_search to find authentication-related code"
```

#### Sub-Agent Features

- **Complete TUI Integration**: Sub-agent operations are fully visible in the terminal interface
- **Comprehensive Tracing**: All sub-agent LLM calls are recorded in separate trace files
- **Robust Error Handling**: Smart fallback mechanisms ensure useful results even with complex queries
- **Extensible Architecture**: Easy to add new specialized sub-agents for different domains

## Trace Recording

Tiny Trae supports detailed tracing of model interactions for debugging and analysis purposes.

### Enabling Tracing

Use the `--trace` flag to enable tracing:

```bash
./tiny-trae --trace
```

### Trace Output

When tracing is enabled, each model interaction creates a timestamped subdirectory in the traces folder containing:

- `request.json`: Complete API request data including messages, tools, and configuration
- `response.json`: API response data including the model's output and token usage
- `prompt.md`: Human-readable markdown representation of the entire interaction

#### Sub-Agent Tracing

Sub-agents generate their own separate trace files for each internal LLM interaction:

- **Main Agent Traces**: Show high-level tool calls to sub-agents (e.g., `codebase_search`)
- **Sub-Agent Traces**: Show detailed internal operations of each sub-agent
- **Complete Visibility**: Every LLM call, whether from main agent or sub-agent, is fully traced

This provides complete transparency into the multi-layered AI decision-making process.

### Trace Directory Structure

```
traces/
├── 20240115-143022-000000/
│   ├── request.json
│   ├── response.json
│   └── prompt.md
├── 20240115-143045-000000/
│   ├── request.json
│   ├── response.json
│   └── prompt.md
└── ...
```

### Custom Trace Directory

Specify a custom directory for traces:

```bash
./tiny-trae --trace --trace-dir /path/to/custom/traces
```

## Available Tools

The agent comes with several built-in tools:

### Core Tools

1. **`bash`**: Execute shell commands
   - Run any shell command and get the output
   - Useful for system operations, file management, and running scripts

2. **`read_file`**: Read file contents
   - Read the entire content of a specified file
   - Supports various file formats and encodings

3. **`edit_file`**: Edit files with search and replace
   - Modify files by replacing specific text patterns
   - Precise text replacement with exact matching

4. **`list_files`**: List directory contents
   - List all files and directories within a given path
   - Recursive directory traversal with filtering options

5. **`ripgrep`**: Search for patterns across files
   - Fast text search using regular expressions
   - Search across multiple files and directories
   - **Requirement**: Must have `ripgrep` installed (`brew install ripgrep` on macOS)

### Advanced Tools

6. **`codebase_search`**: Intelligent code search and discovery
   - **Powered by specialized sub-agent**: Uses AI to understand code context and relationships
   - **Multi-strategy search**: Combines file listing, pattern matching, and content analysis
   - **Conceptual understanding**: Goes beyond keyword matching to find functionally related code
   - **Smart iteration**: Performs up to 10 rounds of analysis for complex queries
   - **Graceful degradation**: Provides partial results even for complex or incomplete searches

### Tool Extensibility

You can extend the agent by:
- Adding new `ToolDefinition` structs in the `internal/tools` package
- Implementing the tool function with proper input/output handling
- Registering the tool in the appropriate profile configuration
- For complex tools, consider implementing them as sub-agents for better modularity

### Sub-Agent Architecture

The `codebase_search` tool demonstrates the sub-agent pattern:
- **Specialized Intelligence**: Dedicated AI agent for specific domain tasks
- **Tool Composition**: Sub-agents can use multiple core tools in combination
- **Independent Tracing**: Each sub-agent interaction is fully traced
- **TUI Integration**: Sub-agent operations are visible in the terminal interface
- **Robust Error Handling**: Smart fallback mechanisms ensure reliable operation
