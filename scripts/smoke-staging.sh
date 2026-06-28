#!/usr/bin/env bash
# Smoke test for a deployed staging environment: liveness, auth, core reads,
# and the import safety limits (MAX_IMPORT_ROWS, MAX_REQUEST_BODY_BYTES)
# that prevent a repeat of the 368k-row import crash.
#
# Covers: GET /health, POST /api/auth/login, GET /api/me, GET /api/leads,
# GET /api/imports, a small successful import, and an oversized import
# rejection.
#
# Required env vars:
#   STAGING_URL       e.g. https://estatelink-staging.onrender.com
#   STAGING_EMAIL      admin or analyst account email
#   STAGING_PASSWORD   password for that account
#
# Optional env vars:
#   OVERSIZED_ROW_COUNT   rows to send to trigger the row-count rejection
#                         (default 1000000; must exceed MAX_IMPORT_ROWS on
#                         the target environment)
#
# Usage:
#   STAGING_URL=https://... STAGING_EMAIL=... STAGING_PASSWORD=... \
#     ./scripts/smoke-staging.sh

set -euo pipefail

: "${STAGING_URL:?STAGING_URL is required}"
: "${STAGING_EMAIL:?STAGING_EMAIL is required}"
: "${STAGING_PASSWORD:?STAGING_PASSWORD is required}"

OVERSIZED_ROW_COUNT="${OVERSIZED_ROW_COUNT:-1000000}"

pass() { echo "PASS: $1"; }
fail() { echo "FAIL: $1" >&2; exit 1; }

echo "== Health check =="
health_code=$(curl -s -o /dev/null -w '%{http_code}' "$STAGING_URL/health")
[ "$health_code" = "200" ] || fail "health check returned $health_code"
pass "health check returned 200"

echo "== Login =="
login_body=$(curl -s -X POST "$STAGING_URL/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$STAGING_EMAIL\",\"password\":\"$STAGING_PASSWORD\"}")

access_token=$(printf '%s' "$login_body" | grep -o '"accessToken":"[^"]*"' | head -1 | cut -d'"' -f4)
[ -n "$access_token" ] || fail "login did not return an access token: $login_body"
pass "logged in and obtained access token"

auth_header="Authorization: Bearer $access_token"

echo "== Current user (/api/me) =="
me_code=$(curl -s -o /dev/null -w '%{http_code}' -H "$auth_header" "$STAGING_URL/api/me")
[ "$me_code" = "200" ] || fail "/api/me expected 200, got $me_code"
pass "/api/me returned 200"

echo "== Leads list (/api/leads) =="
leads_code=$(curl -s -o /dev/null -w '%{http_code}' -H "$auth_header" "$STAGING_URL/api/leads")
[ "$leads_code" = "200" ] || fail "/api/leads expected 200, got $leads_code"
pass "/api/leads returned 200"

echo "== Small import succeeds =="
small_payload='[{"source":"smoke-test","property_id":"smoke-1","title":"Smoke test listing","price_val":100000}]'

small_code=$(curl -s -o /tmp/smoke_small_response.json -w '%{http_code}' \
  -X POST "$STAGING_URL/api/imports/clean-listings" \
  -H "$auth_header" \
  -H 'Content-Type: application/json' \
  -d "$small_payload")

[ "$small_code" = "202" ] || fail "small import expected 202, got $small_code: $(cat /tmp/smoke_small_response.json)"
pass "small import accepted (202)"

job_id=$(grep -o '"jobId":"[^"]*"' /tmp/smoke_small_response.json | cut -d'"' -f4)
if [ -n "$job_id" ]; then
  echo "== Import job status =="
  job_code=$(curl -s -o /tmp/smoke_job_response.json -w '%{http_code}' \
    -H "$auth_header" \
    "$STAGING_URL/api/imports/$job_id")
  [ "$job_code" = "200" ] || fail "fetching import job expected 200, got $job_code"
  pass "import job $job_id fetched (200)"
fi

echo "== List import jobs =="
list_code=$(curl -s -o /dev/null -w '%{http_code}' \
  -H "$auth_header" \
  "$STAGING_URL/api/imports?limit=5")
[ "$list_code" = "200" ] || fail "listing import jobs expected 200, got $list_code"
pass "import job list returned 200"

echo "== Oversized import is rejected (row count) =="
oversized_payload=$(python3 - "$OVERSIZED_ROW_COUNT" <<'PY'
import sys, json
n = int(sys.argv[1])
rows = [{"source": "smoke-test", "property_id": f"smoke-{i}"} for i in range(n)]
print(json.dumps(rows))
PY
)

oversized_code=$(curl -s -o /tmp/smoke_oversized_response.json -w '%{http_code}' \
  -X POST "$STAGING_URL/api/imports/clean-listings" \
  -H "$auth_header" \
  -H 'Content-Type: application/json' \
  -d "$oversized_payload")

if [ "$oversized_code" = "400" ] || [ "$oversized_code" = "413" ]; then
  pass "oversized import rejected ($oversized_code)"
else
  fail "oversized import expected 400 or 413, got $oversized_code: $(cat /tmp/smoke_oversized_response.json)"
fi

echo
echo "All smoke checks passed against $STAGING_URL"
