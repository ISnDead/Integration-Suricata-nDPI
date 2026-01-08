package integration

import (
	"fmt"
	"net"
	"time"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ConnectSuricata выполняет подключение к сокету управления Suricata.
// Это необходимо для реализации "apply без downtime".
func ConnectSuricata() (*SuricataClient, error) {
	socketPath, err := FirstExistingPath(SuricataSocketCandidates)
	if err != nil {
		return nil, fmt.Errorf("не найден управляющий сокет Suricata: %w", err)
	}

	logger.Log.Info("Начало процесса подключения к Suricata",
		zap.String("socket_path", socketPath))

	// Устанавливаем соединение с таймаутом, чтобы сервис не завис
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		logger.Log.Error("Не удалось открыть Unix-сокет",
			zap.String("path", socketPath),
			zap.Error(err))
		return nil, fmt.Errorf("ошибка подключения к %s: %w", socketPath, err)
	}

	// Чтобы чтение/запись по сокету не зависали бесконечно
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		logger.Log.Warn("Не удалось установить deadline для сокета", zap.Error(err))
	}

	logger.Log.Info("Соединение с Suricata Socket успешно установлено",
		zap.String("status", "connected"))

	return &SuricataClient{
		Conn: conn,
		Path: socketPath,
	}, nil
}
