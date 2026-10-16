// Command api serves GET /status for the app, fetching from the sources on
// demand (cached). Locally it is an HTTP server on PORT; in Lambda it answers
// the Function URL, and the cache lives as long as the warm instance.
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

	"github.com/aws/aws-lambda-go/lambda"

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

	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(functionURL(httpapi.NewHandler(svc)))
		return
	}

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
