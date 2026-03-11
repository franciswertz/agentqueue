package worker

import (
    "context"
    "encoding/json"
    "errors"
    "log"
    "strings"
    "time"

	"github.com/franciswertz/agentqueue/internal/config"
	"github.com/franciswertz/agentqueue/internal/db"
	"github.com/franciswertz/agentqueue/internal/mqtt"
	"github.com/franciswertz/agentqueue/internal/runner"
	"github.com/franciswertz/agentqueue/internal/types"

    mqttpaho "github.com/eclipse/paho.mqtt.golang"
    "github.com/jackc/pgx/v5/pgxpool"
)

type Worker struct {
    Config   config.Config
    MQTT     *mqtt.Client
    DB       *pgxpool.Pool
    Runner   runner.OpenCodeRunner
    Shutdown chan struct{}
}

func (w *Worker) Start(ctx context.Context) error {
    if err := w.MQTT.Subscribe(w.Config.MQTTEnqueueTopic, w.Config.MQTTQoS, w.handleEnqueue()); err != nil {
        return err
    }

    go w.runLoop(ctx)
    return nil
}

func (w *Worker) handleEnqueue() mqttpaho.MessageHandler {
    return func(client mqttpaho.Client, message mqttpaho.Message) {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        job, raw, err := parseJobRequest(message.Payload())
        if err != nil {
            log.Printf("invalid job payload: %v", err)
            return
        }

        if job.AppID == "" {
            job.AppID = appIDFromTopic(message.Topic())
            raw["app_id"] = job.AppID
        }

        if job.CallbackTopic == "" {
            job.CallbackTopic = renderTopic(w.Config.MQTTCompleteTopic, job)
            raw["callback_topic"] = job.CallbackTopic
        }

        if job.JobID == "" || job.Prompt == "" || job.AppID == "" {
            log.Printf("missing required job fields")
            return
        }

        payloadBytes, err := json.Marshal(raw)
        if err != nil {
            log.Printf("marshal job payload failed: %v", err)
            return
        }

        if err := db.InsertJob(ctx, w.DB, job, payloadBytes); err != nil {
            log.Printf("insert job failed: %v", err)
            return
        }
    }
}

func (w *Worker) runLoop(ctx context.Context) {
    ticker := time.NewTicker(w.Config.WorkerPollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-w.Shutdown:
            return
        case <-ticker.C:
            if err := w.processOnce(ctx); err != nil {
                continue
            }
        }
    }
}

func (w *Worker) processOnce(ctx context.Context) error {
    record, err := db.ClaimNextJob(ctx, w.DB)
    if err != nil {
        log.Printf("claim job failed: %v", err)
        return err
    }
    if record == nil {
        return nil
    }

    job, err := decodeJob(record.Payload)
    if err != nil {
        log.Printf("decode job failed: %v", err)
        _ = db.FailJob(ctx, w.DB, record.JobID, err.Error(), record.Attempts, w.Config.MaxAttempts)
        return err
    }

    result, err := w.Runner.Run(ctx, job)
    if err != nil {
        log.Printf("job run failed: %v", err)
        _ = db.FailJob(ctx, w.DB, record.JobID, err.Error(), record.Attempts, w.Config.MaxAttempts)
        return err
    }

    resultBytes, err := json.Marshal(result)
    if err != nil {
        log.Printf("marshal result failed: %v", err)
        _ = db.FailJob(ctx, w.DB, record.JobID, err.Error(), record.Attempts, w.Config.MaxAttempts)
        return err
    }

    if err := db.CompleteJob(ctx, w.DB, record.JobID, resultBytes); err != nil {
        log.Printf("complete job failed: %v", err)
        return err
    }

    completionTopic := job.CallbackTopic
    if completionTopic == "" {
        completionTopic = renderTopic(w.Config.MQTTCompleteTopic, job)
    }

    if err := w.MQTT.Publish(completionTopic, w.Config.MQTTQoS, false, resultBytes); err != nil {
        log.Printf("publish completion failed: %v", err)
    }
    return nil
}

func parseJobRequest(payload []byte) (types.JobRequest, map[string]any, error) {
    var raw map[string]any
    if err := json.Unmarshal(payload, &raw); err != nil {
        return types.JobRequest{}, nil, err
    }

    var job types.JobRequest
    if err := json.Unmarshal(payload, &job); err != nil {
        return types.JobRequest{}, nil, err
    }
    job.Raw = raw

    return job, raw, nil
}

func decodeJob(payload []byte) (types.JobRequest, error) {
    var raw map[string]any
    if err := json.Unmarshal(payload, &raw); err != nil {
        return types.JobRequest{}, err
    }
    var job types.JobRequest
    if err := json.Unmarshal(payload, &job); err != nil {
        return types.JobRequest{}, err
    }
    job.Raw = raw
    return job, nil
}

func appIDFromTopic(topic string) string {
    parts := strings.Split(topic, "/")
    if len(parts) < 3 {
        return ""
    }
    if parts[0] != "jobs" || parts[1] != "enqueue" {
        return ""
    }
    return parts[2]
}

func renderTopic(template string, job types.JobRequest) string {
    if template == "" {
        return ""
    }
    output := strings.ReplaceAll(template, "{app_id}", job.AppID)
    output = strings.ReplaceAll(output, "{job_id}", job.JobID)
    return output
}

var ErrInvalidJob = errors.New("invalid job")
