# Status Trens SP

Android app that shows the live status of São Paulo metro and train lines and sends a push notification when a favorited line changes status. No login, no user accounts: favorites live on the device and the backend never knows who the user is.

Experimental portfolio project. Full product, architecture and decision docs live in Notion (page "🚆 Status Trens SP", subpages Arquitetura, Fontes de Dados, Decisões, Roadmap). **Check Decisões and Roadmap before proposing changes** — most trade-offs are already settled there.

## Current phase

**Phase 3 — AWS.** Phase 1 (display) and Phase 2 (notification) are done and run locally: `GET /status` serves the app; the app subscribes each favorite to its FCM topic and handles the notification permission; `cmd/poller` detects changes and pushes them through FCM, validated with a real cycle (`-seed`) on Samsung and Xiaomi devices; the app ships per-line notification icons and the "status" channel. Next: deploy to AWS (Lambda, EventBridge, DynamoDB store, key in SSM).

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
FIREBASE_SERVICE_ACCOUNT_FILE=~/.config/status-trens-sp/firebase-sa.json go run ./cmd/poller                # polls every POLL_INTERVAL (1m), pushes changes
FIREBASE_SERVICE_ACCOUNT_FILE=~/.config/status-trens-sp/firebase-sa.json go run ./cmd/poller -send-test 9   # one test push to linha-9 (add NOTIFICATION_IMAGE_BASE_URL for the badge)
FIREBASE_SERVICE_ACCOUNT_FILE=~/.config/status-trens-sp/firebase-sa.json go run ./cmd/poller -seed 3=paralisada   # dev only: line 3 starts as paralisada, so the first real reading is pushed (full cycle, no waiting)
go run ./cmd/badges -out badges   # line badges for the notification picture (backend/badges/); commit them after a catalog change

docker build -t status-trens-sp-api .                  # distroless image (~17 MB), non-root
docker run -d --name status-trens-sp-api -p 8080:8080 status-trens-sp-api
```

App (run from `app/`, Yarn 4 with `nodeLinker: node-modules` — Metro does not support PnP):

```sh
yarn test          # jest-expo + Testing Library v14 (render/fireEvent are async: always await)
yarn lint          # expo lint (ESLint flat config)
npx tsc --noEmit   # type check
npx expo start --dev-client   # needs the EAS dev build on the phone (Expo Go has no Firebase); set EXPO_PUBLIC_API_URL in app/.env (see .env.example)
```

All of the above (backend and app) must pass before any commit.

## Structure

```
backend/
├── cmd/
│   ├── api/        # wiring + local HTTP server (Phase 1)
│   └── poller/     # local loop around poll.Poller.Run (Phase 3: Lambda handler)
└── internal/
    ├── status/     # domain: canonical states, normalization, change detection — pure, no I/O
    ├── source/     # StatusSource interface, Fallback, clients (proximotrem, cptm); testdata/ has real responses
    ├── catalog/    # line metadata as data (name, colors, operator id) and Present(): catalog first, source fills gaps
    ├── httpapi/    # GET /status: merges catalog + readings, 30s cache, serves stale data if sources fail
    ├── poll/       # one cycle: fetch → status.Next per line → push → save; Store and Notifier interfaces
    ├── store/      # line memory: InMemory (local) and DynamoDB (one item per line: code, state, lastNotified)
    └── notify/     # FCM HTTP v1 to topic linha-<code>: icon ic_line_<code>, line color, channel "status", ttl 1h
app/                # Expo SDK 57, React Native 0.86, expo-router
├── assets/notification/  # generated ic_line_N small icons (scripts/notification-icons.sh, ImageMagick 7)
├── plugins/        # with-notifications: copies the icons into res/ and creates the "status" channel at prebuild
└── src/
    ├── app/        # routes: _layout (fonts), index (status), sobre
    ├── api/        # GET /status client + contract types
    ├── domain/     # pure logic: grouping, late-night detection, screen state reducer
    ├── favorites/  # AsyncStorage persistence
    ├── push/       # FCM topic per line (`linha-<code>`), notification permission; `Push` is injected into the screen
    ├── theme/      # tokens from the Claude Design reference, contrast helpers
    ├── components/ # LineRow, Banner, Skeleton, StatusDot, PermissionSheet
    └── screens/    # StatusScreen (takes `load` and `push` as props for tests)
