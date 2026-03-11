# AGENTS.md

This file guides agentic coding tools working in this repo.
Keep changes small, follow existing patterns, and avoid adding secrets.

## Repository summary
- Go worker that processes MQTT jobs and stores results in Postgres.
- Uses `opencode` as a subprocess to run prompts.
- Optional Playwright MCP sidecar for browsing tasks.

## Layout
- `cmd/worker`: entrypoint.
- `internal/`: app packages (config, db, mqtt, runner, worker, httpapi, types).
- `db/migrations`: SQL migrations.
- `deploy/`: Kubernetes manifests and kustomize base/overlays.
- `scripts/`: local helper scripts.
- `agents/`: opencode agent definitions.

## Build, lint, test
All commands are run from repo root.

### Build
- Build worker: `go build ./cmd/worker`
- Build all packages: `go build ./...`
- Docker image: `docker build -t agentq-worker:local .`
- Playwright MCP image: `docker build -t agentq-playwright-mcp:local -f Dockerfile.playwright-mcp .`

### Lint / format
- Format (all Go): `gofmt -w ./cmd ./internal`
- Vet: `go vet ./...`
- Staticcheck: not configured
- Lint runner: not configured

### Tests
- Run all tests (none currently): `go test ./...`
- Run a single test: `go test ./internal/<pkg> -run TestName -count=1`
- There are no `_test.go` files yet; add tests when needed.

### Local run
- Ensure Postgres is available and migrations are applied.
- Example local compose: `docker compose up --build`
- Example job producer: `./scripts/test-job.sh`

## Kubernetes
- Base kustomize: `kubectl apply -k deploy`
- Example overlay: `kubectl apply -k deploy/overlays/example`
- Secrets are intentionally not in source control.
- Example secrets live in `deploy/*.example.yaml` and must be created separately.

## Environment configuration
- `.env` is ignored; use `.env.example` as a template.
- `data/` is ignored except `data/.gitkeep`.
- Place opencode auth at `data/auth.json` (do not commit).
- `docker-compose.yml` mounts `./data/auth.json` into the container.

## Cursor / Copilot rules
- No `.cursor/rules`, `.cursorrules`, or `.github/copilot-instructions.md` were found.

## Code style guidelines

### Formatting
- Use `gofmt` for all Go files.
- Keep lines reasonably short; rely on gofmt for wrapping.

### Imports
- Group in this order, separated by blank lines:
  1) Standard library
  2) Internal packages (`agentq/...`)
  3) Third‑party packages
- Use explicit aliases only when needed.

### Types and structs
- Exported types use PascalCase; unexported use camelCase.
- JSON payloads use `map[string]any` for raw/unknown fields.
- Add JSON tags with snake_case to match payloads (e.g., `job_id`).

### Naming
- Functions: verbs (e.g., `Start`, `Run`, `Publish`).
- Variables: concise but clear (`cfg`, `pool`, `job`, `result`).
- Errors: `ErrSomething` for exported, `err` for locals.

### Error handling
- Return errors up the stack where possible.
- Use `fmt.Errorf("context: %w", err)` when adding context.
- Error strings are lowercase, no trailing punctuation.
- Log and continue only when failure is non‑fatal.

### Logging
- Use `log.Printf` / `log.Fatalf` (current pattern).
- Include short context prefix in messages (e.g., `"db connect: %v"`).

### Context and timeouts
- Accept `context.Context` in long‑running operations.
- Use `context.WithTimeout` for external calls or long work.
- Always `defer cancel()` when creating derived contexts.

### Concurrency
- Avoid goroutine leaks; tie goroutines to context or shutdown signals.
- Prefer channels for graceful shutdown; see `Worker.Shutdown`.

### Database
- Use pgxpool from `internal/db` helpers.
- Close pools on shutdown.
- Keep SQL in `db/migrations` only.

### MQTT
- Use wrapper in `internal/mqtt`.
- Publish with configured QoS and topic templates.
- Validate required job fields before insert.

### Opencode runner
- Keep stdout/stderr capped to avoid OOM.
- Add new flags via `OPENCODE_ARGS` unless required by code.
- The prompt is passed as a positional argument.

## Adding new dependencies
- Update `go.mod` and `go.sum` via `go get` or `go mod tidy`.
- Prefer standard library over new deps.

## Security / secrets
- Never commit real credentials, auth tokens, or cookies.
- Keep secrets in local files or external secret managers.
- Use `*.example` for public templates only.

## Documentation
- No README is present by design for now.
- If you add docs, keep them minimal and public‑safe.

## Common gotchas
- `OPENCODE_ARGS` is split with `strings.Fields`; quote carefully.
- `MQTT_CLIENT_ID` adds hostname when it ends with `-`.
- `JOB_LOCK_TIMEOUT` is used by DB code; keep defaults consistent.

## When changing code
- Update env defaults in `internal/config/config.go` when adding vars.
- Ensure new fields are plumbed from config to worker/runner.
- Keep config names stable; prefer additive changes.
