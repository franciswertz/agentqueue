package httpapi

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"

	"github.com/franciswertz/agentqueue/internal/db"

    "github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
    DB *pgxpool.Pool
}

func (s *Server) Handler() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"ok":true}`))
    })

    mux.HandleFunc("/jobs/", func(w http.ResponseWriter, r *http.Request) {
        jobID := strings.TrimPrefix(r.URL.Path, "/jobs/")
        if jobID == "" {
            http.Error(w, "missing job id", http.StatusBadRequest)
            return
        }

        job, err := db.GetJob(r.Context(), s.DB, jobID)
        if err != nil {
            http.Error(w, "job not found", http.StatusNotFound)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(job)
    })

    return mux
}

func (s *Server) Start(ctx context.Context, addr string) error {
    server := &http.Server{Addr: addr, Handler: s.Handler()}

    go func() {
        <-ctx.Done()
        _ = server.Shutdown(context.Background())
    }()

    return server.ListenAndServe()
}
