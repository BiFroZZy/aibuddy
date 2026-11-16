# AI-buddy frontend

React/TypeScript interface for the Go API. The visual source is `docs/adaptation-plan/index.html`; all business state comes from the backend.

## Development

Start the backend with PostgreSQL and mock-hr using the repository's Docker Compose setup, then:

```sh
cd frontend
npm ci
npm run dev
```

Open http://localhost:3000. Choose one of five demo profiles. The profile is validated through `/api/me`; roles and team come from the server. Requests are prepared in the chat and submitted only after confirmation.

Vite proxies `/api` to `http://localhost:8080`. Set `API_PROXY_TARGET` to change that address. For separate API hosting, set `VITE_API_BASE_URL` at build time and configure the backend's `CORS_ALLOWED_ORIGIN`. A production same-origin host must proxy `/api` to the Go server; Vite's development proxy is not included in the build. Navigation uses hash URLs and needs no route rewrites.

Tokens live in session storage. Only the index of conversation IDs/titles is stored locally, separately for each server user; messages, progress, actions and requests are loaded from the API. Legacy `buddy-demo-*` data is not used.

## Verification

```sh
npm run lint
npm run build
npm test
npx playwright install chromium
npm run test:e2e:local
```

The last command requires Go, PostgreSQL binaries (`initdb`, `pg_ctl`), curl and a non-root user. It creates an isolated UTF-8 PostgreSQL cluster under `/tmp`, builds and starts the API and mock-hr, runs the browser tests, then stops all services. It never resets or writes to an existing project database. Logs remain under the printed temporary directory. Ports default to 55433, 28080 and 28081; override `E2E_PG_PORT`, `E2E_API_PORT`, `E2E_HR_PORT` if necessary.

Alternatively, against a **fresh dedicated** backend/database with seeded users:

```sh
API_PROXY_TARGET=http://localhost:8080 npm run test:e2e
```

E2E scenarios change demo progress and requests. Screenshots at widths 360, 390, 430, 820, 1024, 1366, 1440 and 1920 are saved under `test-results/screens`. Failure traces are also retained under `test-results`.

## API limitations reflected in the interface

- No request-creation form or saved drafts: preparation is performed through chat.
- Mentor comments and history of reviewed steps are unavailable; only the pending queue is shown.
- Managers see pending requests; there is no server endpoint for their decision history.
- Metrics show active users, event counts and tool usage, with a 7/30/90-day window, without a team filter.
- Request and action filters operate on the latest 50 records. Conversation history returns up to 100 messages.
- Materials link to the chat and public Playbook; there is no document-catalog endpoint.

To include backend integration tests with the race detector in the same isolated UTF-8 cluster:

```sh
E2E_BACKEND_TESTS=1 npm run test:e2e:local
```
