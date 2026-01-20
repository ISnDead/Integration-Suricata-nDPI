#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./scripts/deploy.sh --dry-run
#   sudo ./scripts/deploy.sh --apply
#
# Options:
#   --config <path>   (default: config/config.yaml)
#   --verbose

CONFIG_PATH="config/config.yaml"
MODE="dry-run"
VERBOSE=0

log()  { printf '%s\n' "$*"; }
vlog() { if [[ "$VERBOSE" == "1" ]]; then printf '%s\n' "$*"; fi; }
die()  { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --config)   CONFIG_PATH="${2:-}"; shift 2 ;;
    --dry-run)  MODE="dry-run"; shift ;;
    --apply)    MODE="apply"; shift ;;
    --verbose)  VERBOSE=1; shift ;;
    -h|--help)
      cat <<EOF
Usage:
  ./scripts/deploy.sh --dry-run [--config config/config.yaml]
  sudo ./scripts/deploy.sh --apply [--config config/config.yaml]

Options:
  --config <path>   Path to YAML config file (default: config/config.yaml)
  --dry-run         Print an execution plan and show config diff (default)
  --apply           Apply changes (requires root)
  --verbose         Print extra diagnostics
EOF
      exit 0
      ;;
    *) die "Unknown argument: $1" ;;
  esac
done

[[ -f "$CONFIG_PATH" ]] || die "Config file not found: $CONFIG_PATH"

# ---- YAML reader (prefers yq, falls back to python+PyYAML) ----
have_yq=0
if command -v yq >/dev/null 2>&1; then
  have_yq=1
fi

have_pyyaml=0
if command -v python3 >/dev/null 2>&1; then
  if python3 - <<'PY' >/dev/null 2>&1
import yaml  # PyYAML
PY
  then
    have_pyyaml=1
  fi
fi

yaml_get() {
  local expr="$1"
  if [[ "$have_yq" == "1" ]]; then
    # yq v4
    yq -r "$expr" "$CONFIG_PATH"
    return 0
  fi
  if [[ "$have_pyyaml" == "1" ]]; then
    python3 - <<PY
import yaml, sys
with open("$CONFIG_PATH","r",encoding="utf-8") as f:
    data = yaml.safe_load(f) or {}

def get(obj, path):
    cur = obj
    for part in path.split("."):
        if part == "":
            continue
        if isinstance(cur, dict) and part in cur:
            cur = cur[part]
        else:
            return None
    return cur

val = get(data, "$expr")
if val is None:
    sys.exit(0)
if isinstance(val, (list, tuple)):
    for x in val:
        print("" if x is None else str(x))
else:
    print(str(val))
PY
    return 0
  fi
  die "Missing dependency: install 'yq' (v4) OR 'python3 + PyYAML' to read $CONFIG_PATH"
}

