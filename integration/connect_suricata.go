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
	// Используем путь, обнаруженный в системе: /var/run/suricata/suricata-command.socket
	socketPath := SocketPath

	logger.Log.Info("Начало процесса подключения к Suricata",
		zap.String("socket_path", socketPath))

	// Устанавливаем соединение с таймаутом, чтобы сервис не завис
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		logger.Log.Error("Не удалось открыть Unix-сокет",
			zap.String("path", socketPath),
			zap.Error(err))

		return nil, fmt.Errorf("ошибка подключения к %s: %v", socketPath, err)
	}

	logger.Log.Info("Соединение с Suricata Socket успешно установлено",
		zap.String("status", "connected"))

	// Возвращаем структуру клиента, определенную в types.go
	return &SuricataClient{
		Conn: conn,
		Path: socketPath,
	}, nil
}
