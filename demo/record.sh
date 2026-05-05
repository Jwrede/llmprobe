#!/usr/bin/env bash
set -euo pipefail

# Records an asciinema demo of llmprobe.
# Usage: cd llmprobe && bash demo/record.sh
#
# Prerequisites:
#   - asciinema installed
#   - Go installed (builds mock server and llmprobe on the fly)
#
# Output: demo/llmprobe-demo.cast

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CAST_FILE="$SCRIPT_DIR/llmprobe-demo.cast"

cd "$PROJECT_DIR"

echo "Building llmprobe..."
go build -o llmprobe .

echo "Building mock server..."
go build -o demo/mockserver demo/mockserver.go

echo "Starting mock server..."
demo/mockserver &
MOCK_PID=$!
sleep 1

cleanup() {
    kill "$MOCK_PID" 2>/dev/null || true
    wait "$MOCK_PID" 2>/dev/null || true
}
trap cleanup EXIT

echo "Recording to $CAST_FILE"
echo "Run these commands inside the recording session:"
echo ""
echo "  cat demo/probes.yml"
echo "  ./llmprobe probe -c demo/probes.yml"
echo "  ./llmprobe probe -c demo/probes.yml -f json"
echo "  exit"
echo ""

asciinema rec "$CAST_FILE" \
    --title "llmprobe: Probe LLM API endpoints" \
    --cols 100 \
    --rows 30

echo ""
echo "Recording saved to $CAST_FILE"
echo "Upload with: asciinema upload $CAST_FILE"
echo "Or embed: asciinema auth (then link to your account)"
