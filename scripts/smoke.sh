#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://localhost:8080}"
INTERN="Authorization: Bearer dev-intern-1"
MANAGER="Authorization: Bearer dev-manager"
J="Content-Type: application/json"
DATE="${ABSENCE_DATE:-2030-01-10}"

jq_() { python3 -c "import sys,json; d=json.load(sys.stdin); print($1)"; }
step() { printf '\n\033[1m== %s\033[0m\n' "$*"; }
fail() { printf '\033[31mFAIL: %s\033[0m\n' "$*" >&2; exit 1; }

step "readiness"
curl -fsS "$BASE/readyz" >/dev/null || fail "service is not ready"

step "who am I"
curl -fsS -H "$INTERN" "$BASE/api/me"; echo

step "1. Question about the process (answer from the knowledge base)"
curl -fsS -H "$INTERN" -H "$J" -d '{"message":"Как оформить отгул?"}' "$BASE/api/chat" | jq_ "d['reply']"

step "2. Ask to file the day off: a confirmation card comes back, nothing is sent yet"
RESP=$(curl -fsS -H "$INTERN" -H "$J" -d "{\"message\":\"Оформи отгул на $DATE\"}" "$BASE/api/chat")
echo "$RESP" | jq_ "d['reply']"
ACTION=$(echo "$RESP" | jq_ "d['pending_actions'][0]['id']") || fail "no pending action"

step "3. Confirm: the simulated absence service returns a result"
curl -fsS -X POST -H "$INTERN" "$BASE/api/actions/$ACTION/confirm" | jq_ "d['status'], d['result']"

step "4. Manager approves"
REQ=$(curl -fsS -H "$MANAGER" "$BASE/api/manager/requests" | jq_ "d['requests'][0]['id']")
curl -fsS -X POST -H "$MANAGER" -H "$J" -d '{"decision":"approve","comment":"ок"}' "$BASE/api/manager/requests/$REQ/decision" | jq_ "d['status']"

step "5. Status in chat"
curl -fsS -H "$INTERN" -H "$J" -d '{"message":"Какой статус моей заявки на отгул?"}' "$BASE/api/chat" | jq_ "d['reply']"

step "6. Checklist progress"
curl -fsS -H "$INTERN" "$BASE/api/progress" | jq_ "'%d%% (%d/%d)' % (d['percent'], d['done'], d['total'])"

step "7. Rights: an intern cannot read metrics"
CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "$INTERN" "$BASE/api/metrics")
[ "$CODE" = "403" ] || fail "expected 403, got $CODE"
echo "403 as expected"

printf '\n\033[32mSmoke test passed\033[0m\n'
