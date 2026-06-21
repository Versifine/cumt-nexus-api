#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$SCRIPT_DIR/../.." && pwd)"
RESULT_DIR="$REPO/docs/reports/loadtest"
TMP_DIR="$REPO/.tmp"
RUN_LOCAL_SCRIPT="$SCRIPT_DIR/run-local-loadtest.sh"

ENDPOINT=""
RPS_STEPS="5,10,15,20,25,30"
STEP_DURATION_SECONDS=60
WARMUP_SECONDS=10
VUS=80
USERS=1000
COMMUNITIES=50
POSTS=20000
COMMENTS=80000
POST_VOTES=120000
POST_SAVES=30000
NOTIFICATIONS=12000
REPORTS=3000
PORT=18080
DATABASE="cumt_nexus_loadtest"
P95_THRESHOLD_MS=2000
ERROR_RATE_THRESHOLD=0.01
ACTUAL_RPS_RATIO_THRESHOLD=0.95
SKIP_SEED=false

usage() {
  cat <<'USAGE'
Usage: scripts/loadtest/run-fixed-rps-ladder.sh --endpoint ENDPOINT [options]

Options:
  --endpoint NAME, -Endpoint NAME
  --rps-steps LIST, -RpsSteps LIST
  --step-duration-seconds N, -StepDurationSeconds N
  --warmup-seconds N, -WarmupSeconds N
  --vus N, -VUs N
  --users N, -Users N
  --communities N, -Communities N
  --posts N, -Posts N
  --comments N, -Comments N
  --post-votes N, -PostVotes N
  --post-saves N, -PostSaves N
  --notifications N, -Notifications N
  --reports N, -Reports N
  --port N, -Port N
  --database NAME, -Database NAME
  --p95-threshold-ms N, -P95ThresholdMs N
  --error-rate-threshold N, -ErrorRateThreshold N
  --actual-rps-ratio-threshold N, -ActualRpsRatioThreshold N
  --skip-seed, -SkipSeed
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --endpoint|-Endpoint) ENDPOINT="${2:-}"; shift 2 ;;
    --rps-steps|-RpsSteps) RPS_STEPS="${2:-}"; shift 2 ;;
    --step-duration-seconds|-StepDurationSeconds) STEP_DURATION_SECONDS="${2:-}"; shift 2 ;;
    --warmup-seconds|-WarmupSeconds) WARMUP_SECONDS="${2:-}"; shift 2 ;;
    --vus|-VUs) VUS="${2:-}"; shift 2 ;;
    --users|-Users) USERS="${2:-}"; shift 2 ;;
    --communities|-Communities) COMMUNITIES="${2:-}"; shift 2 ;;
    --posts|-Posts) POSTS="${2:-}"; shift 2 ;;
    --comments|-Comments) COMMENTS="${2:-}"; shift 2 ;;
    --post-votes|-PostVotes) POST_VOTES="${2:-}"; shift 2 ;;
    --post-saves|-PostSaves) POST_SAVES="${2:-}"; shift 2 ;;
    --notifications|-Notifications) NOTIFICATIONS="${2:-}"; shift 2 ;;
    --reports|-Reports) REPORTS="${2:-}"; shift 2 ;;
    --port|-Port) PORT="${2:-}"; shift 2 ;;
    --database|-Database) DATABASE="${2:-}"; shift 2 ;;
    --p95-threshold-ms|-P95ThresholdMs) P95_THRESHOLD_MS="${2:-}"; shift 2 ;;
    --error-rate-threshold|-ErrorRateThreshold) ERROR_RATE_THRESHOLD="${2:-}"; shift 2 ;;
    --actual-rps-ratio-threshold|-ActualRpsRatioThreshold) ACTUAL_RPS_RATIO_THRESHOLD="${2:-}"; shift 2 ;;
    --skip-seed|-SkipSeed) SKIP_SEED=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *)
      if [[ -z "$ENDPOINT" ]]; then
        ENDPOINT="$1"
        shift
      else
        echo "unknown argument: $1" >&2
        usage >&2
        exit 2
      fi
      ;;
  esac
done

if [[ -z "$ENDPOINT" ]]; then
  echo "--endpoint is required" >&2
  usage >&2
  exit 2
