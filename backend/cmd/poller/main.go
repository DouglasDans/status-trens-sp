// Command poller checks the sources on an interval and pushes status changes
// to the favorites of each line. Locally it loops on POLL_INTERVAL with an
// in-memory store; in Lambda each invocation runs one cycle (poll.Poller.Run)
// against the DynamoDB table named by LINES_TABLE.
//
// Environment:
//
//	FIREBASE_SERVICE_ACCOUNT_PARAM SSM SecureString holding the Firebase service account key (AWS)
//	FIREBASE_SERVICE_ACCOUNT_FILE  path to that key when there is no parameter (local; never in git)
//	POLL_INTERVAL                  Go duration between cycles (default 1m; local only)
//	NOTIFICATION_IMAGE_BASE_URL    public folder with the line badges (linha-<code>.png); empty sends none
//	LINES_TABLE                    DynamoDB table with the line memory (required in Lambda)
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

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

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

	// Lambda sets AWS_LAMBDA_FUNCTION_NAME: then each invocation runs one cycle
	// against DynamoDB, and EventBridge Scheduler supplies the interval.
	inLambda := os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != ""
	src := keySource{Param: os.Getenv("FIREBASE_SERVICE_ACCOUNT_PARAM"), File: os.Getenv("FIREBASE_SERVICE_ACCOUNT_FILE")}
	var cfg aws.Config
	if src.Param != "" || inLambda {
		var err error
		if cfg, err = config.LoadDefaultConfig(ctx); err != nil {
			logger.Error("aws config", "error", err)
			os.Exit(1)
		}
		src.SSM = ssm.NewFromConfig(cfg)
	}
	// Read once per process: in Lambda that is once per cold start.
	key, err := serviceAccountKey(ctx, src)
	if err != nil {
		logger.Error("firebase key", "error", err)
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

	client := &http.Client{Timeout: 10 * time.Second}
	p := &poll.Poller{
		Source: source.Fallback{
			source.NewProximoTrem(source.ProximoTremURL, client),
			source.NewCPTM(source.CPTMURL, client),
		},
		Notifier: fcm,
		Logger:   logger,
	}

	if inLambda {
		table := os.Getenv("LINES_TABLE")
		if table == "" {
			logger.Error("LINES_TABLE is required in Lambda")
			os.Exit(1)
		}
		p.Store = store.NewDynamoDB(dynamodb.NewFromConfig(cfg), table)
		lambda.Start(p.Run)
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

	p.Store = memory

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
