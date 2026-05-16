# llmprobe TODO

Current status:

- Latest release is `v1.3.0`.
- Local docs were updated after the release and are not committed yet.
- Uncommitted docs files:
  - `README.md`
  - `.claude/skills/llmprobe/SKILL.md`
  - `todo.md`
- `go test ./...` passes after the docs update.

## Next

- [ ] Commit and push the documentation cleanup.
  ```bash
  git add README.md .claude/skills/llmprobe/SKILL.md todo.md
  git commit -m "docs: update llmprobe v1.3 docs"
  git push
  ```

- [ ] Smoke-test a clean install from the public release.
  ```bash
  rm -rf /tmp/llmprobe-smoke
  mkdir -p /tmp/llmprobe-smoke
  GOBIN=/tmp/llmprobe-smoke go install github.com/Jwrede/llmprobe@latest
  /tmp/llmprobe-smoke/llmprobe version
  /tmp/llmprobe-smoke/llmprobe --help
  ```

- [ ] Smoke-test MCP with the exact freshly installed binary.
  ```bash
  claude mcp remove llmprobe
  claude mcp add --transport stdio llmprobe -- /tmp/llmprobe-smoke/llmprobe mcp
  ```
  Then verify `list_providers` or `probe_all` works against a small local `probes.yml`.

- [ ] Follow up on the Claude marketplace submission.
  - If the submitted entry predates `v1.3.0`, update the version/features if the process allows it.
  - If reviewers ask for changes, keep the scope limited to install, packaging, privacy, and skill clarity unless they identify a real bug.

## Later

- [ ] Add a short troubleshooting doc if users hit repeated setup issues.
  - wrong `base_url`.
  - local server not exposing `/v1/chat/completions`.
  - model name mismatch.
  - missing API key env var.
  - first probe slow due to cold model load.

- [ ] Keep `llmprobe` scoped as a black-box endpoint probe. Deployment orchestration belongs in `inference-readiness-kit`.

