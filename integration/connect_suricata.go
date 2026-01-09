package integration

import (
	"fmt"
	"net"
	"time"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ConnectSuricata подключается к управляющему unix-сокету Suricata.
// Путь выбирается из списка кандидатов (под разные установки /etc и /usr/local).
func ConnectSuricata(socketCandidates []string, timeout time.Duration) (*SuricataClient, error) {
	socketPath, err := FirstExistingPath(socketCandidates)
	if err != nil {
		return nil, fmt.Errorf("не найден управляющий сокет Suricata: %w", err)
	}

	logger.Log.Info("Подключение к Suricata", zap.String("socket_path", socketPath))

	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		logger.Log.Error("Не удалось подключиться к unix-сокету",
			zap.String("socket_path", socketPath),
			zap.Error(err),
		)
		return nil, fmt.Errorf("ошибка подключения к %s: %w", socketPath, err)
	}

	// Чтобы чтение/запись не зависали вечно.
	_ = conn.SetDeadline(time.Now().Add(timeout))

	logger.Log.Info("Соединение с Suricata установлено", zap.String("socket_path", socketPath))

	return &SuricataClient{
		Conn: conn,
		Path: socketPath,
	}, nil
}
