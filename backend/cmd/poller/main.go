// Command poller checks the sources on an interval and pushes status changes
// to the favorites of each line. In Phase 2 it runs locally with an in-memory
// store; in Phase 3 the same cycle (poll.Poller.Run) moves to Lambda.
//
// Environment:
//
//	FIREBASE_SERVICE_ACCOUNT_FILE  path to the Firebase service account key (required, never in git)
//	POLL_INTERVAL                  Go duration between cycles (default 1m)
//	NOTIFICATION_IMAGE_BASE_URL    public folder with the line badges (linha-<code>.png); empty sends none
//
// With -send-test <line>, it sends one test push to that line's topic and exits.
// With -seed <line>=<state> (dev only), that line starts with the given state
// instead of a baseline, so the first real reading that differs is pushed:
// the whole cycle runs for real without waiting for an actual change.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/catalog"
	"github.com/DouglasDans/status-trens-sp/backend/internal/notify"
	"github.com/DouglasDans/status-trens-sp/backend/internal/poll"
	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
	"github.com/DouglasDans/status-trens-sp/backend/internal/store"
)

func main() {
	testLine := flag.Int("send-test", 0, "send one test push to this line's topic and exit")
	seed := flag.String("seed", "", "dev only: <line>=<state> assumed before the first poll, e.g. 3=paralisada")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	key, err := os.ReadFile(os.Getenv("FIREBASE_SERVICE_ACCOUNT_FILE"))
	if err != nil {
		logger.Error("reading FIREBASE_SERVICE_ACCOUNT_FILE", "error", err)
		os.Exit(1)
	}
	fcm, err := notify.NewFCM(ctx, key)
	if err != nil {
		logger.Error("firebase", "error", err)
		os.Exit(1)
	}
	fcm.ImageBaseURL = os.Getenv("NOTIFICATION_IMAGE_BASE_URL")

	if *testLine != 0 {
		d := catalog.Present(*testLine, "", "")
		msg := notify.Compose(*testLine, d.Name, d.ColorHex, "Teste", "Notificação de teste do Status Trens SP.")
		if err := fcm.Send(ctx, msg); err != nil {
			logger.Error("test push", "error", err)
			os.Exit(1)
		}
		logger.Info("test push sent", "line", *testLine)
		return
	}

	interval, err := time.ParseDuration(envOr("POLL_INTERVAL", "1m"))
	if err != nil {
		logger.Error("POLL_INTERVAL", "error", err)
		os.Exit(1)
	}

	memory := store.NewInMemory()
	if *seed != "" {
		code, state, err := parseSeed(*seed)
		if err != nil {
			logger.Error("-seed", "error", err)
			os.Exit(1)
		}
		_ = memory.Save(ctx, code, status.Memory{State: state, LastNotified: state})
		logger.Info("seeded", "line", code, "state", state)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	p := &poll.Poller{
		Source: source.Fallback{
			source.NewProximoTrem(source.ProximoTremURL, client),
			source.NewCPTM(source.CPTMURL, client),
		},
		Store:    memory,
		Notifier: fcm,
		Logger:   logger,
	}

	logger.Info("polling", "interval", interval)
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
