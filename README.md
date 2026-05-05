# llmprobe

> Probe LLM API endpoints. Measure TTFT, latency, throughput. Single binary, zero SDKs.

[![CI](https://github.com/Jwrede/llmprobe/actions/workflows/ci.yml/badge.svg)](https://github.com/Jwrede/llmprobe/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/go-1.23+-00ADD8?logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

llmprobe is a CLI tool that probes LLM API endpoints and measures the metrics
that matter for production reliability: time to first token (TTFT), total
latency, generation throughput (tokens/sec), and error rates.

Use it as a one-off health check, a continuous monitor, or a CI gate that
blocks deploys when your LLM provider is degraded.

![llmprobe](demo/tui-thumbnail.png?v=2)

![demo](demo/llmprobe-demo.gif)

## Quick start

```bash
go install github.com/Jwrede/llmprobe@latest
```

Create a `probes.yml` (or copy the included example):

```yaml
providers:
  - name: openai
    api_key: ${OPENAI_API_KEY}
    models:
      - name: gpt-4o
        thresholds:
          max_ttft: 2s
      - name: gpt-4o-mini
        thresholds:
          max_ttft: 500ms

  - name: anthropic
    api_key: ${ANTHROPIC_API_KEY}
    models:
      - name: claude-sonnet-4-20250514
        thresholds:
          max_ttft: 1s
```

Run a probe:

```
$ llmprobe probe

Provider   Model                    Status    TTFT    Latency  Tok/s  Tokens  Error
--------   -----                    ------    ----    -------  -----  ------  -----
openai     gpt-4o                   healthy   312ms   2100ms   68.4   42
openai     gpt-4o-mini              healthy   98ms    814ms    112.3  56
anthropic  claude-sonnet-4-20250514 healthy   420ms   2831ms   52.1   38
azure      gpt-4o                   healthy   289ms   1950ms   71.2   44
bedrock    anthropic.claude-3-5...  degraded  1820ms  4510ms   28.1   38

4 healthy, 1 degraded, 0 errors
```

## What it measures

| Metric | What it means |
|--------|--------------|
| **TTFT** | Time from request send to first content token. This is what users feel as "lag" before the response starts streaming. |
| **Latency** | Total time from request to stream close. |
| **Tok/s** | Generation throughput: tokens produced per second after the first token. Calculated as `token_count / (latency - ttft)`. |
| **Tokens** | Total output tokens. Prefers provider usage metadata when available, falls back to SSE event counting. |
| **Status** | `healthy` if all thresholds pass, `degraded` if any threshold is exceeded, `error` if the request failed. |

## Commands

### `llmprobe probe`

One-off health check. Probes all configured endpoints and prints results.

```bash
llmprobe probe                        # table output
llmprobe probe -f json                # JSON output
llmprobe probe --fail-on degraded     # exit 1 if any endpoint is degraded
llmprobe probe -c custom-config.yml   # custom config path
```

Exit codes for CI:

| `--fail-on` | Exit 0 | Exit 1 |
|-------------|--------|--------|
| `error` (default) | healthy or degraded | any error |
| `degraded` | healthy only | degraded or error |
| `none` | always | never |

### `llmprobe watch`

Continuous monitoring. Probes all endpoints on an interval and prints a
summary line per iteration.

```bash
llmprobe watch                          # default 60s interval
llmprobe watch --interval 30s           # custom interval
llmprobe watch --tui                    # live terminal dashboard with TTFT chart
llmprobe watch --tui --load data.jsonl  # load historical data into the dashboard
llmprobe watch -f json                  # JSONL output (one line per result)
```

The `--tui` flag launches a live terminal dashboard with a TTFT chart,
color legend, and statistics table. Use `--load` to import historical
JSONL data (from `llmprobe watch -f json > data.jsonl`).

```
$ llmprobe watch --interval 30s

Watching 4 endpoints every 30s (Ctrl+C to stop)

[14:01:02] All 4 endpoints healthy.
[14:01:32] All 4 endpoints healthy.
[14:02:02] 3 healthy, 1 degraded, 0 errors. DEGRADED: openai/gpt-4o (TTFT 1820ms)
[14:02:32] All 4 endpoints healthy.
```

## CI integration

Use `llmprobe probe` as a pre-deploy gate:

```yaml
# .github/workflows/deploy.yml
- name: Check LLM providers
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: |
    go install github.com/Jwrede/llmprobe@latest
    llmprobe probe --fail-on degraded
```

This blocks the deploy if any LLM provider is experiencing degraded
performance right now.

## Configuration

```yaml
defaults:
  prompt: "Hello"                                # probe prompt
  max_tokens: 20                                 # max output tokens
  timeout: 30s                                   # per-probe timeout
  concurrency: 5                                 # max parallel probes

providers:
  - name: openai                    # openai, anthropic, google, azure, bedrock
    api_key: ${OPENAI_API_KEY}      # env var expansion
    base_url: https://custom.api    # optional, override endpoint
    models:
      - name: gpt-4o
        prompt: "Say hello."        # override default prompt
        max_tokens: 10              # override default max_tokens
        thresholds:
          max_ttft: 2s              # alert if TTFT exceeds this
          max_latency: 10s          # alert if total latency exceeds this
          min_tokens_per_sec: 20    # alert if throughput drops below this

  - name: azure
    api_key: ${AZURE_OPENAI_API_KEY}
    base_url: https://your-resource.openai.azure.com
    api_version: "2024-10-21"       # optional, defaults to 2024-10-21
    models:
      - name: gpt-4o               # deployment name

  - name: bedrock
    access_key: ${AWS_ACCESS_KEY_ID}
    secret_key: ${AWS_SECRET_ACCESS_KEY}
    region: us-east-1
    models:
      - name: anthropic.claude-3-5-sonnet-20241022-v2:0
```

API keys and AWS credentials support `${ENV_VAR}` syntax. Only credential
fields are expanded, so env var references in prompts or model names are
left as-is.

### OpenAI-compatible providers

Many providers (Groq, Together AI, Fireworks, DeepSeek, Mistral, OpenRouter,
Ollama, vLLM) expose an OpenAI-compatible API. These work out of the box
by setting `base_url`:

```yaml
providers:
  # Groq
  - name: openai
    api_key: ${GROQ_API_KEY}
    base_url: https://api.groq.com/openai
    models:
      - name: llama-3.3-70b-versatile

  # DeepSeek
  - name: openai
    api_key: ${DEEPSEEK_API_KEY}
    base_url: https://api.deepseek.com
    models:
      - name: deepseek-chat

  # Together AI
  - name: openai
    api_key: ${TOGETHER_API_KEY}
    base_url: https://api.together.xyz
    models:
      - name: meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo

  # Local Ollama
  - name: openai
    api_key: unused
    base_url: http://localhost:11434/v1
    models:
      - name: llama3.2
```

## Architecture

```
probes.yml
  -> Config loader (YAML + env var expansion)
    -> Probe engine (concurrent goroutines per provider/model)
      -> Provider clients (raw HTTP + SSE parsing, no SDKs)
        -> Results (TTFT, latency, tokens/sec, status)
          -> Output (table, JSON, JSONL)
```

Each provider client is a thin HTTP wrapper that sends a streaming request
and parses the response. No LLM SDKs are imported. The SSE parser handles
both data-only events (OpenAI, Google) and named events (Anthropic). The
Bedrock client implements SigV4 signing and AWS binary event stream parsing
from scratch.

TTFT is measured from the moment the HTTP request is sent to the first
event that contains actual content text (not role assignments or metadata).

## Providers

| Provider | Endpoint | Auth | Streaming format |
|----------|----------|------|-----------------|
| OpenAI | `/v1/chat/completions` | `Authorization: Bearer` | SSE, `[DONE]` sentinel |
| Anthropic | `/v1/messages` | `x-api-key` header | named-event SSE |
| Google | `/v1beta/models/{model}:streamGenerateContent?alt=sse` | `key` query param | SSE |
| Azure OpenAI | `/openai/deployments/{model}/chat/completions` | `api-key` header | SSE, `[DONE]` sentinel |
| AWS Bedrock | `/model/{model}/converse-stream` | SigV4 | AWS binary event stream |
| OpenAI-compat | `/v1/chat/completions` (custom base_url) | `Authorization: Bearer` | SSE |

OpenAI-compatible covers: Groq, Together AI, Fireworks, DeepSeek, Mistral,
OpenRouter, Ollama, vLLM, and any endpoint that speaks the OpenAI chat
completions API.

## Roadmap

- Baseline tracking: store rolling percentiles, alert when current probe
  exceeds Nx baseline
- OpenTelemetry metric export for integration with Grafana/Datadog
- Prometheus `/metrics` endpoint
- Structured output validation: verify JSON mode responses parse correctly

## License

MIT
