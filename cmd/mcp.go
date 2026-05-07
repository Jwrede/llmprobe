package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Jwrede/llmprobe/internal/config"
	"github.com/Jwrede/llmprobe/internal/probe"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start an MCP server over stdio",
	Long:  "Runs llmprobe as a Model Context Protocol server.\nExposes probe and check_model tools for LLM API health checking.",
	RunE:  runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

// probeArgs is the input schema for the probe tool.
type probeArgs struct {
	Config string `json:"config,omitempty" jsonschema:"path to probes.yml config file"`
}

// checkModelArgs is the input schema for the check_model tool.
type checkModelArgs struct {
	Provider  string `json:"provider" jsonschema:"provider name (openai, anthropic, google, azure, bedrock)"`
	Model     string `json:"model" jsonschema:"model identifier (e.g. gpt-4o, claude-sonnet-4-20250514)"`
	APIKeyEnv string `json:"api_key_env" jsonschema:"environment variable name containing the API key"`
}

func runMCP(cmd *cobra.Command, args []string) error {
	// All logging must go to stderr so stdout is reserved for MCP transport.
	log.SetOutput(os.Stderr)

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "llmprobe",
			Version: "1.0.0",
		},
		nil,
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "probe",
		Description: "Run a health check against configured LLM API endpoints. Returns TTFT, latency, throughput, and health status for each model defined in the config file.",
	}, handleProbe)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_model",
		Description: "Check a single model's health by probing it once. Returns TTFT, latency, throughput, and health status.",
	}, handleCheckModel)

	log.Println("Starting llmprobe MCP server on stdio")
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

func handleProbe(_ context.Context, _ *mcp.CallToolRequest, args probeArgs) (*mcp.CallToolResult, any, error) {
	cfgPath := args.Config
	if cfgPath == "" {
		cfgPath = "probes.yml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to load config %q: %v", cfgPath, err)), nil, nil
	}

	engine := probe.NewEngine(cfg)
	results, err := engine.RunAll()
	if err != nil {
		return errorResult(fmt.Sprintf("probe failed: %v", err)), nil, nil
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal results: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

func handleCheckModel(_ context.Context, _ *mcp.CallToolRequest, args checkModelArgs) (*mcp.CallToolResult, any, error) {
	if args.Provider == "" {
		return errorResult("provider is required"), nil, nil
	}
	if args.Model == "" {
		return errorResult("model is required"), nil, nil
	}
	if args.APIKeyEnv == "" {
		return errorResult("api_key_env is required"), nil, nil
	}

	apiKey := os.Getenv(args.APIKeyEnv)
	if apiKey == "" {
		return errorResult(fmt.Sprintf("environment variable %q is not set or empty", args.APIKeyEnv)), nil, nil
	}

	cfg := &config.Config{
		Defaults: config.Defaults{
			Prompt:      "Hello",
			MaxTokens:   20,
			Timeout:     config.Duration{Duration: 30 * time.Second},
			Concurrency: 1,
		},
		Providers: []config.Provider{
			{
				Name:   args.Provider,
				APIKey: apiKey,
				Models: []config.Model{
					{
						Name:      args.Model,
						Prompt:    "Hello",
						MaxTokens: 20,
					},
				},
			},
		},
	}

	engine := probe.NewEngine(cfg)
	results, err := engine.RunAll()
	if err != nil {
		return errorResult(fmt.Sprintf("probe failed: %v", err)), nil, nil
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal results: %v", err)), nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
