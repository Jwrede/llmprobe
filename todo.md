# llmprobe TODO

Current status:

- Latest release is `v1.3.0`.
- Documentation cleanup for `v1.3.0` is committed and pushed.
- Clean install from the public release was smoke-tested.
- MCP was smoke-tested with the freshly installed binary.
- Claude marketplace submission was updated to `v1.3.0`.
- `go test ./...` passes.

## Done

- [x] Commit and push the documentation cleanup.
- [x] Smoke-test a clean install from the public release. (v1.3.0 confirmed)
- [x] Smoke-test MCP with the exact freshly installed binary. (all 4 tools working)
- [x] Follow up on the Claude marketplace submission. (updated to v1.3.0)

## Optional Later

- [ ] Add a short troubleshooting doc if users hit repeated setup issues.
  - wrong `base_url`.
  - local server not exposing `/v1/chat/completions`.
  - model name mismatch.
  - missing API key env var.
  - first probe slow due to cold model load.

## Scope Note

Keep `llmprobe` scoped as a black-box endpoint probe. Deployment orchestration belongs in `inference-readiness-kit`.
