#!/usr/bin/env bash
set -euo pipefail

# -----------------------------
# SBOM + Trivy dependency/vuln check
# -----------------------------
# Modes:
#   image: scan container image reference (recommended in CI)
#   fs:    scan filesystem path (e.g. unpacked rootfs)
#
# Outputs:
#   artifacts/sbom.cdx.json         CycloneDX SBOM
#   artifacts/vuln.report.json      Trivy JSON report (filtered by severity)
#   artifacts/vuln.report.txt       Human-readable table
#
# Exit codes:
#   0 - OK (no findings at/above severity threshold)
#   1 - Findings detected (policy violated)
#   2 - Script usage error / missing tooling

MODE="${MODE:-image}"                  # image|fs
TARGET="${TARGET:-}"                   # image ref OR filesystem path
OUT_DIR="${OUT_DIR:-artifacts}"

# Policy
SEVERITY="${SEVERITY:-HIGH,CRITICAL}"  # what counts as failure
EXIT_CODE_ON_FINDINGS="${EXIT_CODE_ON_FINDINGS:-1}"

# Trivy behavior
SCANNERS="${SCANNERS:-vuln}"           # vuln | vuln,license (if you want license findings too)
TRIVY_TIMEOUT="${TRIVY_TIMEOUT:-5m}"
TRIVY_CACHE_DIR="${TRIVY_CACHE_DIR:-$OUT_DIR/.trivy-cache}"

# Naming
SBOM_FILE="${SBOM_FILE:-$OUT_DIR/sbom.cdx.json}"
JSON_REPORT="${JSON_REPORT:-$OUT_DIR/vuln.report.json}"
TXT_REPORT="${TXT_REPORT:-$OUT_DIR/vuln.report.txt}"

usage() {
  cat <<EOF
Usage:
  MODE=image TARGET=<image:tag> ./scripts/check-deps.sh
  MODE=fs    TARGET=/path/to/rootfs ./scripts/check-deps.sh

Env knobs:
  MODE=image|fs
  TARGET=...
  OUT_DIR=artifacts
  SEVERITY=HIGH,CRITICAL
  EXIT_CODE_ON_FINDINGS=1
  SCANNERS=vuln | vuln,license
  TRIVY_TIMEOUT=5m
  TRIVY_CACHE_DIR=artifacts/.trivy-cache

Examples:
  MODE=image TARGET=registry.local/suri-ndpi:1.2.3 ./scripts/check-deps.sh
  MODE=fs TARGET=/ ./scripts/check-deps.sh
EOF
}

log()  { echo "[check-deps] $*"; }
err()  { echo "[check-deps][ERROR] $*" >&2; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { err "Missing command: $1"; exit 2; }
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage; exit 0
  fi

  need_cmd trivy
  mkdir -p "$OUT_DIR" "$TRIVY_CACHE_DIR"

  if [[ -z "$TARGET" ]]; then
    err "TARGET is required"
    usage
    exit 2
  fi

  export TRIVY_CACHE_DIR
  export TRIVY_TIMEOUT

  # 1) Generate SBOM (CycloneDX)
  log "Generating SBOM (CycloneDX) -> $SBOM_FILE"
  if [[ "$MODE" == "image" ]]; then
    trivy image --format cyclonedx --output "$SBOM_FILE" "$TARGET"
  elif [[ "$MODE" == "fs" ]]; then
    [[ -d "$TARGET" ]] || { err "TARGET path does not exist or not a directory: $TARGET"; exit 2; }
    trivy fs --format cyclonedx --output "$SBOM_FILE" "$TARGET"
  else
    err "Unknown MODE=$MODE (use image|fs)"
    exit 2
  fi

  # 2) Scan SBOM with policy
  log "Scanning SBOM with Trivy (scanners=$SCANNERS severity=$SEVERITY)"
  set +e
  trivy sbom "$SBOM_FILE" \
    --scanners "$SCANNERS" \
    --severity "$SEVERITY" \
    --exit-code "$EXIT_CODE_ON_FINDINGS" \
    --format json \
    --output "$JSON_REPORT"
  rc=$?
  set -e

  # 3) Make a human-readable report from JSON
  log "Converting JSON -> table -> $TXT_REPORT"
  trivy convert --format table "$JSON_REPORT" > "$TXT_REPORT" || true

  # Summary
  log "Artifacts:"
  log "  SBOM:   $SBOM_FILE"
  log "  JSON:   $JSON_REPORT"
  log "  TABLE:  $TXT_REPORT"

  if [[ "$rc" -eq 0 ]]; then
    log "OK: no findings at/above $SEVERITY"
    exit 0
  elif [[ "$rc" -eq "$EXIT_CODE_ON_FINDINGS" ]]; then
    err "POLICY FAIL: findings at/above $SEVERITY (exit $rc)"
    exit "$rc"
  else
    err "Trivy error (exit $rc). Check logs/report."
    exit 2
  fi
}

main "$@