# JSONL output schema

This document defines the stable JSONL format emitted by `llmprobe watch --format json`.
Each line is a self-contained JSON object representing one probe result.

## Schema

| Field | Type | Description |
|-------|------|-------------|
| `provider` | string | Provider or label name (e.g. "openai", "vllm-local") |
| `model` | string | Model identifier as configured in probes.yml |
| `status` | string | One of: `healthy`, `degraded`, `error` |
| `ttft_ms` | integer | Time to first token in milliseconds. 0 if status is error. |
| `latency_ms` | integer | Total request latency in milliseconds. 0 if status is error. |
| `tokens_per_sec` | number | Generation throughput (output tokens / generation time). 0 if error or generation time < 1ms. |
| `token_count` | integer | Number of output tokens. Uses provider usage metadata when available, falls back to chunk counting. |
| `error` | string | Error message. Omitted when status is healthy or degraded without error. |
| `timestamp` | string | ISO 8601 UTC timestamp (format: `2006-01-02T15:04:05Z`) |

## Example

```json
{"provider":"openai","model":"gpt-4o","status":"healthy","ttft_ms":120,"latency_ms":450,"tokens_per_sec":45.2,"token_count":18,"timestamp":"2025-01-15T10:00:00Z"}
{"provider":"openai","model":"gpt-4o","status":"error","ttft_ms":0,"latency_ms":0,"tokens_per_sec":0,"token_count":0,"error":"HTTP 429 from OpenAI: rate limited","timestamp":"2025-01-15T10:01:00Z"}
{"provider":"anthropic","model":"claude-sonnet-4-20250514","status":"degraded","ttft_ms":500,"latency_ms":1500,"tokens_per_sec":14.0,"token_count":17,"timestamp":"2025-01-15T10:02:00Z"}
```

## Stability guarantee

This schema is considered stable. Fields will not be removed or renamed. New fields may be added in the future but existing consumers should ignore unknown fields.

## Consuming JSONL

Generate a summary report:

```bash
llmprobe report data.jsonl
```

Parse with standard tools:

```bash
# Extract all errors
jq 'select(.status == "error")' data.jsonl

# Average TTFT per model
jq -s 'group_by(.model) | map({model: .[0].model, avg_ttft: (map(.ttft_ms) | add / length)})' data.jsonl
```

Feed into monitoring systems:

```bash
# Stream to a Prometheus pushgateway, Datadog agent, or any JSONL-consuming pipeline
llmprobe watch --format json | your-ingestion-tool
```
