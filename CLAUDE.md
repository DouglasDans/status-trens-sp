# Status Trens SP

Android app that shows the live status of São Paulo metro and train lines and sends a push notification when a favorited line changes status. No login, no user accounts: favorites live on the device and the backend never knows who the user is.

Experimental portfolio project. Full product, architecture and decision docs live in Notion (page "🚆 Status Trens SP", subpages Arquitetura, Fontes de Dados, Decisões, Roadmap). **Check Decisões and Roadmap before proposing changes** — most trade-offs are already settled there.

## Current phase

**Phase 1 — Display.** Go backend running locally exposes `GET /status`; the app consumes it. No AWS, no poller, no push yet (those are Phases 2 and 3).

## Stack

| Layer | Choice |
|---|---|
| Backend | Go 1.26 (`backend/`, module `github.com/DouglasDans/status-trens-sp/backend`) |
| App | React Native 0.86 + Expo SDK 57, TypeScript, Android only (`app/`) |
| Later | AWS Lambda `sa-east-1`, DynamoDB, EventBridge Scheduler, Firebase Cloud Messaging topics |

## Commands

Backend (run from `backend/`):

```sh
go test ./...        # all tests (use -race when touching httpapi)
go vet ./...         # static checks
go run ./cmd/api     # local API on :8080 (PORT overrides), GET /status

docker build -t status-trens-sp-api .                  # distroless image (~17 MB), non-root
docker run -d --name status-trens-sp-api -p 8080:8080 status-trens-sp-api
```

App (run from `app/`, Yarn 4 with `nodeLinker: node-modules` — Metro does not support PnP):

```sh
yarn test          # jest-expo + Testing Library v14 (render/fireEvent are async: always await)
yarn lint          # expo lint (ESLint flat config)
npx tsc --noEmit   # type check
npx expo start     # Expo Go on the phone; set EXPO_PUBLIC_API_URL in app/.env (see .env.example)
```

All of the above (backend and app) must pass before any commit.

## Structure

```
backend/
├── cmd/
│   ├── api/        # wiring + local HTTP server (Phase 1)
│   └── poller/     # Phase 2
└── internal/
    ├── status/     # domain: canonical states, normalization, change detection — pure, no I/O
    ├── source/     # StatusSource interface, Fallback, clients (proximotrem, cptm); testdata/ has real responses
    ├── catalog/    # static line metadata: name, color, operator id — the only place to update concessions
    ├── httpapi/    # GET /status: merges catalog + readings, 30s cache, serves stale data if sources fail
    ├── store/      # Phase 2 (DynamoDB)
    └── notify/     # Phase 2 (FCM)
app/                # Expo SDK 57, React Native 0.86, expo-router
└── src/
    ├── app/        # routes: _layout (fonts), index (status), sobre
    ├── api/        # GET /status client + contract types
    ├── domain/     # pure logic: grouping, late-night detection, screen state reducer
    ├── favorites/  # AsyncStorage persistence
    ├── theme/      # tokens from the Claude Design reference, contrast helpers
    ├── components/ # LineRow, Banner, Skeleton, StatusDot
    └── screens/    # StatusScreen (takes `load` as a prop for tests)
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

## App conventions

- Visual reference: Claude Design project "Status Trens SP v2" (via the `claude_design` MCP). Tokens live in `src/theme/tokens.ts`; keep them in sync with it.
- Status is never shown by color alone: every state has color + icon + label (`stateVisual`).
- The row shows the source `label` (e.g. "Atividade Programada") with the canonical state's color and icon.
- Logic that can be pure goes in `src/domain` and is unit tested; screens get dependencies as props.
- Operator logos: `src/operators.ts` maps operator id → SVG in `assets/images/operators/` (rendered by expo-image on a transparent background; in dark mode `logoTint` turns them fully white, except operators listed in `keepColorsInDark`). An operator without a logo shows its name; a line without operator shows no badge. All behind `SHOW_OPERATOR_LOGO` (trademark risk).
- Firebase: `@react-native-firebase/*` v26 (modular API only). Expo Go keeps working until app code imports Firebase (`npx expo start --go`); after that, use the EAS dev build (`eas.json` profile `development`). `app/google-services.json` is committed on purpose: it only identifies the Firebase project and ships inside every APK. The secret is the Firebase service account used by the backend to send pushes — that one never goes to git.

## Domain essentials

Canonical states: `normal`, `reduzida`, `parcial`, `paralisada`, `encerrada`, `desconhecido`. Unmapped source labels map to `desconhecido` and are logged — that log is how new labels get discovered and added to the rules (e.g. "Operação Transitória"/"Assistida" → `normal`: normal operation on limited hours).

**Lines, operators and labels are data, not code.** Concessions change and new lines and labels appear over time; never encode that history in code or comments. The system must accept them without refactoring: a new line from a source shows up by itself, the catalog only fixes its presentation, a new operator is one entry in `catalog.Operators`.

Data sources, in priority order:
1. **Próximo Trem** — `https://apim-proximotrem-prd-brazilsouth-001.azure-api.net/api/v1/lines`, no auth, 15 lines, no timestamp in the response.
2. **CPTM** — `http://apps.cptm.sp.gov.br:8080/AppMobileService/api/LinhasMetropolitanas`, no auth, 11 lines. Uses content negotiation: always send `Accept: application/json` (browsers get XML).
3. **Direto dos Trens** — requires a token (not available yet).

The app never calls a source directly; it only talks to our `GET /status`.

Sources report status plus the line name (cleaned: "LARANJA" → "Laranja", "Linha 4-Amarela" → "Amarela") and color when they have one (CPTM sends none). `GET /status` returns the union of catalog lines and source lines, sorted by code: catalog metadata wins, the source fills the gaps, and a line nobody colors gets a neutral gray. A catalog line missing from the source is `desconhecido`.

## GET /status contract

```json
{
  "updatedAt": "2026-10-08T01:30:38Z",
  "source": "proximotrem",
  "stale": false,
  "lines": [
    { "code": 9, "name": "Esmeralda", "colorHex": "#00a88e", "textColorHex": "#ffffff",
      "operator": "viamobilidade", "operatorName": "ViaMobilidade",
      "status": "reduzida", "label": "Atividade Programada", "description": "..." }
  ]
}
```

`operator`/`operatorName`/`textColorHex` are empty for a line without catalog entry (the app then picks the number color by contrast). Line colors and number colors in the catalog follow the current network map and station signs, not contrast math. `stale: true` means every source failed and this is the last known data. `503` with `{"error": ...}` only when there is no data at all. `changedAt` arrives in Phase 2 with the store.
