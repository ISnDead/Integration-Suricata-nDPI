package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ValidateNDPIConfig проверяет наличие локальных ресурсов (папка правил nDPI).
func ValidateNDPIConfig() error {
	logger.Log.Info("Проверка локальных ресурсов nDPI",
		zap.String("rules_path", NDPIRulesLocalPath))

	// Проверяем, что папка существует и это директория.
	info, err := os.Stat(NDPIRulesLocalPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Log.Error("Папка правил не найдена", zap.String("path", NDPIRulesLocalPath))
			return fmt.Errorf("отсутствует папка %s", NDPIRulesLocalPath)
		}
		return fmt.Errorf("не удалось проверить папку %s: %w", NDPIRulesLocalPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("путь %s не является директорией", NDPIRulesLocalPath)
	}

	// Проверяем, есть ли файлы правил (пустая папка — не фатально, но предупреждаем).
	files, err := filepath.Glob(filepath.Join(NDPIRulesLocalPath, "*"))
	if err != nil {
		logger.Log.Error("Ошибка чтения списка файлов правил", zap.Error(err))
		return fmt.Errorf("ошибка чтения файлов в %s: %w", NDPIRulesLocalPath, err)
	}

	if len(files) == 0 {
		logger.Log.Warn("Папка правил пуста", zap.String("path", NDPIRulesLocalPath))
	} else {
		logger.Log.Info("Файлы правил найдены", zap.Int("file_count", len(files)))
	}

	return nil
}
