package integration

import (
	"fmt"
	"os"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// EnsureSuricataRunning проверяет, что Suricata доступна по unix-сокету.
func EnsureSuricataRunning() error {
	socketPath, err := FirstExistingPath(SuricataSocketCandidates)
	if err != nil {
		return fmt.Errorf("suricata недоступна: управляющий сокет не найден: %w", err)
	}

	logger.Log.Info("Проверка доступности Suricata", zap.String("socket_path", socketPath))

	info, err := os.Stat(socketPath)
	if err != nil {
		return fmt.Errorf("не удалось проверить сокет suricata (%s): %w", socketPath, err)
	}

	if (info.Mode() & os.ModeSocket) == 0 {
		return fmt.Errorf("suricata недоступна: путь не является unix-сокетом: %s", socketPath)
	}

	logger.Log.Info("Suricata доступна по сокету", zap.String("socket_path", socketPath))
	return nil
}
