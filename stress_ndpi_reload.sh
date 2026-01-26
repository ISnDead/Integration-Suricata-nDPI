#!/usr/bin/env bash
set -euo pipefail

SOCK="/run/ndpi-agent.sock"
BASE="http://localhost"
SC="/usr/local/bin/suricatasc"
SURI_SOCK="/run/suricata/suricata-command.socket"

fail() {
  echo
  echo "=== FAILED ==="
  echo "time: $(date -Is)"
  echo "--- suricata status"
  systemctl status suricata --no-pager -l | sed -n '1,120p' || true
  echo "--- last suricata logs"
  journalctl -u suricata -n 120 --no-pager || true
  echo "--- last ndpi-agent logs"
  journalctl -u ndpi-agent.service -n 200 --no-pager || true
  echo "--- socket check"
  ss -xlp | rg -n "suricata-command|ndpi-agent" || true
  echo "--- manual suricatasc reload-rules (control)"
  sudo "$SC" -c reload-rules "$SURI_SOCK" || true
  exit 1
}

post() {
  local path="$1"
  sudo curl -sS -X POST --unix-socket "$SOCK" "$BASE$path"
}

echo "=== precheck: suricata socket"
sudo ss -xlp | rg -n "suricata-command" || fail
echo "=== precheck: manual suricatasc uptime"
sudo "$SC" -c uptime "$SURI_SOCK" || fail

for i in $(seq 1 20); do
  echo
  echo "===== ROUND $i ====="

  echo "[1] enable"
  post /ndpi/enable | tee /dev/stderr | rg -q '"ok": true' || fail

  echo "[2] reload via agent"
  out="$(post /suricata/reload || true)"
  echo "$out"
  echo "$out" | rg -q '"ok": true' || fail

  echo "[3] disable"
  post /ndpi/disable | tee /dev/stderr | rg -q '"ok": true' || fail

  echo "[4] reload via agent"
  out="$(post /suricata/reload || true)"
  echo "$out"
  echo "$out" | rg -q '"ok": true' || fail

  echo "[5] quick health"
  systemctl is-active --quiet suricata || fail
  sudo "$SC" -c uptime "$SURI_SOCK" >/dev/null || fail
done

echo
echo "=== OK: 20 rounds passed ==="
