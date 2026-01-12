package integration

import (
	"fmt"
	"net"
	"os"
	"time"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// EnsureSuricataRunning проверяет, что Suricata доступна через управляющий unix-сокет.

func EnsureSuricataRunning(socketCandidates []string) error {
	socketPath, err := FirstExistingPath(socketCandidates)
	if err != nil {
		return fmt.Errorf("suricata недоступна: управляющий сокет не найден: %w", err)
	}

	logger.Log.Info("Проверка доступности Suricata", zap.String("socket_path", socketPath))

	info, err := os.Stat(socketPath)
	if err != nil {
		return fmt.Errorf("не удалось проверить сокет suricata (%s): %w", socketPath, err)
	}

	if (info.Mode() & os.ModeSocket) == 0 {
		return fmt.Errorf("путь существует, но это не unix-сокет: %s", socketPath)
	}

	const probeTimeout = 2 * time.Second
	conn, err := net.DialTimeout("unix", socketPath, probeTimeout)
	if err != nil {
		return fmt.Errorf("suricata сокет есть, но подключиться нельзя (%s): %w", socketPath, err)
	}
	_ = conn.Close()

	logger.Log.Info("Suricata доступна по сокету", zap.String("socket_path", socketPath))
	return nil
}
