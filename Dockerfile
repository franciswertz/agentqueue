FROM golang:1.22 AS builder

WORKDIR /workspace
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
ENV GOMAXPROCS=1
ENV GOFLAGS="-p=1"
COPY go.mod go.sum ./
RUN /usr/local/go/bin/go mod download

COPY . .
RUN /usr/local/go/bin/go build -o /out/agentq-worker ./cmd/worker

FROM ghcr.io/anomalyco/opencode:1.1.48 AS opencode

FROM alpine:3.20
RUN apk add --no-cache \
    bash \
    ca-certificates \
    libgcc \
    libstdc++ \
    ripgrep
COPY --from=opencode /usr/local/bin/opencode /usr/local/bin/opencode
ENV PATH="/usr/local/bin:${PATH}"
COPY --from=builder /out/agentq-worker /usr/local/bin/agentq-worker
COPY agents /root/.config/opencode/agents
COPY opencode.json /root/.config/opencode/opencode.json
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/agentq-worker"]
