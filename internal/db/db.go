package db

import (
    "context"
    "encoding/json"
    "errors"
    "time"

	"github.com/franciswertz/agentqueue/internal/types"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

type JobRecord struct {
    JobID    string
    AppID    string
    Payload  json.RawMessage
    Attempts int
    State    string
}

func InsertJob(ctx context.Context, pool *pgxpool.Pool, job types.JobRequest, payload json.RawMessage) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO jobs (job_id, app_id, parent_job_id, state, payload, attempts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 0, NOW(), NOW())
		ON CONFLICT (job_id) DO NOTHING
	`, job.JobID, job.AppID, nullIfEmpty(job.ParentJobID), types.StateQueued, payload)
	return err
}

func ClaimNextJob(ctx context.Context, pool *pgxpool.Pool) (*JobRecord, error) {
    var record JobRecord
    tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return nil, err
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback(ctx)
        }
    }()

    row := tx.QueryRow(ctx, `
        SELECT job_id, app_id, payload, attempts, state
        FROM jobs
        WHERE state = $1
        ORDER BY created_at ASC
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    `, types.StateQueued)

    if scanErr := row.Scan(&record.JobID, &record.AppID, &record.Payload, &record.Attempts, &record.State); scanErr != nil {
        if errors.Is(scanErr, pgx.ErrNoRows) {
            _ = tx.Rollback(ctx)
            return nil, nil
        }
        err = scanErr
        return nil, scanErr
    }

    _, err = tx.Exec(ctx, `
        UPDATE jobs
        SET state = $1, started_at = NOW(), updated_at = NOW(), attempts = attempts + 1
        WHERE job_id = $2
    `, types.StateProcessing, record.JobID)
    if err != nil {
        return nil, err
    }

    if commitErr := tx.Commit(ctx); commitErr != nil {
        return nil, commitErr
    }

    record.Attempts = record.Attempts + 1
    record.State = types.StateProcessing
    return &record, nil
}

func CompleteJob(ctx context.Context, pool *pgxpool.Pool, jobID string, result json.RawMessage) error {
    _, err := pool.Exec(ctx, `
        UPDATE jobs
        SET state = $1, result = $2, completed_at = NOW(), updated_at = NOW()
        WHERE job_id = $3
    `, types.StateCompleted, result, jobID)
    return err
}

func FailJob(ctx context.Context, pool *pgxpool.Pool, jobID string, errMsg string, attempts int, maxAttempts int) error {
    state := types.StateFailed
    completedAt := time.Now()
    if attempts < maxAttempts {
        state = types.StateQueued
        completedAt = time.Time{}
    }

    if state == types.StateQueued {
        _, err := pool.Exec(ctx, `
            UPDATE jobs
            SET state = $1, error = $2, updated_at = NOW()
            WHERE job_id = $3
        `, state, errMsg, jobID)
        return err
    }

    _, err := pool.Exec(ctx, `
        UPDATE jobs
        SET state = $1, error = $2, completed_at = $3, updated_at = NOW()
        WHERE job_id = $4
    `, state, errMsg, completedAt, jobID)
    return err
}

func GetJob(ctx context.Context, pool *pgxpool.Pool, jobID string) (map[string]any, error) {
	row := pool.QueryRow(ctx, `
		SELECT job_id, app_id, parent_job_id, state, payload, result, error, attempts, created_at, updated_at, started_at, completed_at
		FROM jobs
		WHERE job_id = $1
	`, jobID)

    var (
		appID     string
		parentID  *string
		state     string
		payload   json.RawMessage
        result    *json.RawMessage
        errMsg    *string
        attempts  int
        createdAt time.Time
        updatedAt time.Time
        startedAt *time.Time
        completed *time.Time
    )

	if err := row.Scan(&jobID, &appID, &parentID, &state, &payload, &result, &errMsg, &attempts, &createdAt, &updatedAt, &startedAt, &completed); err != nil {
		return nil, err
	}

    response := map[string]any{
        "job_id":       jobID,
        "app_id":       appID,
		"state":        state,
		"payload":      json.RawMessage(payload),
		"attempts":     attempts,
		"created_at":   createdAt,
		"updated_at":   updatedAt,
		"started_at":   startedAt,
		"completed_at": completed,
	}

	if parentID != nil {
		response["parent_job_id"] = *parentID
	}

    if result != nil {
        response["result"] = json.RawMessage(*result)
    }
    if errMsg != nil {
        response["error"] = *errMsg
    }

	return response, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
