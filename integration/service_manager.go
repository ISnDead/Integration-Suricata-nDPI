package integration

import (
	"integration-suricata-ndpi/pkg/logger"
	"os/exec"

	"go.uber.org/zap"
)

// EnsureSuricataRunning проверяет процесс и запускает его, если нужно.
func EnsureSuricataRunning() error {
	// Проверяем, запущен ли процесс (через pgrep)
	cmd := exec.Command("pgrep", "suricata")
	if err := cmd.Run(); err != nil {
		// Если pgrep вернул ошибку, значит процесса нет
		logger.Log.Info("Suricata не запущена. Попытка старта через systemctl...")

		startCmd := exec.Command("sudo", "systemctl", "start", "suricata")
		if err := startCmd.Run(); err != nil {
			logger.Log.Error("Не удалось запустить сервис Suricata", zap.Error(err))
			return err
		}

		logger.Log.Info("Сервис Suricata успешно запущен")
	} else {
		logger.Log.Info("Процесс Suricata уже активен")
	}
	return nil
}
