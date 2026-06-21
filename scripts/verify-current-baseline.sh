#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$SCRIPT_DIR/.." && pwd)"

SKIP_HTTP_SMOKE=false
SKIP_CONTRACT_DOC_CHECKS=false
R2_MODE="SkipWhenMissing"
STAGE13_PORT=18130
STAGE14_PORT=18131
STAGE15_PORT=18132

usage() {
  cat <<'USAGE'
Usage: scripts/verify-current-baseline.sh [options]

Options:
  --skip-http-smoke, -SkipHttpSmoke
  --skip-contract-doc-checks, -SkipContractDocChecks
  --r2-mode MODE, -R2Mode MODE              SkipWhenMissing, Require, or Skip
  --stage13-port PORT, -Stage13Port PORT
  --stage14-port PORT, -Stage14Port PORT
  --stage15-port PORT, -Stage15Port PORT
  -h, --help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-http-smoke|-SkipHttpSmoke)
      SKIP_HTTP_SMOKE=true
      shift
      ;;
    --skip-contract-doc-checks|-SkipContractDocChecks)
      SKIP_CONTRACT_DOC_CHECKS=true
      shift
      ;;
    --r2-mode|-R2Mode)
      R2_MODE="${2:-}"
      shift 2
      ;;
    --stage13-port|-Stage13Port)
      STAGE13_PORT="${2:-}"
      shift 2
      ;;
    --stage14-port|-Stage14Port)
      STAGE14_PORT="${2:-}"
      shift 2
      ;;
    --stage15-port|-Stage15Port)
      STAGE15_PORT="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

case "$R2_MODE" in
  SkipWhenMissing|Require|Skip) ;;
  *)
    echo "R2 mode must be SkipWhenMissing, Require, or Skip" >&2
    exit 2
    ;;
esac

run_step() {
  local name="$1"
  shift
  echo "==> $name"
  "$@"
  echo "<== $name passed"
}

skip_step() {
  local name="$1"
  local reason="$2"
  echo "==> $name"
  echo "<== $name skipped: $reason"
}

cd "$REPO"

if [[ "$SKIP_CONTRACT_DOC_CHECKS" == true ]]; then
  skip_step "contract docs inventory" "SkipContractDocChecks"
else
  run_step "api contract route/auth/query inventory" "$SCRIPT_DIR/verify-api-contract-doc.sh"
  run_step "api schema fields/routes/required inventory" "$SCRIPT_DIR/verify-api-schema-doc.sh"
  run_step "http error contract inventory" "$SCRIPT_DIR/verify-http-error-contract-doc.sh"
  run_step "configuration contract inventory" "$SCRIPT_DIR/verify-config-contract-doc.sh"
  run_step "configuration semantic contract" "$SCRIPT_DIR/verify-config-semantics-doc.sh"
  run_step "migration contract inventory" "$SCRIPT_DIR/verify-migration-contract.sh"
fi

run_step "go run ./cmd/migrate up" go run ./cmd/migrate up
run_step "go test ./..." go test ./...
run_step "go build -buildvcs=false ./..." go build -buildvcs=false ./...

if [[ "$SKIP_HTTP_SMOKE" == true ]]; then
  skip_step "stage 13 content smoke" "SkipHttpSmoke"
  skip_step "stage 14 content lifecycle smoke" "SkipHttpSmoke"
else
  run_step "stage 13 content smoke" "$SCRIPT_DIR/smoke-stage-13-content-system.sh" --port "$STAGE13_PORT" --skip-migration
  run_step "stage 14 content lifecycle smoke" "$SCRIPT_DIR/smoke-stage-14-content-lifecycle.sh" --port "$STAGE14_PORT" --skip-migration
fi

case "$R2_MODE" in
  Skip)
    skip_step "stage 15 R2 smoke" "R2Mode=Skip"
    ;;
  Require)
    run_step "stage 15 R2 smoke" "$SCRIPT_DIR/smoke-stage-15-r2-upload.sh" --port "$STAGE15_PORT" --skip-migration
    ;;
  SkipWhenMissing)
    run_step "stage 15 R2 smoke or credential gate" "$SCRIPT_DIR/smoke-stage-15-r2-upload.sh" --port "$STAGE15_PORT" --skip-migration --skip-when-missing-credentials
    ;;
esac

HTTP_SMOKE_SKIPPED_JSON="false"
CONTRACT_DOC_CHECKS_SKIPPED_JSON="false"
if [[ "$SKIP_HTTP_SMOKE" == true ]]; then
  HTTP_SMOKE_SKIPPED_JSON="true"
fi
if [[ "$SKIP_CONTRACT_DOC_CHECKS" == true ]]; then
  CONTRACT_DOC_CHECKS_SKIPPED_JSON="true"
fi

python3 - <<PY
import json
print(json.dumps({
    "status": "passed",
    "r2_mode": "$R2_MODE",
    "http_smoke_skipped": json.loads("$HTTP_SMOKE_SKIPPED_JSON"),
    "contract_doc_checks_skipped": json.loads("$CONTRACT_DOC_CHECKS_SKIPPED_JSON")
}, indent=2))
PY
