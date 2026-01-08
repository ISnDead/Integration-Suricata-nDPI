package integration

import (
	"fmt"
	"os"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ApplyConfig выполняет рендеринг конфигурации и синхронизацию изменений с системным файлом Suricata.
// После успешного обновления отправляет сигнал на горячую перезагрузку (hot reload) через управляющий сокет.
func ApplyConfig(client *SuricataClient) error {
	logger.Log.Info("Запуск процесса применения конфигурации nDPI")

	// 1. Чтение шаблона конфигурации из локальных ресурсов проекта.
	// Используется SuricataTemplatePath = "config/suricata.yaml.tpl".
	tmplData, err := os.ReadFile(SuricataTemplatePath)
	if err != nil {
		logger.Log.Error("Ошибка чтения шаблона конфигурации",
			zap.String("path", SuricataTemplatePath),
			zap.Error(err))
		return fmt.Errorf("не удалось прочитать шаблон: %w", err)
	}

	// 2. Генерация финального конфига.
	// На данном этапе выполняется прямая передача данных из шаблона.
	// В будущем здесь будет реализована логика внедрения правил из NDPIRulesLocalPath.
	finalConfig := tmplData

	// 3. Запись сгенерированного конфига в системную директорию Suricata.
	// Требуются права на запись в SuricataConfigPath = "/etc/suricata/suricata.yaml".
	err = os.WriteFile(SuricataConfigPath, finalConfig, 0644)
	if err != nil {
		logger.Log.Error("Критическая ошибка при записи системного конфига",
			zap.String("target", SuricataConfigPath),
			zap.Error(err))
		return fmt.Errorf("ошибка обновления системного файла: %w", err)
	}
	logger.Log.Info("Системный файл конфигурации успешно обновлен",
		zap.String("path", SuricataConfigPath))

	// 4. Отправка команды перезагрузки в Unix-сокет.
	// Команда позволяет применить изменения без остановки процесса и разрыва соединений.
	// Путь к сокету подтвержден в системе: /var/run/suricata/suricata-command.socket.
	reloadCommand := `{"command": "reconfigure"}` // Стандартная команда для Suricata Management Protocol
	_, err = client.Conn.Write([]byte(reloadCommand))
	if err != nil {
		logger.Log.Error("Ошибка передачи сигнала reload в управляющий сокет",
			zap.Error(err))
		return fmt.Errorf("не удалось выполнить горячую перезагрузку правил: %w", err)
	}

	logger.Log.Info("Конфигурация nDPI успешно применена и активирована")
	return nil
}
