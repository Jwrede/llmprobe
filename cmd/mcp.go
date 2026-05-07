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
	Long:  "Runs llmprobe as a Model Context Protocol server.\nExposes tools for LLM API health checking and configuration.",
	RunE:  runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

type probeAllArgs struct {
	Config string `json:"config,omitempty" jsonschema:"path to probes.yml config file"`
}

type probeModelArgs struct {
	Provider  string `json:"provider" jsonschema:"provider name (openai, anthropic, google, azure, bedrock)"`
	Model     string `json:"model" jsonschema:"model identifier (e.g. gpt-4o, claude-sonnet-4-20250514)"`
	APIKeyEnv string `json:"api_key_env" jsonschema:"environment variable name containing the API key"`
}

type listProvidersArgs struct {
	Config string `json:"config,omitempty" jsonschema:"path to probes.yml config file"`
}

type getConfigArgs struct {
	Config string `json:"config,omitempty" jsonschema:"path to probes.yml config file"`
}

func runMCP(cmd *cobra.Command, args []string) error {
	log.SetOutput(os.Stderr)

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "llmprobe",
			Version: "1.1.0",
		},
		nil,
	)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "probe_all",
		Description: "Probe all configured LLM API endpoints. Returns TTFT (ms), total latency (ms), throughput (tokens/sec), and health status for every model in the config file.",
	}, handleProbeAll)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "probe_model",
		Description: "Probe a single LLM model by provider and model name. Use this for ad-hoc checks without a config file. Returns TTFT (ms), total latency (ms), throughput (tokens/sec), and health status.",
	}, handleProbeModel)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_providers",
		Description: "List all providers and models defined in the config file. Returns provider names, model identifiers, and any configured thresholds. Use this to discover what models are available before probing.",
	}, handleListProviders)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_config",
		Description: "Return the full parsed configuration including defaults, providers, models, and thresholds. Useful for understanding the current probe setup or debugging configuration issues.",
	}, handleGetConfig)

	log.Println("Starting llmprobe MCP server on stdio")
	return server.Run(context.Background(), &mcp.StdioTransport{})
}

func handleProbeAll(_ context.Context, _ *mcp.CallToolRequest, args probeAllArgs) (*mcp.CallToolResult, any, error) {
	cfg, err := loadConfig(args.Config)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	engine := probe.NewEngine(cfg)
	results, err := engine.RunAll()
	if err != nil {
		return errorResult(fmt.Sprintf("probe failed: %v", err)), nil, nil
	}

	return jsonResult(results)
}

func handleProbeModel(_ context.Context, _ *mcp.CallToolRequest, args probeModelArgs) (*mcp.CallToolResult, any, error) {
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

	return jsonResult(results)
}

type providerSummary struct {
	Name   string         `json:"name"`
	Models []modelSummary `json:"models"`
}

type modelSummary struct {
	Name       string              `json:"name"`
	Thresholds *thresholdsSummary  `json:"thresholds,omitempty"`
}

type thresholdsSummary struct {
	MaxTTFT       string  `json:"max_ttft,omitempty"`
	MaxLatency    string  `json:"max_latency,omitempty"`
	MinTokensPerS float64 `json:"min_tokens_per_sec,omitempty"`
}

func handleListProviders(_ context.Context, _ *mcp.CallToolRequest, args listProvidersArgs) (*mcp.CallToolResult, any, error) {
	cfg, err := loadConfig(args.Config)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	var providers []providerSummary
	for _, p := range cfg.Providers {
		ps := providerSummary{Name: p.Name}
		for _, m := range p.Models {
			ms := modelSummary{Name: m.Name}
			if m.Thresholds.MaxTTFT.Duration > 0 || m.Thresholds.MaxLatency.Duration > 0 || m.Thresholds.MinTokensPerS > 0 {
				ms.Thresholds = &thresholdsSummary{
					MaxTTFT:       durationString(m.Thresholds.MaxTTFT.Duration),
					MaxLatency:    durationString(m.Thresholds.MaxLatency.Duration),
					MinTokensPerS: m.Thresholds.MinTokensPerS,
				}
			}
			ps.Models = append(ps.Models, ms)
		}
		providers = append(providers, ps)
	}

	return jsonResult(providers)
}

type configSummary struct {
	Defaults  config.Defaults   `json:"defaults"`
	Providers []providerConfig  `json:"providers"`
}

type providerConfig struct {
	Name       string         `json:"name"`
	BaseURL    string         `json:"base_url,omitempty"`
	APIVersion string         `json:"api_version,omitempty"`
	Region     string         `json:"region,omitempty"`
	HasAPIKey  bool           `json:"has_api_key"`
	Models     []config.Model `json:"models"`
}

func handleGetConfig(_ context.Context, _ *mcp.CallToolRequest, args getConfigArgs) (*mcp.CallToolResult, any, error) {
	cfg, err := loadConfig(args.Config)
	if err != nil {
		return errorResult(err.Error()), nil, nil
	}

	summary := configSummary{Defaults: cfg.Defaults}
	for _, p := range cfg.Providers {
		summary.Providers = append(summary.Providers, providerConfig{
			Name:       p.Name,
			BaseURL:    p.BaseURL,
			APIVersion: p.APIVersion,
			Region:     p.Region,
			HasAPIKey:  p.APIKey != "",
			Models:     p.Models,
		})
	}

	return jsonResult(summary)
}

func loadConfig(path string) (*config.Config, error) {
	if path == "" {
		path = "probes.yml"
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config %q: %v", path, err)
	}
	return cfg, nil
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal results: %v", err)), nil, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(data)},
		},
	}, nil, nil
}

func durationString(d time.Duration) string {
	if d == 0 {
		return ""
	}
	return d.String()
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
