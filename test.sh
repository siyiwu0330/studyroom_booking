#!/usr/bin/env bash
set -euo pipefail

# ===== Config (override via flags or env) =====
BASE_URL="${BASE_URL:-http://localhost:8080}"
EMAIL="${EMAIL:-alice@example.com}"
PASSWORD="${PASSWORD:-supersecret}"
START="${START:-2025-10-10T10:00:00Z}"
END="${END:-2025-10-10T12:00:00Z}"
MIN_CAP="${MIN_CAP:-2}"

# 6 test profiles: threads, conns, duration
T1_T="${T1_T:-2}";  T1_C="${T1_C:-20}";   T1_D="${T1_D:-30s}"
T2_T="${T2_T:-2}";  T2_C="${T2_C:-50}";   T2_D="${T2_D:-30s}"
T3_T="${T3_T:-4}";  T3_C="${T3_C:-100}";  T3_D="${T3_D:-30s}"
T4_T="${T4_T:-4}";  T4_C="${T4_C:-150}";  T4_D="${T4_D:-30s}"
T5_T="${T5_T:-8}";  T5_C="${T5_C:-200}";  T5_D="${T5_D:-30s}"
T6_T="${T6_T:-8}";  T6_C="${T6_C:-300}";  T6_D="${T6_D:-30s}"

usage() {
  cat <<EOF
Usage: $0 [--base URL] [--email E] [--password P] [--start RFC3339] [--end RFC3339] [--mincap N]
Env overrides: BASE_URL EMAIL PASSWORD START END MIN_CAP
Profiles: T{1..6}_T, T{1..6}_C, T{1..6}_D  (threads, conns, duration)
EOF
  exit 1
}

# ----- Parse flags -----
while [[ $# -gt 0 ]]; do
  case "$1" in
    --base) BASE_URL="$2"; shift 2;;
    --email) EMAIL="$2"; shift 2;;
    --password) PASSWORD="$2"; shift 2;;
    --start) START="$2"; shift 2;;
    --end) END="$2"; shift 2;;
    --mincap) MIN_CAP="$2"; shift 2;;
    *) usage;;
  esac
done

# ----- Pick wrk (local or docker) -----
if command -v wrk >/dev/null 2>&1; then
  WRK="wrk"; WRK_MODE="local"
else
  WRK="docker run --rm --net=host williamyeh/wrk"; WRK_MODE="docker"
fi

# ----- Login (capture cookie) -----
TMP_HEADERS="$(mktemp)"; trap 'rm -f "$TMP_HEADERS"' EXIT
curl -i -s -X POST "$BASE_URL/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" > "$TMP_HEADERS"

COOKIE="$(grep -i '^Set-Cookie:' "$TMP_HEADERS" | head -n1 | sed -E 's/^Set-Cookie: ([^;]+);.*$/\1/i')"
[[ -z "$COOKIE" ]] && { echo "Login failed—no Set-Cookie returned"; exit 1; }

AUTH="Cookie: $COOKIE"
URL="$BASE_URL/search?start=$(printf %s "$START")&end=$(printf %s "$END")&min_capacity=$MIN_CAP"

echo "== Monolith (cookie) READ benchmark with wrk [$WRK_MODE]"
echo "URL: $URL"
echo "Header: $AUTH"
echo

run_case () {
  local name="$1" t="$2" c="$3" d="$4"
  echo "-- $name: ${t} threads, ${c} conns, ${d}"
  $WRK -t"$t" -c"$c" -d"$d" -H "$AUTH" "$URL"
  echo
}

run_case "T1 Light"    "$T1_T" "$T1_C" "$T1_D"
run_case "T2 Light+"   "$T2_T" "$T2_C" "$T2_D"
run_case "T3 Medium"   "$T3_T" "$T3_C" "$T3_D"
run_case "T4 Medium+"  "$T4_T" "$T4_C" "$T4_D"
run_case "T5 Heavy"    "$T5_T" "$T5_C" "$T5_D"
run_case "T6 Heavy+"   "$T6_T" "$T6_C" "$T6_D"
