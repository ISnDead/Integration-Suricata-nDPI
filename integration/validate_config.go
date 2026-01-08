package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ValidateNDPIConfig выполняет предварительную проверку наличия необходимых
// конфигурационных ресурсов в локальном репозитории микросервиса.
func ValidateNDPIConfig() error {
	// NDPIRulesLocalPath определен в types.go как "rules/ndpi/"
	logger.Log.Info("Запуск валидации локальных ресурсов nDPI",
		zap.String("target_path", NDPIRulesLocalPath))

	// Проверка физического существования директории в корне проекта.
	// Ошибка на этом этапе блокирует дальнейший запуск системы.
	info, err := os.Stat(NDPIRulesLocalPath)
	if os.IsNotExist(err) {
		logger.Log.Error("Локальная директория правил не обнаружена",
			zap.String("path", NDPIRulesLocalPath))
		return fmt.Errorf("критическая ошибка: отсутствует папка %s", NDPIRulesLocalPath)
	}

	// Убеждаемся, что путь ведет именно к директории.
	if !info.IsDir() {
		return fmt.Errorf("объект по пути %s не является директорией", NDPIRulesLocalPath)
	}

	// Сканирование содержимого для определения готовности набора правил.
	// Используется Glob для поиска любых файлов в целевой папке.
	files, err := filepath.Glob(filepath.Join(NDPIRulesLocalPath, "*"))
	if err != nil {
		logger.Log.Error("Ошибка при чтении списка файлов правил", zap.Error(err))
		return err
	}

	// Логирование текущего состояния наполнения репозитория.
	// Пустая папка не является фатальной ошибкой, но требует уведомления.
	if len(files) == 0 {
		logger.Log.Warn("В локальном хранилище отсутствуют файлы правил nDPI",
			zap.String("path", NDPIRulesLocalPath))
	} else {
		logger.Log.Info("Обнаружены локальные файлы конфигурации nDPI",
			zap.Int("file_count", len(files)))
	}

	logger.Log.Info("Валидация конфигурационной среды успешно завершена")
	return nil
}
