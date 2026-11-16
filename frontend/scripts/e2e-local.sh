#!/usr/bin/env bash
# Runs on an isolated temporary PostgreSQL cluster; never touches project databases.
set -euo pipefail
frontend_dir="$(cd "$(dirname "$0")/.." && pwd)"
repo_dir="$(dirname "$frontend_dir")"
run_dir="$(mktemp -d /tmp/aibuddy-ui-e2e.XXXXXX)"
pg_port="${E2E_PG_PORT:-55433}"
api_port="${E2E_API_PORT:-28080}"
hr_port="${E2E_HR_PORT:-28081}"
api_pid=''
hr_pid=''
cleanup() {
  if [[ -n "$api_pid" ]]; then kill "$api_pid" 2>/dev/null || true; wait "$api_pid" 2>/dev/null || true; fi
  if [[ -n "$hr_pid" ]]; then kill "$hr_pid" 2>/dev/null || true; wait "$hr_pid" 2>/dev/null || true; fi
  pg_ctl -D "$run_dir/pg" -m fast stop >/dev/null 2>&1 || true
  echo "E2E service logs: $run_dir"
}
trap cleanup EXIT
initdb -D "$run_dir/pg" --encoding=UTF8 --no-locale -A trust -U app >"$run_dir/init.log"
pg_ctl -D "$run_dir/pg" -l "$run_dir/postgres.log" -o "-h 127.0.0.1 -p $pg_port -k $run_dir" start
export DATABASE_URL="postgres://app@127.0.0.1:$pg_port/postgres?sslmode=disable"
(cd "$repo_dir/backend" && go build -o "$run_dir/server" ./cmd/app && go build -o "$run_dir/mockhr" ./cmd/mockhr)
if [[ "${E2E_BACKEND_TESTS:-0}" == 1 ]]; then
  (cd "$repo_dir/backend" && TEST_DATABASE_URL="$DATABASE_URL" go test -race -p 1 -count=1 ./...)
fi
MOCKHR_ADDR="127.0.0.1:$hr_port" "$run_dir/mockhr" >"$run_dir/hr.log" 2>&1 &
hr_pid=$!
for attempt in $(seq 1 100); do
  if curl -fsS "http://127.0.0.1:$hr_port/healthz" >/dev/null 2>&1; then break; fi
  sleep .1
done
HTTP_ADDR="127.0.0.1:$api_port" HR_URL="http://127.0.0.1:$hr_port" LLM_MODE=fake "$run_dir/server" >"$run_dir/api.log" 2>&1 &
api_pid=$!
for attempt in $(seq 1 100); do
  if curl -fsS "http://127.0.0.1:$api_port/readyz" >/dev/null 2>&1; then break; fi
  sleep .1
done
curl -fsS "http://127.0.0.1:$api_port/readyz" >/dev/null
cd "$frontend_dir"
API_PROXY_TARGET="http://127.0.0.1:$api_port" npm run test:e2e -- "$@"
