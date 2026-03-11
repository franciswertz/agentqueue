package types

import "encoding/json"

type JobRequest struct {
    JobID         string                 `json:"job_id"`
    AppID         string                 `json:"app_id"`
    Prompt        string                 `json:"prompt"`
    Provider      string                 `json:"provider,omitempty"`
    Model         string                 `json:"model,omitempty"`
    Params        map[string]any         `json:"params,omitempty"`
    Tools         []map[string]any       `json:"tools,omitempty"`
	CallbackTopic string                 `json:"callback_topic,omitempty"`
	ParentJobID   string                 `json:"parent_job_id,omitempty"`
	Metadata      map[string]any         `json:"metadata,omitempty"`
	Raw           map[string]any         `json:"-"`
}

type JobResult struct {
    JobID     string          `json:"job_id"`
    AppID     string          `json:"app_id"`
    Status    string          `json:"status"`
    Output    json.RawMessage `json:"output,omitempty"`
    Error     string          `json:"error,omitempty"`
    Tokens    map[string]any  `json:"tokens,omitempty"`
    LatencyMS int64           `json:"latency_ms,omitempty"`
    TraceID   string          `json:"trace_id,omitempty"`
}

const (
    StateQueued     = "queued"
    StateProcessing = "processing"
    StateCompleted  = "completed"
    StateFailed     = "failed"
)
