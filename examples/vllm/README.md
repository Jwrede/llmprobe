# vLLM example

Probe a local vLLM server via the OpenAI-compatible API.

## Prerequisites

Start vLLM serving a model:

```bash
vllm serve meta-llama/Llama-3.1-8B-Instruct --port 8000
```

## Run

```bash
# vLLM does not require an API key by default, but the field must be non-empty
export VLLM_API_KEY=dummy

llmprobe probe -c examples/vllm/probes.yml
```

## Continuous monitoring

```bash
llmprobe watch -c examples/vllm/probes.yml --interval 30s --format json >> vllm.jsonl
```

## Notes

- Set `base_url` to wherever your vLLM instance listens.
- Model name must match the `--model` or `--served-model-name` flag used when starting vLLM.
- Adjust thresholds based on your hardware. The defaults here assume a single A100.
