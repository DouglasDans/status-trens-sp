// Command api serves GET /status for the app. In Phase 1 it runs locally and
// fetches from the sources on demand (cached); in Phase 3 it moves to Lambda.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/httpapi"
	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
)

const cacheTTL = 30 * time.Second

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	client := &http.Client{Timeout: 10 * time.Second}
	src := source.Fallback{
		source.NewProximoTrem(source.ProximoTremURL, client),
		source.NewCPTM(source.CPTMURL, client),
	}
	svc := httpapi.NewStatusService(src, cacheTTL, time.Now, logger)

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewHandler(svc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "error", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
