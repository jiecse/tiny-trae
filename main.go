package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"tiny-trae/internal/agent"
	"tiny-trae/internal/frontend"
	"tiny-trae/internal/profile"
	"tiny-trae/internal/trace"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// main is the entry point of the application.
// It initializes the Anthropic client, sets up the available tools,
// creates a new agent with a TUI frontend, and starts its execution.
// It supports both interactive and non-interactive modes.
// Any errors that occur during the agent's run are displayed in the TUI.
func main() {
	// Define command line flags
	promptFlag := flag.String("p", "", "Accept a string as user input")
	listProfilesFlag := flag.Bool("list-profiles", false, "List all available profiles")
	profileFlag := flag.String("profile", "default", "Specify which profile to use (default, coding, minimal, closeai)")
	traceFlag := flag.Bool("trace", false, "Enable tracing of model interactions")
	traceDir := flag.String("trace-dir", "./traces", "Directory to store trace files")
	flag.Parse()

	// Handle list profiles flag
	if *listProfilesFlag {
		profile.ListProfiles()
		return
	}

	var options []option.RequestOption
	
	// Check for closeai configuration first
	if closeaiKey := os.Getenv("CLOSEAI_API_KEY"); closeaiKey != "" {
		// Use closeai endpoint (SDK will append /v1/messages automatically)
		options = append(options, option.WithBaseURL("https://api.openai-proxy.org/anthropic"))
		// For closeai, use the API key as both x-api-key header and main auth
		options = append(options, option.WithAPIKey(closeaiKey))
		options = append(options, option.WithHeader("x-api-key", closeaiKey))
	} else {
		// Use standard anthropic configuration
		if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			options = append(options, option.WithAPIKey(apiKey))
		}
		if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
			options = append(options, option.WithBaseURL(baseURL))
		}
	}
	
	client := agent.NewClientWithOptions(options...)

	// Determine if running in interactive mode
	interactive := *promptFlag == ""
	var initialMessage string
	if *promptFlag != "" {
		initialMessage = *promptFlag
	}

	// Set up signal handler to ensure Ctrl+C always works
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println()
		os.Exit(0)
	}()

	// Create TUI frontend
	agentFrontend := frontend.NewTUIFrontend(interactive)
	defer agentFrontend.Close()

	// Select profile based on command line flag
	agentProfile := profile.GetProfileByName(*profileFlag)
	if agentProfile == nil {
		fmt.Printf("Error: Unknown profile '%s'. Use --list-profiles to see available profiles.\n", *profileFlag)
		os.Exit(1)
	}

	fmt.Printf("Using profile: %s\n", agentProfile.Name)

	// Create tracer
	tracer := trace.NewTracer(*traceDir)
	tracer.SetEnabled(*traceFlag)
	if *traceFlag {
		// Ensure trace directory exists
		if err := os.MkdirAll(*traceDir, 0755); err != nil {
			fmt.Printf("Warning: Failed to create trace directory: %v\n", err)
			tracer.SetEnabled(false)
		} else {
			absTraceDir, _ := filepath.Abs(*traceDir)
			fmt.Printf("Tracing enabled. Traces will be saved to: %s\n", absTraceDir)
		}
	}

	// Create agent with the selected frontend
	agentInstance := agent.NewAgent(client, agentProfile, agentFrontend, tracer)

	// Run the agent
	err := agentInstance.Run(context.TODO(), initialMessage)
	if err != nil {
		// This should only happen in non-interactive mode now
		// since interactive mode handles errors internally
		fmt.Fprintf(os.Stderr, "Agent error: %v\n", err)
		os.Exit(1)
	}
}