fi

mkdir -p "$RESULT_DIR" "$TMP_DIR"

mapfile -t RPS_VALUES < <(python3 - "$RPS_STEPS" <<'PY'
import re
import sys
values = [item for item in re.split(r"[,;\s]+", sys.argv[1].strip()) if item]
if not values:
    raise SystemExit("RpsSteps must contain at least one numeric target RPS")
for value in values:
    float(value)
    print(value)
PY
)

SAFE_ENDPOINT="$(python3 - "$ENDPOINT" <<'PY'
import re
import sys
print(re.sub(r"[^a-zA-Z0-9_-]", "_", sys.argv[1]))
PY
)"
ROWS_JSONL="$TMP_DIR/fixed-rps-ladder-$SAFE_ENDPOINT.rows.jsonl"
rm -f "$ROWS_JSONL"
STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
SEED_ALREADY_HANDLED="$SKIP_SEED"

for rps in "${RPS_VALUES[@]}"; do
  echo "Running fixed RPS step: endpoint=$ENDPOINT target_rps=$rps"
  LOG_PREFIX="$TMP_DIR/fixed-rps-step-$SAFE_ENDPOINT-$rps"
  STDOUT_LOG="$LOG_PREFIX.stdout.log"
  STDERR_LOG="$LOG_PREFIX.stderr.log"
  ARGS=(
    --users "$USERS"
    --communities "$COMMUNITIES"
    --posts "$POSTS"
    --comments "$COMMENTS"
    --post-votes "$POST_VOTES"
    --post-saves "$POST_SAVES"
    --notifications "$NOTIFICATIONS"
    --reports "$REPORTS"
    --vus "$VUS"
    --duration-seconds "$STEP_DURATION_SECONDS"
    --warmup-seconds "$WARMUP_SECONDS"
    --port "$PORT"
    --database "$DATABASE"
    --include "$ENDPOINT"
    --target-rps "$rps"
  )
  if [[ "$SEED_ALREADY_HANDLED" == true ]]; then
    ARGS+=(--skip-seed)
  fi
  if ! "$RUN_LOCAL_SCRIPT" "${ARGS[@]}" >"$STDOUT_LOG" 2>"$STDERR_LOG"; then
    cat "$STDOUT_LOG" || true
    cat "$STDERR_LOG" >&2 || true
    echo "fixed RPS step failed for endpoint=$ENDPOINT target_rps=$rps" >&2
    exit 1
  fi
  SEED_ALREADY_HANDLED=true
  JSON_PATH="$(grep -E '^JSON report:' "$STDOUT_LOG" | tail -n 1 | sed -E 's/^JSON report:[[:space:]]*//')"
  if [[ -z "$JSON_PATH" ]]; then
    cat "$STDOUT_LOG" || true
    cat "$STDERR_LOG" >&2 || true
    echo "runner did not print JSON report path for endpoint=$ENDPOINT target_rps=$rps" >&2
    exit 1
  fi
  python3 - "$JSON_PATH" "$ENDPOINT" "$rps" "$P95_THRESHOLD_MS" "$ERROR_RATE_THRESHOLD" "$ACTUAL_RPS_RATIO_THRESHOLD" <<'PY' >>"$ROWS_JSONL"
import json
import sys
from pathlib import Path

json_path, endpoint, rps, p95_threshold, error_threshold, actual_ratio = sys.argv[1:]
report = json.loads(Path(json_path).read_text(encoding="utf-8"))
metrics = report.get("endpoints", {}).get(endpoint)
if not metrics:
    raise SystemExit(f"report {json_path} does not contain endpoint {endpoint}")
target = float(rps)
row = {
    "endpoint": endpoint,
    "target_rps": target,
    "actual_rps": float(metrics["rps"]),
    "requests": int(metrics["requests"]),
    "error_rate": float(metrics["error_rate"]),
    "p50_ms": float(metrics["p50_ms"]),
    "p95_ms": float(metrics["p95_ms"]),
    "p99_ms": float(metrics["p99_ms"]),
    "max_ms": float(metrics["max_ms"]),
    "json_report": json_path,
    "markdown_report": str(Path(json_path).with_suffix(".md")),
}
row["healthy"] = (
    row["error_rate"] <= float(error_threshold)
    and row["p95_ms"] <= float(p95_threshold)
    and row["actual_rps"] >= target * float(actual_ratio)
)
print(json.dumps(row, ensure_ascii=False))
PY
done

