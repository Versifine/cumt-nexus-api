#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESULT_DIR="$REPO/docs/reports/loadtest"
TMP_DIR="$REPO/.tmp"

USERS=1000
COMMUNITIES=50
POSTS=20000
COMMENTS=80000
POST_VOTES=120000
POST_SAVES=30000
NOTIFICATIONS=12000
REPORTS=3000
VUS=50
DURATION_SECONDS=60
WARMUP_SECONDS=5
PORT=18080
DATABASE="cumt_nexus_loadtest"
INCLUDE=""
EXCLUDE=""
TARGET_RPS=0
SKIP_SEED=false
KEEP_API=false
API_PID=""

usage() {
  cat <<'USAGE'
Usage: scripts/loadtest/run-local-loadtest.sh [options]

Options:
  --users N, -Users N
  --communities N, -Communities N
  --posts N, -Posts N
  --comments N, -Comments N
  --post-votes N, -PostVotes N
  --post-saves N, -PostSaves N
  --notifications N, -Notifications N
  --reports N, -Reports N
  --vus N, -VUs N
  --duration-seconds N, -DurationSeconds N
  --warmup-seconds N, -WarmupSeconds N
  --port N, -Port N
  --database NAME, -Database NAME
  --include NAME[,NAME], -Include NAME[,NAME]
  --exclude NAME[,NAME], -Exclude NAME[,NAME]
  --target-rps N, -TargetRPS N
  --skip-seed, -SkipSeed
  --keep-api, -KeepApi
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --users|-Users) USERS="${2:-}"; shift 2 ;;
    --communities|-Communities) COMMUNITIES="${2:-}"; shift 2 ;;
    --posts|-Posts) POSTS="${2:-}"; shift 2 ;;
    --comments|-Comments) COMMENTS="${2:-}"; shift 2 ;;
    --post-votes|-PostVotes) POST_VOTES="${2:-}"; shift 2 ;;
    --post-saves|-PostSaves) POST_SAVES="${2:-}"; shift 2 ;;
    --notifications|-Notifications) NOTIFICATIONS="${2:-}"; shift 2 ;;
    --reports|-Reports) REPORTS="${2:-}"; shift 2 ;;
    --vus|-VUs) VUS="${2:-}"; shift 2 ;;
    --duration-seconds|-DurationSeconds) DURATION_SECONDS="${2:-}"; shift 2 ;;
    --warmup-seconds|-WarmupSeconds) WARMUP_SECONDS="${2:-}"; shift 2 ;;
    --port|-Port) PORT="${2:-}"; shift 2 ;;
    --database|-Database) DATABASE="${2:-}"; shift 2 ;;
    --include|-Include) INCLUDE="${2:-}"; shift 2 ;;
    --exclude|-Exclude) EXCLUDE="${2:-}"; shift 2 ;;
    --target-rps|-TargetRPS) TARGET_RPS="${2:-}"; shift 2 ;;
    --skip-seed|-SkipSeed) SKIP_SEED=true; shift ;;
    --keep-api|-KeepApi) KEEP_API=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

mkdir -p "$RESULT_DIR" "$TMP_DIR"

set_loadtest_env() {
  export APP_NAME="cumt-nexus-api"
  export APP_ENV="test"
  export APP_STARTUP_TIMEOUT="20s"
  export POSTGRES_HOST="localhost"
  export POSTGRES_PORT="5432"
  export POSTGRES_USER="postgres"
  export POSTGRES_PASSWORD="postgres"
  export POSTGRES_DATABASE="$DATABASE"
  export POSTGRES_SSL_MODE="disable"
  export POSTGRES_MAX_CONNS="50"
  export POSTGRES_MAX_CONN_LIFETIME="5m"
  export POSTGRES_MAX_CONN_IDLE_TIME="2m"
  export HTTP_ADDR=":$PORT"
  export HTTP_READ_TIMEOUT="5s"
  export HTTP_WRITE_TIMEOUT="10s"
  export HTTP_SHUTDOWN_TIMEOUT="15s"
  export HTTP_CORS_ALLOWED_ORIGINS="http://localhost:3000,http://127.0.0.1:3000"
  export LOG_LEVEL="warn"
  export LOG_FORMAT="json"
  export GIN_MODE="release"
  export AUTH_TOKEN_SECRET="loadtest-auth-secret"
  export AUTH_ACCESS_TOKEN_TTL="24h"
  export AUTH_EMAIL_ALLOWED_DOMAINS="cumt.edu.cn,mail.cumt.edu.cn"
  export AUTH_EMAIL_CODE_TTL="10m"
  export AUTH_EMAIL_CODE_RESEND_INTERVAL="1m"
  export AUTH_EMAIL_CODE_MAX_ATTEMPTS="5"
  export AUTH_EMAIL_CODE_DAILY_LIMIT="10"
  export AUTH_EMAIL_CODE_IP_HOURLY_LIMIT="30"
  export AUTH_EMAIL_CODE_LENGTH="6"
  export MAIL_PROVIDER="log"
  export SMTP_HOST=""
  export SMTP_PORT="587"
  export SMTP_USERNAME=""
  export SMTP_PASSWORD=""
  export SMTP_FROM=""
  export SMTP_TLS_MODE="starttls"
  export OBJECT_STORAGE_PROVIDER="local"
  export OBJECT_STORAGE_ENDPOINT=""
  export OBJECT_STORAGE_REGION="auto"
  export OBJECT_STORAGE_BUCKET=""
  export OBJECT_STORAGE_ACCESS_KEY_ID=""
  export OBJECT_STORAGE_SECRET_ACCESS_KEY=""
  export OBJECT_STORAGE_PUBLIC_BASE_URL="http://localhost:$PORT/uploads"
  export OBJECT_STORAGE_FORCE_PATH_STYLE="true"
  export OBJECT_STORAGE_LOCAL_ROOT="var/uploads-loadtest"
  export UPLOAD_IMAGE_MAX_BYTES="5242880"
  export UPLOAD_IMAGE_MAX_COUNT_PER_POST="9"
  export UPLOAD_IMAGE_MAX_COUNT_PER_COMMENT="1"
}