trim() {
  local s="$1"
  s="${s#"${s%%[![:space:]]*}"}"
  s="${s%"${s##*[![:space:]]}"}"
  printf '%s' "$s"
}

SURICATA_TEMPLATE="$(yaml_get "paths.suricata_template" | head -n 1 || true)"
SURICATASC_PATH="$(yaml_get "paths.suricatasc" | head -n 1 || true)"
RELOAD_CMD="$(yaml_get "reload.command" | head -n 1 || true)"
RELOAD_TIMEOUT_RAW="$(yaml_get "reload.timeout" | head -n 1 || true)"

mapfile -t CONFIG_CANDIDATES < <(yaml_get "suricata.config_candidates" || true)
mapfile -t SOCKET_CANDIDATES < <(yaml_get "suricata.socket_candidates" || true)

SURICATA_TEMPLATE="$(trim "${SURICATA_TEMPLATE:-}")"
SURICATASC_PATH="$(trim "${SURICATASC_PATH:-}")"
RELOAD_CMD="$(trim "${RELOAD_CMD:-}")"
RELOAD_TIMEOUT_RAW="$(trim "${RELOAD_TIMEOUT_RAW:-}")"

[[ -n "$SURICATA_TEMPLATE" ]] || die "paths.suricata_template is empty"
[[ -f "$SURICATA_TEMPLATE" ]] || die "Template not found: $SURICATA_TEMPLATE"

if [[ "${#CONFIG_CANDIDATES[@]}" -eq 0 ]]; then
  die "suricata.config_candidates is empty"
fi

TARGET_CONFIG=""
for p in "${CONFIG_CANDIDATES[@]}"; do
  p="$(trim "$p")"
  [[ -z "$p" ]] && continue
  if [[ -f "$p" ]]; then
    TARGET_CONFIG="$p"
    break
  fi
done
if [[ -z "$TARGET_CONFIG" ]]; then
  TARGET_CONFIG="$(trim "${CONFIG_CANDIDATES[0]}")"
fi
[[ -n "$TARGET_CONFIG" ]] || die "Failed to resolve target Suricata config path"

# ---- Render template ----
render_template() {
  local tpl="$1"

  if ! command -v envsubst >/dev/null 2>&1; then
    cat "$tpl"
    return 0
  fi

  local vars=""
  vars="$(grep -oE '\$\{[A-Za-z_][A-Za-z0-9_]*\}' "$tpl" | sort -u | tr '\n' ' ' || true)"

  if [[ -z "${vars// }" ]]; then
    cat "$tpl"
    return 0
  fi

  vlog "envsubst whitelist: $vars"
  envsubst "$vars" < "$tpl"
}

NEW_RENDERED="$(mktemp)"
cleanup() { rm -f "$NEW_RENDERED" "${NEW_RENDERED}.diff" "${NEW_RENDERED}.tmp"; }
trap cleanup EXIT

render_template "$SURICATA_TEMPLATE" > "$NEW_RENDERED"

# ---- Plan / Diff ----
log "=== Deploy plan ==="
log "Config file:      $CONFIG_PATH"
log "Template:         $SURICATA_TEMPLATE"
log "Target config:    $TARGET_CONFIG"
log "suricatasc:       ${SURICATASC_PATH:-<empty>}"
log "reload.command:   ${RELOAD_CMD:-<empty>}"
log "reload.timeout:   ${RELOAD_TIMEOUT_RAW:-<empty>}"
if [[ "${#SOCKET_CANDIDATES[@]}" -gt 0 ]]; then
  log "socket.candidates:"
  for s in "${SOCKET_CANDIDATES[@]}"; do
    s="$(trim "$s")"
    [[ -n "$s" ]] && log "  - $s"
  done
fi

# If current config exists, show diff (best-effort)
if [[ -f "$TARGET_CONFIG" ]]; then
  if diff -u "$TARGET_CONFIG" "$NEW_RENDERED" > "${NEW_RENDERED}.diff" 2>/dev/null; then
    log "Diff: no changes"
    CHANGED=0
  else
    log "Diff: changes detected"
    CHANGED=1
    if [[ "$MODE" == "dry-run" ]]; then
      log "----- diff (current vs new) -----"
      cat "${NEW_RENDERED}.diff" || true
      log "--------------------------------"
    fi
  fi
else
  log "Target config does not exist yet: it will be created"
  CHANGED=1
fi

if [[ "$MODE" == "dry-run" ]]; then
  log "Dry-run complete."
  exit 0
fi

# ---- Apply ----
[[ "$EUID" -eq 0 ]] || die "Apply mode requires root (run with sudo)"

if [[ "$CHANGED" -eq 0 ]]; then
  log "No config changes. Skipping write and reload."
  exit 0
fi

# Preserve permissions if file exists, else default 0644.
PERM="0644"
if [[ -e "$TARGET_CONFIG" ]]; then
  PERM="$(stat -c '%a' "$TARGET_CONFIG" 2>/dev/null || echo "0644")"
fi

TMP_OUT="${NEW_RENDERED}.tmp"
umask 022
cp "$NEW_RENDERED" "$TMP_OUT"
chmod "$PERM" "$TMP_OUT" || true
mv -f "$TMP_OUT" "$TARGET_CONFIG"

log "Config written atomically: $TARGET_CONFIG"

# Reload (best-effort)
cmd_norm="$(echo "$RELOAD_CMD" | tr '[:upper:]' '[:lower:]' | xargs || true)"
if [[ -z "$cmd_norm" || "$cmd_norm" == "none" ]]; then
  log "Reload skipped (reload.command is empty/none)."
  exit 0
fi
[[ "$cmd_norm" != "shutdown" ]] || die "reload.command=shutdown is forbidden"

[[ -n "$SURICATASC_PATH" ]] || die "paths.suricatasc is empty but reload.command is set"
[[ -x "$SURICATASC_PATH" ]] || die "suricatasc is not executable: $SURICATASC_PATH"

# Timeout: accept values like 10s, 2m, etc. If empty -> 10s.
RELOAD_TIMEOUT="10s"
if [[ -n "$RELOAD_TIMEOUT_RAW" ]]; then
  RELOAD_TIMEOUT="$RELOAD_TIMEOUT_RAW"
fi

log "Reloading via suricatasc: $SURICATASC_PATH -c $RELOAD_CMD (timeout=$RELOAD_TIMEOUT)"

if command -v timeout >/dev/null 2>&1; then
  if timeout "$RELOAD_TIMEOUT" "$SURICATASC_PATH" -c "$RELOAD_CMD"; then
    log "Reload succeeded."
    exit 0
  fi
  log "Reload failed or timed out (best-effort). The config is already written."
  exit 0
else
  if "$SURICATASC_PATH" -c "$RELOAD_CMD"; then
    log "Reload succeeded."
    exit 0
  fi
  log "Reload failed (best-effort). The config is already written."
  exit 0
fi