```

Create packages only when the first test needs them.

## Backend conventions

- `internal/status` is pure domain: no HTTP, no AWS, no clock reads — pass time in as a value.
- External dependencies enter through small interfaces defined by the consumer (`StatusSource`, later `Store`, `Notifier`).
- Config via environment variables; secrets never in code or tests. The Firebase service account key lives outside the repo (locally `~/.config/status-trens-sp/`, Phase 3 in SSM) and is read via `FIREBASE_SERVICE_ACCOUNT_FILE`.
- A failed push leaves the line's memory untouched, so the next cycle detects and pushes the same change again.
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
- Operator logos: `src/operators.ts` maps operator id → SVG in `assets/images/operators/` (rendered by expo-image on a transparent background; in dark mode `logoFor` uses the operator's own dark variant when it has one — Metrô: the official "traço grosso" signature extracted from the Metrô visual identity guide — and tints every other logo fully white). An operator without a logo shows its name; a line without operator shows no badge. All behind `SHOW_OPERATOR_LOGO` (trademark risk).
- Push: favoriting subscribes to `linha-<code>`, unfavoriting unsubscribes, and every stored favorite is subscribed again on app start (idempotent; repairs failed subscriptions). A failed subscription never blocks favoriting. On Android 13+ the first favorite opens the explanation sheet, then Android's dialog; a refusal is remembered in AsyncStorage (Android only says granted or not) and shows the "Notificações desativadas" banner while there are favorites. Returning to the foreground loads the status again and checks the permission again (Android may keep the app alive for hours). Firebase is mocked in `jest.setup.ts`.
- Firebase: `@react-native-firebase/*` v26 (modular API only). App code imports Firebase, so Expo Go no longer runs the app: use the EAS dev build (`eas.json` profile `development`, `eas build --profile development --platform android`). JS-only changes need no new build. `app/google-services.json` is committed on purpose: it only identifies the Firebase project and ships inside every APK. The secret is the Firebase service account used by the backend to send pushes — that one never goes to git.
- Notifications are FCM notification messages displayed by Android, never by the app: small icon `ic_line_N` (white rounded square, number knocked out; Android keeps only alpha) tinted with the line color sent by the backend, single channel `status` (set as RNFB's default in `app/firebase.json`). The colored line badge comes as the FCM `image` (Android shows it as the picture on the right): `backend/cmd/badges` renders `linha-<code>.png` from the catalog (Open Sans Bold, ImageMagick 7) into `backend/badges/`, committed and served from this public repo: `NOTIFICATION_IMAGE_BASE_URL=https://raw.githubusercontent.com/DouglasDans/status-trens-sp/main/backend/badges`. Nothing is shown while the app is open, by decision. Icons and channel are native: changing them needs a new EAS build.

## Domain essentials

Canonical states: `normal`, `reduzida`, `parcial`, `paralisada`, `encerrada`, `desconhecido`. Unmapped source labels map to `desconhecido` and are logged — that log is how new labels get discovered and added to the rules (e.g. "Operação Transitória"/"Assistida" → `normal`: normal operation on limited hours).

Notification rules live in `status.Next` (pure, table tested): the first reading is a baseline; `desconhecido` carries no information; closing never notifies and an ordinary opening (`encerrada → normal`) only does when the last push was about a problem; every other change of canonical state notifies (label/description changes within a state do not). Phase 2 runs without AWS: the store is an interface with an in-memory version locally, DynamoDB arrives in Phase 3.

**The system only relays what sources and operators say.** No schedules or operating rules in code (opening times differ per line and change; only a source says a line is closed); descriptions are always the operator's text.

**Lines, operators and labels are data, not code.** Concessions change and new lines and labels appear over time; never encode that history in code or comments. The system must accept them without refactoring: a new line from a source shows up by itself, the catalog only fixes its presentation, a new operator is one entry in `catalog.Operators`.

Data sources, in priority order:
1. **Próximo Trem** — `https://apim-proximotrem-prd-brazilsouth-001.azure-api.net/api/v1/lines`, no auth, 15 lines, no timestamp in the response.
2. **CPTM** — `http://apps.cptm.sp.gov.br:8080/AppMobileService/api/LinhasMetropolitanas`, no auth, 11 lines. Uses content negotiation: always send `Accept: application/json` (browsers get XML).
3. **Direto dos Trens** — requires a token (not available yet).

The app never calls a source directly; it only talks to our `GET /status`.

Sources report status plus the line name (cleaned: "LARANJA" → "Laranja", "Linha 4-Amarela" → "Amarela") and color when they have one (CPTM sends none). `GET /status` returns exactly the lines the source reports, sorted by code: catalog metadata wins, the source fills the gaps, and a line nobody colors gets a neutral gray. The catalog may hold lines prepared before they open; they stay hidden until a source reports them. Known trade-off: when the fallback source lacks a line (CPTM has no 13, 15, 17), that line disappears until the primary source is back.

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
