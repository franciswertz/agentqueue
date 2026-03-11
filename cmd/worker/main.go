package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

	"github.com/franciswertz/agentqueue/internal/config"
	"github.com/franciswertz/agentqueue/internal/httpapi"
	"github.com/franciswertz/agentqueue/internal/mqtt"
	"github.com/franciswertz/agentqueue/internal/runner"
	"github.com/franciswertz/agentqueue/internal/worker"

    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    cfg := config.Load()

    pool, err := pgxpool.New(ctx, cfg.DBDSN)
    if err != nil {
        log.Fatalf("db connect: %v", err)
    }
    defer pool.Close()

    mqttClient, err := mqtt.New(mqtt.Config{
        BrokerURL: cfg.MQTTBrokerURL,
        Username:  cfg.MQTTUsername,
        Password:  cfg.MQTTPassword,
        ClientID:  cfg.MQTTClientID,
    })
    if err != nil {
        log.Fatalf("mqtt connect: %v", err)
    }
    defer mqttClient.Close()

	runner := runner.OpenCodeRunner{
		Cmd:     cfg.OpenCodeCmd,
		Args:    cfg.OpenCodeArgs,
		Timeout: cfg.OpenCodeTimeout,
		Dir:     cfg.OpenCodeDir,
		MaxOutputBytes: int64(cfg.OpenCodeMaxOutputBytes),
		DebugMem: cfg.OpenCodeDebugMem,
		DebugArgs: cfg.OpenCodeDebugArgs,
	}

    work := &worker.Worker{
        Config:   cfg,
        MQTT:     mqttClient,
        DB:       pool,
        Runner:   runner,
        Shutdown: make(chan struct{}),
    }

    if err := work.Start(ctx); err != nil {
        log.Fatalf("worker start: %v", err)
    }

    api := &httpapi.Server{DB: pool}
    go func() {
        if err := api.Start(ctx, cfg.HTTPAddr); err != nil {
            log.Printf("http server stopped: %v", err)
        }
    }()

    signals := make(chan os.Signal, 1)
    signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
    <-signals
    close(work.Shutdown)
    cancel()
}
