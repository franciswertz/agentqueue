# AgentQueue (agentqueue)

AgentQueue is an async job processor for LLM workflows. It ingests jobs over MQTT, stores state and results in Postgres, executes jobs via the `opencode` CLI, and publishes completions back to MQTT. A lightweight HTTP status endpoint is available for health and job lookup.

## What it does
- Accepts JSON job requests on MQTT topics.
- Persists jobs in Postgres for durability and retries.
- Executes prompts using `opencode` as a subprocess.
- Publishes job results to MQTT and stores results in Postgres.
- Exposes HTTP endpoints for health and job status.

## Dependencies
- Go 1.22 (for local builds)
- Postgres (required)
- MQTT broker (required)
- `opencode` CLI (required, invoked by the worker)
- Docker (optional, for local compose)
- Kubernetes (optional, for cluster deployment)
- Playwright MCP sidecar (optional, only if your jobs use browser tools)

## Job schema
Minimal job request payload:

```json
{
  "job_id": "job_123",
  "app_id": "my_app",
  "prompt": "Summarize this...",
  "provider": "openai",
  "model": "gpt-5.1-codex-mini",
  "params": { "temperature": 0.2 }
}
```

Completions are published to `jobs/complete/{app_id}` (or `callback_topic` when provided) and include output, status, latency, and optional token usage.

## Local setup
1) Create opencode auth at `data/auth.json` (not committed). Example:

```json
{
  "providers": [
    { "provider": "openai", "api_key": "REPLACE_ME" }
  ]
}
```

2) Start services with Docker Compose:

```bash
docker compose up --build
```

3) Send a test job (requires Docker on host):

```bash
./scripts/test-job.sh
```

## Kubernetes setup
1) Create secrets (do not commit real values):

```bash
kubectl apply -f deploy/opencode-auth-secret.example.yaml
kubectl apply -f deploy/agentq-mqtt-secret.example.yaml
kubectl apply -f deploy/playwright-mcp-storage-secret.example.yaml
```

2) Apply the base manifests:

```bash
kubectl apply -k deploy
```

3) Optional: use the example kustomize overlay to set image registry/tag:

```bash
kubectl apply -k deploy/overlays/example
```

## Configuration
Configuration is via environment variables. See `.env.example` for defaults and expected variables. In Kubernetes, configure them via manifests or secrets.

## Status endpoints
- `GET /healthz`
- `GET /jobs/{job_id}`

## Notes
- This project is under active development; interfaces and defaults may change.
- Secrets and runtime data must remain out of source control.

## Instacart session recipe (for Playwright MCP)
Use this when you need Instacart agents and want to run authentication locally.

1) Install Playwright (once per machine):

```bash
npx playwright install chromium
```

2) Launch an interactive browser and authenticate, saving storage state:

```bash
npx playwright codegen https://www.instacart.com --save-storage=data/instacart-storage.json
```

3) Log in in the opened browser, then close the window. The session is saved to `data/instacart-storage.json`.

When you start the Playwright MCP container, it reads that file from `data/` and reuses the session.
