package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/franciswertz/agentqueue/internal/types"
)

type OpenCodeRunner struct {
	Cmd     string
	Args    []string
	Timeout time.Duration
	Dir     string
	MaxOutputBytes int64
	DebugMem bool
	DebugArgs bool
}

type OpenCodeInput struct {
    JobID    string                 `json:"job_id"`
    AppID    string                 `json:"app_id"`
    Prompt   string                 `json:"prompt"`
    Provider string                 `json:"provider,omitempty"`
    Model    string                 `json:"model,omitempty"`
    Params   map[string]any         `json:"params,omitempty"`
    Tools    []map[string]any       `json:"tools,omitempty"`
    Meta     map[string]any         `json:"metadata,omitempty"`
    Raw      map[string]any         `json:"raw,omitempty"`
}

type OpenCodeOutput struct {
    Output json.RawMessage `json:"output"`
    Tokens map[string]any  `json:"tokens,omitempty"`
    Trace  string          `json:"trace_id,omitempty"`
}

type opencodeEvent struct {
    Type string `json:"type"`
    Part struct {
        Type     string                 `json:"type"`
        Text     string                 `json:"text"`
        Tokens   map[string]any          `json:"tokens"`
        Metadata map[string]any          `json:"metadata"`
    } `json:"part"`
}

func (r OpenCodeRunner) Run(ctx context.Context, job types.JobRequest) (types.JobResult, error) {
    if r.Cmd == "" {
        return types.JobResult{}, errors.New("opencode cmd is empty")
    }

    callCtx, cancel := context.WithTimeout(ctx, r.Timeout)
    defer cancel()

	args := append([]string{}, r.Args...)
	prompt := strings.TrimSpace(job.Prompt)
	if agentName, ok := parseAgentMention(prompt); ok && !hasFlag(args, "--agent") {
		args = append(args, "--agent", agentName)
		prompt = strings.TrimSpace(strings.TrimPrefix(prompt, "@"+agentName))
	}
	cmdDir := ""
	if r.Dir != "" {
		cmdDir = r.Dir
	}
	if job.Model != "" && !hasFlag(args, "--model") {
		modelValue := job.Model
		if job.Provider != "" && !strings.Contains(job.Model, "/") {
			modelValue = job.Provider + "/" + job.Model
		}
		args = append(args, "--model", modelValue)
	}
	if prompt == "" {
		return types.JobResult{}, errors.New("prompt is empty")
	}
	args = append(args, prompt)
	if r.DebugArgs {
		log.Printf("opencode args: %q", args)
		log.Printf("opencode prompt length: %d", len(prompt))
	}

	cmd := exec.CommandContext(callCtx, r.Cmd, args...)
	if cmdDir != "" {
		cmd.Dir = cmdDir
	}

	stdout := newCappedBuffer(r.MaxOutputBytes)
	stderr := newCappedBuffer(r.MaxOutputBytes)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

    start := time.Now()
    if err := cmd.Run(); err != nil {
		stderrText := strings.TrimSpace(stderr.String())
		stdoutText := strings.TrimSpace(stdout.String())
        if stderrText != "" {
            return types.JobResult{}, errors.New(stderrText)
        }
        if stdoutText != "" {
            return types.JobResult{}, errors.New(stdoutText)
        }
        return types.JobResult{}, err
    }

    latency := time.Since(start)

    result := types.JobResult{
        JobID:     job.JobID,
        AppID:     job.AppID,
        Status:    types.StateCompleted,
        LatencyMS: latency.Milliseconds(),
    }

	if parsed, ok := parseOpenCodeEvents(stdout.Bytes()); ok {
        result.Output = parsed.Output
        result.Tokens = parsed.Tokens
        result.TraceID = parsed.Trace
        return result, nil
    }

	outputFallback, err := json.Marshal(map[string]string{"text": stdout.String()})
    if err != nil {
        return types.JobResult{}, err
    }
    result.Output = outputFallback
    return result, nil
}

type cappedBuffer struct {
	buf       bytes.Buffer
	maxBytes  int64
	truncated bool
}

func newCappedBuffer(maxBytes int64) *cappedBuffer {
	return &cappedBuffer{maxBytes: maxBytes}
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.maxBytes <= 0 {
		return b.buf.Write(p)
	}
	remaining := b.maxBytes - int64(b.buf.Len())
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if int64(len(p)) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *cappedBuffer) Bytes() []byte {
	return b.buf.Bytes()
}

func (b *cappedBuffer) String() string {
	return b.buf.String()
}

func hasFlag(args []string, flag string) bool {
    for _, arg := range args {
        if arg == flag {
            return true
        }
    }
    return false
}

func parseAgentMention(prompt string) (string, bool) {
    trimmed := strings.TrimSpace(prompt)
    if !strings.HasPrefix(trimmed, "@") {
        return "", false
    }
    end := strings.IndexFunc(trimmed, func(r rune) bool {
        return r == ' ' || r == '\t' || r == '\n'
    })
    if end == -1 {
        end = len(trimmed)
    }
    name := strings.TrimPrefix(trimmed[:end], "@")
    if name == "" {
        return "", false
    }
    return name, true
}

func parseOpenCodeEvents(data []byte) (OpenCodeOutput, bool) {
    decoder := json.NewDecoder(bytes.NewReader(data))
    var texts []string
    var tokens map[string]any
    var trace string

    for {
        var ev opencodeEvent
        if err := decoder.Decode(&ev); err != nil {
            if errors.Is(err, io.EOF) {
                break
            }
            return OpenCodeOutput{}, false
        }
        if ev.Type == "text" && ev.Part.Text != "" {
            texts = append(texts, ev.Part.Text)
            if trace == "" {
                if metadata, ok := ev.Part.Metadata["openai"].(map[string]any); ok {
                    if itemID, ok := metadata["itemId"].(string); ok {
                        trace = itemID
                    }
                }
            }
        }
        if ev.Type == "step_finish" && ev.Part.Tokens != nil {
            tokens = ev.Part.Tokens
        }
    }

    if len(texts) == 0 && tokens == nil {
        return OpenCodeOutput{}, false
    }

    outputBytes, err := json.Marshal(map[string]string{"text": strings.Join(texts, "")})
    if err != nil {
        return OpenCodeOutput{}, false
    }

    return OpenCodeOutput{
        Output: outputBytes,
        Tokens: tokens,
        Trace:  trace,
    }, true
}
