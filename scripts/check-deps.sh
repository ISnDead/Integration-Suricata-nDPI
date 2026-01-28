#!/usr/bin/env bash
set -euo pipefail

# -----------------------------
# Проверка зависимостей и уязвимостей через SBOM + Trivy
# -----------------------------
# Режимы:
#   image: скан контейнерного образа (рекомендуется для CI)
#   fs:    скан файловой системы по пути (например, распакованный rootfs)
#
# Артефакты:
#   artifacts/sbom.cdx.json         SBOM в формате CycloneDX (список компонентов/зависимостей)
#   artifacts/vuln.report.json      JSON-отчёт Trivy по уязвимостям (с фильтром по SEVERITY)
#   artifacts/vuln.report.txt       Человекочитаемый отчёт (таблица) из JSON
#
# Коды выхода:
#   0 - ОК (нет находок уровня SEVERITY и выше)
#   1 - Найдены уязвимости по политике (pipeline должен упасть)
#   2 - Ошибка использования / не хватает утилит / внутренняя ошибка Trivy

MODE="${MODE:-image}"                  # image|fs
TARGET="${TARGET:-}"                   # имя образа (image:tag) или путь к директории (fs)
OUT_DIR="${OUT_DIR:-artifacts}"

# Политика
SEVERITY="${SEVERITY:-HIGH,CRITICAL}"  # при каких уровнях "падаем"
EXIT_CODE_ON_FINDINGS="${EXIT_CODE_ON_FINDINGS:-1}"

# Поведение Trivy
SCANNERS="${SCANNERS:-vuln}"           # vuln | vuln,license (если нужно проверять ещё и лицензии)
TRIVY_TIMEOUT="${TRIVY_TIMEOUT:-5m}"
TRIVY_CACHE_DIR="${TRIVY_CACHE_DIR:-$OUT_DIR/.trivy-cache}"

# Имена файлов
SBOM_FILE="${SBOM_FILE:-$OUT_DIR/sbom.cdx.json}"
JSON_REPORT="${JSON_REPORT:-$OUT_DIR/vuln.report.json}"
TXT_REPORT="${TXT_REPORT:-$OUT_DIR/vuln.report.txt}"

usage() {
  cat <<EOF
Использование:
  MODE=image TARGET=<image:tag> ./scripts/check-deps.sh
  MODE=fs    TARGET=/path/to/rootfs ./scripts/check-deps.sh

Переменные окружения:
  MODE=image|fs
  TARGET=...
  OUT_DIR=artifacts
  SEVERITY=HIGH,CRITICAL
  EXIT_CODE_ON_FINDINGS=1
  SCANNERS=vuln | vuln,license
  TRIVY_TIMEOUT=5m
  TRIVY_CACHE_DIR=artifacts/.trivy-cache

Примеры:
  MODE=image TARGET=registry.local/suri-ndpi:1.2.3 ./scripts/check-deps.sh
  MODE=fs TARGET=/ ./scripts/check-deps.sh
EOF
}

log()  { echo "[check-deps] $*"; }
err()  { echo "[check-deps][ОШИБКА] $*" >&2; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || { err "Не найдена команда: $1"; exit 2; }
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
  fi

  need_cmd trivy
  mkdir -p "$OUT_DIR" "$TRIVY_CACHE_DIR"

  if [[ -z "$TARGET" ]]; then
    err "Не задан TARGET"
    usage
    exit 2
  fi

  export TRIVY_CACHE_DIR
  export TRIVY_TIMEOUT

  # 1) Генерируем SBOM (CycloneDX)
  log "Генерация SBOM (CycloneDX) -> $SBOM_FILE"
  if [[ "$MODE" == "image" ]]; then
    trivy image --format cyclonedx --output "$SBOM_FILE" "$TARGET"
  elif [[ "$MODE" == "fs" ]]; then
    [[ -d "$TARGET" ]] || { err "TARGET не существует или не директория: $TARGET"; exit 2; }
    trivy fs --format cyclonedx --output "$SBOM_FILE" "$TARGET"
  else
    err "Неизвестный MODE=$MODE (доступно: image|fs)"
    exit 2
  fi

  # 2) Сканируем SBOM по уязвимостям (и/или лицензиям) + применяем политику выхода
  log "Сканирование SBOM Trivy (scanners=$SCANNERS severity=$SEVERITY)"
  set +e
  trivy sbom "$SBOM_FILE" \
    --scanners "$SCANNERS" \
    --severity "$SEVERITY" \
    --exit-code "$EXIT_CODE_ON_FINDINGS" \
    --format json \
    --output "$JSON_REPORT"
  rc=$?
  set -e

  # 3) Делаем человекочитаемый отчёт из JSON
  log "Конвертация отчёта JSON -> таблица -> $TXT_REPORT"
  trivy convert --format table "$JSON_REPORT" > "$TXT_REPORT" || true

  # Итог
  log "Артефакты:"
  log "  SBOM:   $SBOM_FILE"
  log "  JSON:   $JSON_REPORT"
  log "  TABLE:  $TXT_REPORT"

  if [[ "$rc" -eq 0 ]]; then
    log "ОК: уязвимостей уровня $SEVERITY не найдено"
    exit 0
  elif [[ "$rc" -eq "$EXIT_CODE_ON_FINDINGS" ]]; then
    err "ПОЛИТИКА НЕ ПРОЙДЕНА: найдены уязвимости уровня $SEVERITY (exit $rc)"
    exit "$rc"
  else
    err "Ошибка Trivy (exit $rc). Смотрите логи/отчёт."
    exit 2
  fi
}

main "$@"
