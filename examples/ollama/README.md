# Ollama example

Probe models running locally via Ollama.

## Prerequisites

Install Ollama and pull a model:

```bash
ollama pull llama3.1
ollama pull mistral
```

Ollama serves an OpenAI-compatible API on port 11434 by default.

## Run

```bash
llmprobe probe -c examples/ollama/probes.yml
```

## Notes

- Ollama accepts any string as the API key (it does not authenticate).
- Model names must match what `ollama list` shows.
- Thresholds are generous here since Ollama runs on CPU or consumer GPUs.
- First probe after a cold start may be slow due to model loading. Subsequent probes measure steady-state performance.
