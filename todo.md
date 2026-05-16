# llmprobe TODO

This file is intentionally scoped to `llmprobe` as it exists today. The project
is a black-box LLM endpoint probe: it sends streaming requests, measures user
visible health signals, and returns results for CLI, CI, TUI, JSONL, and MCP
workflows.

Do not turn this repository into a Kubernetes, KServe, vLLM, SGLang, or llm-d
operations suite. That work should live in a separate inference-readiness repo
that consumes `llmprobe` JSONL as one input.

## Current Status

- CLI commands:
  - `probe`: one-off endpoint health check with CI gate behavior.
  - `watch`: repeated probing with table summaries, JSONL output, optional TUI, and Prometheus metrics.
  - `report`: generate Markdown percentile summary from JSONL data.
  - `baseline`: create baseline.json from JSONL for multiplier-based regression detection.
  - `mcp`: stdio MCP server exposing `probe_all`, `probe_model`, `list_providers`, and `get_config`.
- Providers:
  - OpenAI-compatible chat completions via the `openai` provider, optional `base_url`, and `label` field.
  - Anthropic, Google Gemini, Azure OpenAI, and AWS Bedrock.
- Measurements:
  - TTFT from request start to first content token.
  - total latency until stream end.
  - output tokens from usage metadata when available, otherwise streaming event count.
  - tokens/sec computed from `token_count / (latency - ttft)`.
  - status from configured thresholds (absolute or baseline multiplier): `healthy`, `degraded`, `error`.
- Outputs:
  - human table output.
  - JSON array for `probe -f json`.
  - JSONL for `watch -f json`.
  - TUI with TTFT chart and summary stats.
  - Prometheus /metrics endpoint.
  - Markdown report with p50/p95/p99.
- Tests:
  - config parsing/defaults/env expansion/label/duplicates.
  - SSE parsing.
  - OpenAI, Anthropic, Google, Azure, SigV4, AWS event stream pieces.
  - integration test covering all provider paths with mock servers.
  - report parsing and percentile calculations.
  - baseline create/save/load.
  - Prometheus metric recording.
- Current verification:
  - `go test ./...` passes.

## Scope Boundary

Keep in `llmprobe`:

- black-box endpoint probing.
- OpenAI-compatible endpoint examples.
- stable JSON/JSONL schema.
- CI gate behavior.
- endpoint-health reporting.
- probe-result Prometheus or OpenTelemetry export.
- small Markdown reports derived from `llmprobe` results.

Keep out of `llmprobe`:

- KServe install manifests.
- vLLM or SGLang server deployments.
- Gateway API or Envoy configuration.
- scraping model-server `/metrics`.
- Grafana dashboards for server internals.
- GPU capacity planning.
- llm-d routing experiments.

Those belong in a separate repo such as `inference-readiness-kit`.

## Completed

- [x] P0: Fix repeated OpenAI-compatible provider blocks (label field + duplicate detection).
- [x] P0: Preserve result order (index-based slice).
- [x] P0: Validate --format and --fail-on flag values.
- [x] P0: Handle "successful HTTP but no content tokens" as degraded.
- [x] P0: HTTP error messages include sanitized body snippets.
- [x] P1: Add examples/vllm/, examples/sglang/, examples/ollama/.
- [x] P1: Document JSONL as stable interface (docs/jsonl-schema.md).
- [x] P1: Add `llmprobe report` with p50/p95/p99 percentiles.
- [x] P2: Prometheus metrics export (--prometheus flag on watch).
- [x] P2: Baseline tracking with multipliers.
- [x] P2: Improved CI failure output (failing endpoints only).

## P2: OpenTelemetry Export

- [ ] Add OpenTelemetry export.
  - Emit one span per probe or metrics per probe result.
  - Capture endpoint label, model, status, TTFT, latency, token count, tokens/sec.
  - Do not implement full GenAI tracing for application calls.

## P3: MCP Hardening

- [ ] Add tests around MCP handlers.
  - `probe_all` with a temp config.
  - `probe_model` missing provider/model/api key env.
  - `get_config` does not leak raw secrets.

- [ ] Extend `probe_model` for OpenAI-compatible endpoints.
  - Add optional `base_url`.
  - Add optional `label`.
  - Keep the required fields simple.

## P3: Polish

- [ ] Histogram-based Prometheus metrics for multi-instance aggregation.
- [ ] Structured output validation: verify JSON mode responses parse correctly.
