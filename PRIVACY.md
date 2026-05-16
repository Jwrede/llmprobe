# Privacy Policy

llmprobe runs entirely on your local machine. It does not collect, store, or transmit any data to third parties.

## What llmprobe does

- Sends HTTP requests to LLM API endpoints that you explicitly configure in probes.yml.
- Measures response timing (TTFT, latency, throughput) locally.
- Outputs results to your terminal, local files, or a local Prometheus endpoint.

## What llmprobe does not do

- Does not phone home or send telemetry.
- Does not store API keys, responses, or probe results on any remote server.
- Does not share data with the plugin author or any third party.
- Does not access files outside its configuration and output paths.

## Data flow

```
Your machine -> configured LLM API endpoints (OpenAI, Anthropic, etc.)
Your machine <- streaming responses (measured locally, then discarded)
```

API keys remain in your environment variables or local config file. They are never logged or transmitted beyond the configured provider endpoints.

## MCP server mode

When running as an MCP server (`llmprobe mcp`), the tool responds only to requests from the local MCP host (Claude Code). No network connections are made other than to your configured LLM endpoints.

## Contact

For questions about this policy: https://github.com/Jwrede/llmprobe/issues
