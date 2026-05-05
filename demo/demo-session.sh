#!/usr/bin/env bash
# Simulated interactive session for asciinema recording.
# Types commands with realistic delays to look natural.
# Expects: llmprobe binary in cwd, mock server running on :9119,
# and probes.yml containing the demo config (the record script handles this).

type_cmd() {
    local display="$1"
    local actual="${2:-$1}"
    printf '\n$ '
    for (( i=0; i<${#display}; i++ )); do
        printf '%s' "${display:$i:1}"
        sleep 0.04
    done
    printf '\n'
    sleep 0.3
    eval "$actual"
    sleep 1.5
}

clear
printf '# llmprobe: Probe LLM API endpoints\n'
printf '# Measure TTFT, latency, throughput. Single binary, zero SDKs.\n\n'
sleep 2

type_cmd 'cat probes.yml' 'cat demo/probes-display.yml'

type_cmd 'llmprobe probe' './llmprobe probe -c demo/probes.yml'

type_cmd 'llmprobe probe -f json' './llmprobe probe -c demo/probes.yml -f json'

type_cmd 'llmprobe probe --fail-on degraded; echo "exit code: $?"' \
    './llmprobe probe -c demo/probes.yml --fail-on degraded; echo "exit code: $?"'

printf '\n# Use in CI: llmprobe probe --fail-on degraded\n'
printf '# Blocks deploys when your LLM provider is slow.\n'
sleep 3
