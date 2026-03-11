#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="agentq"
TAG="${TAG:-latest}"
KUBE_BIN="${KUBE_BIN:-kubectl}"
REGISTRY="${REGISTRY:-}"
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH_IMAGES="${PUSH_IMAGES:-false}"

if ! command -v "$KUBE_BIN" >/dev/null 2>&1; then
  echo "kubectl not available. Set KUBE_BIN or ensure kubectl is on PATH." >&2
  exit 1
fi

WORKER_IMAGE="agentq-worker:${TAG}"
MCP_IMAGE="agentq-playwright-mcp:${TAG}"

if [ -n "$REGISTRY" ]; then
  WORKER_IMAGE="${REGISTRY}/agentq-worker:${TAG}"
  MCP_IMAGE="${REGISTRY}/agentq-playwright-mcp:${TAG}"
fi

BUILD_FLAGS=(--platform "$PLATFORM" -t "$WORKER_IMAGE")
MCP_BUILD_FLAGS=(--platform "$PLATFORM" -t "$MCP_IMAGE" -f Dockerfile.playwright-mcp)

if [ "$PUSH_IMAGES" = "true" ]; then
  BUILD_FLAGS+=(--push)
  MCP_BUILD_FLAGS+=(--push)
fi

docker buildx build "${BUILD_FLAGS[@]}" .
docker buildx build "${MCP_BUILD_FLAGS[@]}" .

"$KUBE_BIN" apply -f deploy/namespace.yaml

"$KUBE_BIN" apply -f deploy/opencode-config-configmap.yaml
"$KUBE_BIN" apply -f deploy/playwright-mcp-deployment.yaml
"$KUBE_BIN" apply -f deploy/playwright-mcp-service.yaml

"$KUBE_BIN" apply -f deploy/migration-configmap.yaml
"$KUBE_BIN" apply -f deploy/migration-job.yaml

"$KUBE_BIN" apply -f deploy/worker-deployment.yaml
"$KUBE_BIN" apply -f deploy/worker-service.yaml

"$KUBE_BIN" -n "$NAMESPACE" get pods
