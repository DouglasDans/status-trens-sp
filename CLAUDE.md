# Status Trens SP

Android app that shows the live status of São Paulo metro and train lines and sends a push notification when a favorited line changes status. No login, no user accounts: favorites live on the device and the backend never knows who the user is.

Experimental portfolio project. Full product, architecture and decision docs live in Notion (page "🚆 Status Trens SP", subpages Arquitetura, Fontes de Dados, Decisões, Roadmap). **Check Decisões and Roadmap before proposing changes** — most trade-offs are already settled there.

## Current phase

**Phase 1 — Display.** Go backend running locally exposes `GET /status`; the app consumes it. No AWS, no poller, no push yet (those are Phases 2 and 3).

## Stack

| Layer | Choice |
|---|---|
| Backend | Go 1.26 (`backend/`, module `github.com/DouglasDans/status-trens-sp/backend`) |
| App | React Native + Expo, Android only (`app/`, created in Phase 1d) |
| Later | AWS Lambda `sa-east-1`, DynamoDB, EventBridge Scheduler, Firebase Cloud Messaging topics |

## Commands

Backend (run from `backend/`):

```sh
go test ./...   # all tests
go vet ./...    # static checks
```

Both must pass before any commit. App commands (`yarn test`, `yarn lint`) are added when `app/` exists.

## Structure

```
backend/
├── cmd/
│   ├── api/        # GET /status handler (Phase 1: local HTTP server)
│   └── poller/     # Phase 2
└── internal/
    ├── status/     # domain: canonical states, normalization, change detection — pure, no I/O
    ├── source/     # StatusSource interface + clients (proximotrem, cptm, diretodostrens)
    ├── store/      # Phase 2 (DynamoDB)
    └── notify/     # Phase 2 (FCM)
app/                # Expo app
```

Create packages only when the first test needs them.

## Backend conventions

- `internal/status` is pure domain: no HTTP, no AWS, no clock reads — pass time in as a value.
- External dependencies enter through small interfaces defined by the consumer (`StatusSource`, later `Store`, `Notifier`).
- Config via environment variables; secrets never in code or tests.
- Wrap errors with context: `fmt.Errorf("fetching proximotrem: %w", err)`. No `panic` outside `main`.
- Logging with `log/slog`.

## Testing

- TDD: test first, always.
- Table-driven tests for normalization and change detection.
- HTTP clients tested with `net/http/httptest` serving real recorded responses from `testdata/`.
- No test touches the real network or real AWS.

## Domain essentials

Canonical states: `normal`, `reduzida`, `parcial`, `paralisada`, `encerrada`, `desconhecido`. Unmapped source labels map to `desconhecido` and must be logged.

Data sources, in priority order:
1. **Próximo Trem** — `https://apim-proximotrem-prd-brazilsouth-001.azure-api.net/api/v1/lines`, no auth, 15 lines, no timestamp in the response.
2. **CPTM** — `http://apps.cptm.sp.gov.br:8080/AppMobileService/api/LinhasMetropolitanas`, no auth, 11 lines. Uses content negotiation: always send `Accept: application/json` (browsers get XML).
3. **Direto dos Trens** — requires a token (not available yet).

The app never calls a source directly; it only talks to our `GET /status`.
