# SGLang example

Probe a local SGLang server via the OpenAI-compatible API.

## Prerequisites

Start SGLang:

```bash
python -m sglang.launch_server --model meta-llama/Llama-3.1-8B-Instruct --port 30000
```

## Run

```bash
export SGLANG_API_KEY=dummy

llmprobe probe -c examples/sglang/probes.yml
```

## Continuous monitoring

```bash
llmprobe watch -c examples/sglang/probes.yml --interval 30s --format json >> sglang.jsonl
```

## Notes

- SGLang defaults to port 30000.
- Model name must match what SGLang is serving.
- SGLang supports OpenAI chat completions with streaming, which is all llmprobe needs.
