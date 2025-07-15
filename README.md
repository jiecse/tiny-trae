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
- **Tool Execution:** The agent can execute the following tools:
    - `read_file`: Read the contents of a file.
    - `list_files`: List files and directories.
    - `edit_file`: Modify files by searching and replacing text.
    - `ripgrep`: Search for text patterns within files.
    - `bash`: Execute shell commands.
- **Extensible:** Easily add new tools to the agent.

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

1. **bash**: Execute shell commands
2. **read_file**: Read file contents
3. **edit_file**: Edit files with search and replace
4. **list_files**: List directory contents with filtering
5. **ripgrep**: Search for patterns across files

## Tools

The agent currently supports the following tools:

-   **`read_file`**: Reads the entire content of a specified file.
-   **`list_files`**: Lists all files and directories within a given path.
-   **`edit_file`**: Edits a file by replacing a specified string with a new one.
-   **`ripgrep`**: Searches for a pattern in files using `rg`.
-   **`bash`**: Executes a given command in a bash shell.

You can extend the agent by adding new `ToolDefinition` structs and including them in the `tools` slice in the `main` function.
