# Integration-Suricata-nDPI

Инструмент интеграции Suricata и nDPI. Репозиторий содержит два бинаря:
- `integration` — валидирует окружение, применяет шаблон Suricata и выполняет безопасный reload.
- `host-agent` — локальный агент, который включает/выключает nDPI через изменение конфигурации и рестарт сервиса.

## Структура репозитория

```
cmd/
  integration/        # entrypoint для integration
  host-agent/         # entrypoint для host-agent
internal/
  app/                # lifecycle (Start/Stop, graceful shutdown)
  cli/                # CLI (urfave/cli)
  config/             # загрузка/валидация YAML-конфига
  mocks/              # моки инфраструктурных интерфейсов
  wire/               # DI wiring (wire)
integration/          # core workflow и утилиты
pkg/
  executil/           # запуск внешних команд
  fsutil/             # абстракция файловой системы
  hostagent/          # HTTP агент
  logger/             # логирование
  netutil/            # dialer интерфейс
  systemd/            # systemd менеджер
config/
  config.yaml         # пример конфига
  suricata.yaml.tpl   # шаблон конфигурации Suricata
rules/
  manual/             # примеры правил
```

## Требования

- Go 1.22+
- Suricata и `suricatasc`
- systemd (для `host-agent`, если используется restart)

## Конфигурация

Основной конфиг: `config/config.yaml`.

Ключевые поля:
- `paths.ndpi_rules_local` — директория локальных nDPI правил.
- `paths.ndpi_plugin_path` — путь до `ndpi.so`.
- `paths.suricata_template` — путь до `suricata.yaml` шаблона.
- `paths.suricatasc` — путь до `suricatasc`.
- `suricata.socket_candidates` — кандидаты для unix сокета.
- `suricata.config_candidates` — кандидаты для `suricata.yaml`.
- `reload.command`/`reload.timeout` — команда и таймаут reload.
- `system.systemctl`/`system.suricata_service` — путь до `systemctl` и имя unit.

## Сборка

```bash
go build ./cmd/integration
go build ./cmd/host-agent
```

## Запуск

### integration

```bash
./integration run --config config/config.yaml
```

### host-agent

```bash
./host-agent serve --config config/config.yaml
```

Переопределения для `host-agent`:

```bash
./host-agent serve \
  --config config/config.yaml \
  --sock /run/ndpi-agent.sock \
  --unit suricata \
  --systemctl /usr/bin/systemctl \
  --restart-timeout 20s
```

## Graceful shutdown

Оба сервиса обрабатывают SIGINT/SIGTERM и выполняют корректную остановку.
Таймаут можно изменить флагом `--shutdown-timeout`.

## Генерация wire

Для сборки DI используется `google/wire`.

1) Установить wire:
```bash
go install github.com/google/wire/cmd/wire@latest
```

2) Сгенерировать wiring:
```bash
wire ./internal/wire
```

Файл генерации: `internal/wire/wire_gen.go`.

## Тесты

```bash
go test ./...
```
