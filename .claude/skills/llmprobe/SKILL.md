---
description: Check LLM API endpoint health, measure TTFT, latency, and throughput. Use when deploying LLM-powered features, debugging slow AI responses, verifying provider SLAs, checking if an LLM API is down, or measuring inference performance before a deploy.
allowed-tools: Bash Read
---

# LLM Endpoint Health Check

Run llmprobe to check the health and performance of configured LLM API endpoints.

## Verify install

```bash
llmprobe version
```

## Quick probe (all configured endpoints)

```bash
llmprobe probe -f json -c probes.yml
```

## Probe a specific provider/model ad-hoc

If no probes.yml exists, create a temporary config:

```bash
llmprobe probe -f json -c <(cat <<'YAML'
providers:
  - name: openai
    api_key: ${OPENAI_API_KEY}
    models:
      - name: gpt-4o
        thresholds:
          max_ttft: 2s
YAML
)
```

For local OpenAI-compatible servers such as vLLM, SGLang, or Ollama, use
`name: openai` with `base_url` and a display `label`:

```bash
llmprobe probe -f json -c <(cat <<'YAML'
providers:
  - name: openai
    label: vllm-local
    api_key: unused
    base_url: http://localhost:8000
    models:
      - name: meta-llama/Llama-3.1-8B-Instruct
YAML
)
```

## Interpreting results

For each endpoint, report:
- **Status**: healthy, degraded, or error
- **TTFT**: Time to first token (what users feel as initial lag)
- **Latency**: Total request time
- **Tok/s**: Generation throughput
- **Errors**: Include the error message and suggest fixes

## Action guidance

- If status is **error**: check API key, network, provider status page
- If status is **degraded**: TTFT or latency exceeded threshold; consider fallback model or retry logic
- If all **healthy**: safe to deploy

## Generating reports from historical data

```bash
llmprobe report <jsonl-file>
```

This produces p50/p95/p99 percentile tables for TTFT, latency, and throughput.

## Continuous monitoring

```bash
llmprobe watch --interval 30s -f json >> probes.jsonl
llmprobe watch --prometheus :9090
llmprobe watch --otel localhost:4317
```

## Bounded watch (for scripting)

```bash
llmprobe watch --interval 5s --duration 60s -f json > probes.jsonl
llmprobe watch --interval 5s --count 10 -f json > probes.jsonl
```