FINISHED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
STAMP="$(date +%Y%m%d-%H%M%S)"
SUMMARY_JSON_PATH="$RESULT_DIR/fixed-rps-ladder-$SAFE_ENDPOINT-$STAMP.json"
SUMMARY_MD_PATH="$RESULT_DIR/fixed-rps-ladder-$SAFE_ENDPOINT-$STAMP.md"

python3 - "$ROWS_JSONL" "$SUMMARY_JSON_PATH" "$SUMMARY_MD_PATH" "$ENDPOINT" "$STARTED_AT" "$FINISHED_AT" "$STEP_DURATION_SECONDS" "$WARMUP_SECONDS" "$VUS" "$P95_THRESHOLD_MS" "$ERROR_RATE_THRESHOLD" "$ACTUAL_RPS_RATIO_THRESHOLD" <<'PY'
import json
import sys
from pathlib import Path

(
    rows_jsonl,
    summary_json_path,
    summary_md_path,
    endpoint,
    started_at,
    finished_at,
    step_duration,
    warmup,
    vus,
    p95_threshold,
    error_threshold,
    actual_ratio,
) = sys.argv[1:]

rows = [json.loads(line) for line in Path(rows_jsonl).read_text(encoding="utf-8").splitlines() if line.strip()]
last_healthy = next((row for row in reversed(rows) if row["healthy"]), None)
first_bad = next((row for row in rows if not row["healthy"]), None)
summary = {
    "endpoint": endpoint,
    "started_at": started_at,
    "finished_at": finished_at,
    "step_duration_seconds": int(step_duration),
    "warmup_seconds": int(warmup),
    "vus": int(vus),
    "thresholds": {
        "p95_ms": float(p95_threshold),
        "error_rate": float(error_threshold),
        "actual_rps_ratio": float(actual_ratio),
    },
    "last_healthy": last_healthy,
    "first_bad": first_bad,
    "steps": rows,
}
Path(summary_json_path).write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

def percent(value: float) -> str:
    return f"{value * 100:.2f}%"

lines = [
    "# Fixed RPS Ladder Report",
    "",
    f"- Endpoint: `{endpoint}`",
    f"- Step duration: `{step_duration}s` measured, `{warmup}s` warmup",
    f"- VUs: `{vus}`",
    f"- Healthy rule: error rate <= `{percent(float(error_threshold))}`, p95 <= `{float(p95_threshold):g} ms`, actual RPS >= `{float(actual_ratio) * 100:.2f}%` of target",
]
if last_healthy:
    lines.append(f"- Last healthy step: target `{last_healthy['target_rps']:.2f} req/s` with p95 `{last_healthy['p95_ms']:.2f} ms`")
if first_bad:
    lines.append(f"- First failing step: target `{first_bad['target_rps']:.2f} req/s` with p95 `{first_bad['p95_ms']:.2f} ms` and error rate `{percent(first_bad['error_rate'])}`")
lines.extend([
    "",
    "| Target RPS | Actual RPS | Requests | Error Rate | p50 | p95 | p99 | Max | Healthy |",
    "|---:|---:|---:|---:|---:|---:|---:|---:|---|",
])
for row in rows:
    lines.append(
        f"| {row['target_rps']:.2f} | {row['actual_rps']:.2f} | {row['requests']} | {percent(row['error_rate'])} | "
        f"{row['p50_ms']:.2f} ms | {row['p95_ms']:.2f} ms | {row['p99_ms']:.2f} ms | {row['max_ms']:.2f} ms | {row['healthy']} |"
    )
lines.extend(["", "## Raw Reports", ""])
for row in rows:
    lines.append(f"- Target `{row['target_rps']:.2f} req/s`: `{row['markdown_report']}`")
Path(summary_md_path).write_text("\n".join(lines) + "\n", encoding="utf-8")
PY

echo "Ladder JSON report: $SUMMARY_JSON_PATH"
echo "Ladder Markdown report: $SUMMARY_MD_PATH"