compose() {
  if [[ -n "${COMPOSE_FILE:-}" ]]; then
    docker compose "$@"
  elif [[ -f "$REPO/compose.yaml" || -f "$REPO/compose.yml" || -f "$REPO/docker-compose.yml" || -f "$REPO/docker-compose.yaml" ]]; then
    docker compose "$@"
  else
    docker compose -f "$REPO/docker-compose.prod.yml" "$@"
  fi
}

wait_http_ok() {
  local url="$1"
  local timeout="$2"
  local deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    if curl -fsS --max-time 2 "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "timed out waiting for $url" >&2
  return 1
}

wait_postgres_ready() {
  local timeout="$1"
  local deadline=$((SECONDS + timeout))
  while (( SECONDS < deadline )); do
    if compose exec -T postgres pg_isready -U postgres -d postgres >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  echo "timed out waiting for PostgreSQL readiness" >&2
  return 1
}

stop_process_on_port() {
  local port="$1"
  local pids=""
  if command -v lsof >/dev/null 2>&1; then
    pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  elif command -v fuser >/dev/null 2>&1; then
    pids="$(fuser -n tcp "$port" 2>/dev/null || true)"
  fi
  for pid in $pids; do
    local name
    name="$(ps -p "$pid" -o comm= 2>/dev/null | awk '{print $1}')"
    if [[ "$name" == "api" || "$name" == "loadtest-api" ]]; then
      kill "$pid" >/dev/null 2>&1 || true
    elif [[ -n "$name" ]]; then
      echo "port $port is already used by process $pid ($name); refusing to stop a non-api process" >&2
      return 1
    fi
  done
}

cleanup() {
  if [[ "$KEEP_API" != true && -n "$API_PID" ]]; then
    kill "$API_PID" >/dev/null 2>&1 || true
    wait "$API_PID" >/dev/null 2>&1 || true
  fi
  if [[ "$KEEP_API" != true ]]; then
    stop_process_on_port "$PORT" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

cd "$REPO"
set_loadtest_env

compose up -d postgres
wait_postgres_ready 60
if ! compose exec -T postgres psql -U postgres -d postgres -tc "SELECT 1 FROM pg_database WHERE datname = '$DATABASE'" | grep -q "1"; then
  compose exec -T postgres createdb -U postgres "$DATABASE"
fi

go run ./cmd/migrate up

if [[ "$SKIP_SEED" != true ]]; then
  go run ./scripts/loadtest/cmd/seed \
    -users "$USERS" \
    -communities "$COMMUNITIES" \
    -posts "$POSTS" \
    -comments "$COMMENTS" \
    -post-votes "$POST_VOTES" \
    -post-saves "$POST_SAVES" \
    -notifications "$NOTIFICATIONS" \
    -reports "$REPORTS" \
    -reset | tee "$RESULT_DIR/seed-summary.json"
fi

stop_process_on_port "$PORT"

API_STDOUT_LOG="$TMP_DIR/loadtest-api.stdout.log"
API_STDERR_LOG="$TMP_DIR/loadtest-api.stderr.log"
API_BINARY="$TMP_DIR/loadtest-api"
rm -f "$API_STDOUT_LOG" "$API_STDERR_LOG" "$API_BINARY"
go build -buildvcs=false -o "$API_BINARY" ./cmd/api
"$API_BINARY" >"$API_STDOUT_LOG" 2>"$API_STDERR_LOG" &
API_PID="$!"
wait_http_ok "http://127.0.0.1:$PORT/healthz" 60

STAMP="$(date +%Y%m%d-%H%M%S)"
JSON_PATH="$RESULT_DIR/loadtest-$STAMP.json"
MD_PATH="$RESULT_DIR/loadtest-$STAMP.md"
RUNNER_ARGS=(
  run ./scripts/loadtest/cmd/runner
  -base-url "http://127.0.0.1:$PORT"
  -vus "$VUS"
  -duration "${DURATION_SECONDS}s"
  -warmup "${WARMUP_SECONDS}s"
  -users "$USERS"
  -communities "$COMMUNITIES"
  -posts "$POSTS"
  -comments "$COMMENTS"
  -notifications "$NOTIFICATIONS"
  -reports "$REPORTS"
)
if [[ -n "$INCLUDE" ]]; then
  RUNNER_ARGS+=(-include "$INCLUDE")
fi
if [[ -n "$EXCLUDE" ]]; then
  RUNNER_ARGS+=(-exclude "$EXCLUDE")
fi
if awk "BEGIN {exit !($TARGET_RPS > 0)}"; then
  RUNNER_ARGS+=(-target-rps "$TARGET_RPS")
fi
RUNNER_ARGS+=(-out-json "$JSON_PATH" -out-md "$MD_PATH")

go "${RUNNER_ARGS[@]}"

echo "JSON report: $JSON_PATH"
echo "Markdown report: $MD_PATH"
