// Command api serves GET /status for the app from the board the poller saves;
// it never calls a source on a request. In Lambda it answers the Function URL
// and reads the board from the DynamoDB table named by BOARD_TABLE. Locally it
// is an HTTP server on PORT that runs the poller's cycle itself every
// POLL_INTERVAL (default 1m), in memory and without pushes, so one process is
// enough for development.
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
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/DouglasDans/status-trens-sp/backend/internal/httpapi"
	"github.com/DouglasDans/status-trens-sp/backend/internal/notify"
	"github.com/DouglasDans/status-trens-sp/backend/internal/poll"
	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/store"
)

const cacheTTL = 30 * time.Second

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		table := os.Getenv("BOARD_TABLE")
		if table == "" {
			logger.Error("BOARD_TABLE is required in Lambda")
			os.Exit(1)
		}
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			logger.Error("aws config", "error", err)
			os.Exit(1)
		}
		boards := store.NewDynamoBoard(dynamodb.NewFromConfig(cfg), table)
		svc := httpapi.NewStatusService(boards, cacheTTL, time.Now, logger)
		lambda.Start(functionURL(httpapi.NewHandler(svc)))
		return
	}

	interval, err := time.ParseDuration(envOr("POLL_INTERVAL", "1m"))
	if err != nil {
		logger.Error("POLL_INTERVAL", "error", err)
		os.Exit(1)
	}
	boards := store.NewInMemoryBoard()
	go pollLocally(ctx, boards, interval, logger)

	addr := ":" + envOr("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.NewHandler(httpapi.NewStatusService(boards, cacheTTL, time.Now, logger)),
		ReadHeaderTimeout: 5 * time.Second,
	}

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

// pollLocally runs the poller's cycle on an interval, without pushes. Until
// the first cycle ends, GET /status answers 503.
func pollLocally(ctx context.Context, boards *store.InMemoryBoard, interval time.Duration, logger *slog.Logger) {
	p := &poll.Poller{
		Sources:  source.All(&http.Client{}, logger),
		Timeout:  source.Timeout,
		Store:    store.NewInMemory(),
		Boards:   boards,
		Notifier: noPush{},
		Logger:   logger,
		Now:      time.Now,
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := p.Run(ctx); err != nil {
			logger.Error("poll failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// noPush drops every notification: local development never pushes.
type noPush struct{}

func (noPush) Send(context.Context, notify.Message) error { return nil }

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
