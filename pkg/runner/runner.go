package runner

import (
	"context"
	"fmt"

	"integration-suricata-ndpi/integration"
	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// Runner — оркестратор шагов интеграции nDPI с Suricata.
type Runner struct {
	suricataClient *integration.SuricataClient
}

func NewRunner() *Runner {
	return &Runner{}
}

// Start выполняет шаги запуска и ждёт отмены контекста.
// После отмены контекста вызывает Stop().
func (r *Runner) Start(ctx context.Context, configPath string) error {
	logger.Log.Info("Старт процесса интеграции")

	// Загружаем конфиг микросервиса (пути/кандидаты/команды)
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("не удалось загрузить config.yaml: %w", err)
	}

	// Шаг 1: Валидация локальных ресурсов (папка правил + шаблон Suricata).
	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := integration.ValidateLocalResources(cfg.Paths.NDPIRulesLocal, cfg.Paths.SuricataTemplate); err != nil {
		return fmt.Errorf("шаг 1 (валидация) не пройден: %w", err)
	}

	// Шаг 2: Проверка доступности Suricata по unix-сокету (без sudo/systemctl).
	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := integration.EnsureSuricataRunning(cfg.Suricata.SocketCandidates); err != nil {
		return fmt.Errorf("шаг 2 (suricata) не пройден: %w", err)
	}

	// Шаг 3: Подключение к управляющему unix-сокету Suricata.
	if err := r.checkContext(ctx); err != nil {
		return err
	}
	client, err := integration.ConnectSuricata(cfg.Suricata.SocketCandidates, cfg.Reload.Timeout)
	if err != nil {
		return fmt.Errorf("шаг 3 (подключение) не пройден: %w", err)
	}
	r.suricataClient = client

	// Шаг 4: Применение конфигурации и reload/reconfigure через suricatasc.
	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := integration.ApplyConfig(
		cfg.Paths.SuricataTemplate,
		cfg.Suricata.ConfigCandidates,
		cfg.Paths.SuricataSC,
		cfg.Reload.Command,
	); err != nil {
		return fmt.Errorf("шаг 4 (применение) не пройден: %w", err)
	}

	logger.Log.Info("Интеграция запущена, ожидание сигнала остановки")

	<-ctx.Done()
	r.Stop()
	return nil
}

// Stop освобождает ресурсы. Безопасно вызывать несколько раз.
func (r *Runner) Stop() {
	logger.Log.Info("Остановка процесса интеграции")

	if r.suricataClient != nil && r.suricataClient.Conn != nil {
		logger.Log.Info("Закрытие управляющего сокета Suricata",
			zap.String("socket", r.suricataClient.Path))

		if err := r.suricataClient.Conn.Close(); err != nil {
			logger.Log.Error("Не удалось закрыть сокет Suricata", zap.Error(err))
		}

		// Делаем Stop идемпотентным.
		r.suricataClient.Conn = nil
	}
	r.suricataClient = nil

	logger.Log.Info("Остановка завершена")
}

func (r *Runner) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		logger.Log.Warn("Запуск прерван: контекст отменён")
		return err
	}
	return nil
}
