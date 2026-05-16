# Release Checklist

## Before tagging

1. Update version in `cmd/version.go` (single source of truth).
2. Update version in `.claude-plugin/plugin.json`.
3. Run `go test ./...` and verify all pass.
4. Run `go build ./...` and verify no errors.
5. Verify `llmprobe version` prints the expected version.

## Tagging and release

6. Commit version bump: `git commit -m "release: vX.Y.Z"`
7. Tag: `git tag vX.Y.Z`
8. Push: `git push origin main --tags`
9. GoReleaser runs via GitHub Actions and publishes binaries.

## Post-release verification

10. Check GitHub releases page shows the new tag with binaries.
11. Verify install from tag:
    ```bash
    go install github.com/Jwrede/llmprobe@latest
    llmprobe version
    ```
12. Verify MCP server starts:
    ```bash
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{}}}' | llmprobe mcp
    ```
13. Register with Claude Code and test:
    ```bash
    claude mcp add --transport stdio llmprobe -- llmprobe mcp
    ```

## Marketplace

14. If Claude marketplace submission references this version, confirm the release exists first.
15. Update marketplace entry if version or feature set changed significantly.
