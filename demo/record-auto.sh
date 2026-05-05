#!/usr/bin/env bash
set -euo pipefail

# Fully automated asciinema recording using a scripted session.
# Usage: cd llmprobe && bash demo/record-auto.sh
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
    rm -f demo/mockserver llmprobe
}
trap cleanup EXIT

echo "Recording..."
asciinema rec "$CAST_FILE" \
    --title "llmprobe: Probe LLM API endpoints" \
    --cols 100 \
    --rows 35 \
    --command "bash demo/demo-session.sh" \
    --overwrite

echo ""
echo "Done! Recording saved to $CAST_FILE"
echo ""
echo "Next steps:"
echo "  asciinema upload $CAST_FILE"
echo "  # or convert to GIF with agg:"
echo "  # agg $CAST_FILE demo/llmprobe-demo.gif"
