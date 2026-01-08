package runner

import (
	"context"
	"fmt"

	"integration-suricata-ndpi/integration"
	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// Runner координирует работу микросервиса, управляя жизненным циклом
// и последовательностью этапов интеграции nDPI в Suricata IDS.
type Runner struct {
	suricataClient *integration.SuricataClient
}

// NewRunner инициализирует новый экземпляр управляющего контроллера.
func NewRunner() *Runner {
	return &Runner{}
}

// Start запускает последовательную цепочку инициализации.
// На каждом этапе проверяется состояние контекста для обеспечения корректной остановки (Graceful Shutdown).
func (r *Runner) Start(ctx context.Context) error {
	logger.Log.Info("Запуск интеграционного процесса...")

	// ЭТАП 1: Валидация правил nDPI.
	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := integration.ValidateNDPIConfig(); err != nil {
		return fmt.Errorf("этап 1 (валидация) не пройден: %w", err)
	}

	// ЭТАП 2: Управление состоянием системной службы Suricata.
	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := integration.EnsureSuricataRunning(); err != nil {
		return fmt.Errorf("этап 2 (служба) не пройден: %w", err)
	}

	// ЭТАП 3: Установка соединения с управляющим Unix-сокетом.
	if err := r.checkContext(ctx); err != nil {
		return err
	}
	client, err := integration.ConnectSuricata()
	if err != nil {
		return fmt.Errorf("этап 3 (подключение) не пройден: %w", err)
	}
	r.suricataClient = client

	// ЭТАП 4: Применение динамической конфигурации и обновление правил.
	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := integration.ApplyConfig(r.suricataClient); err != nil {
		return fmt.Errorf("этап 4 (применение) не пройден: %w", err)
	}

	logger.Log.Info("Микросервис успешно запущен. Система находится в режиме мониторинга.")

	// Блокировка до получения сигнала завершения от ОС или родительского контекста.
	<-ctx.Done()

	// Выполняем очистку ресурсов перед выходом.
	r.Stop()
	return nil
}

// Stop выполняет деинициализацию компонентов и корректно закрывает активные соединения.
func (r *Runner) Stop() {
	logger.Log.Info("Запущена процедура остановки микросервиса...")

	if r.suricataClient != nil && r.suricataClient.Conn != nil {
		logger.Log.Info("Завершение сессии управления Suricata",
			zap.String("socket", r.suricataClient.Path))

		if err := r.suricataClient.Conn.Close(); err != nil {
			logger.Log.Error("Не удалось корректно закрыть соединение с сокетом", zap.Error(err))
		}
	}

	logger.Log.Info("Все компоненты успешно остановлены. Завершение процесса.")
}

// checkContext — вспомогательная функция для прерывания цепочки запуска при отмене контекста.
func (r *Runner) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		logger.Log.Warn("Запуск прерван внешним сигналом отмены")
		return err
	}
	return nil
}
