package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ValidateLocalResources проверяет, что в репозитории есть нужные файлы:
// 1) папка с правилами nDPI
// 2) шаблон конфигурации Suricata
func ValidateLocalResources(ndpiRulesDir string, templatePath string) error {
	logger.Log.Info("Валидация локальных ресурсов",
		zap.String("ndpi_rules_dir", ndpiRulesDir),
		zap.String("template_path", templatePath),
	)

	// 1) Папка правил nDPI
	info, err := os.Stat(ndpiRulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("не найдена папка правил nDPI: %s", ndpiRulesDir)
		}
		return fmt.Errorf("ошибка доступа к папке правил nDPI (%s): %w", ndpiRulesDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("путь правил nDPI не является директорией: %s", ndpiRulesDir)
	}

	// Не фатально, но полезно предупредить
	files, err := filepath.Glob(filepath.Join(ndpiRulesDir, "*"))
	if err != nil {
		return fmt.Errorf("не удалось прочитать файлы в папке правил (%s): %w", ndpiRulesDir, err)
	}
	if len(files) == 0 {
		logger.Log.Warn("Папка правил nDPI пустая", zap.String("path", ndpiRulesDir))
	}

	// 2) Шаблон Suricata
	tmplInfo, err := os.Stat(templatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("не найден шаблон конфигурации: %s", templatePath)
		}
		return fmt.Errorf("ошибка доступа к шаблону (%s): %w", templatePath, err)
	}
	if tmplInfo.IsDir() {
		return fmt.Errorf("шаблон конфигурации не должен быть директорией: %s", templatePath)
	}

	logger.Log.Info("Локальные ресурсы валидны")
	return nil
}
