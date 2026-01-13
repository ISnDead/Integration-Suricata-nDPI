package integration

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

func ValidateNDPIConfig(
	ndpiPluginPath string,
	ndpiRulesDir string,
	suricataTemplatePath string,
	suricatascPath string,
	reloadCommand string,
	reloadTimeout time.Duration,
	expectedNdpiRulesPattern string,
) error {
	logger.Log.Info("Валидация конфигурации nDPI",
		zap.String("ndpi_plugin_path", ndpiPluginPath),
		zap.String("ndpi_rules_dir", ndpiRulesDir),
		zap.String("suricata_template", suricataTemplatePath),
		zap.String("suricatasc_path", suricatascPath),
		zap.String("reload_command", reloadCommand),
		zap.Duration("reload_timeout", reloadTimeout),
		zap.String("expected_ndpi_rules_pattern", expectedNdpiRulesPattern),
	)

	if err := mustBeFile(ndpiPluginPath, "nDPI plugin (ndpi.so)"); err != nil {
		return err
	}

	if err := mustBeDir(ndpiRulesDir, "директория правил nDPI"); err != nil {
		return err
	}
	ruleFiles, _ := filepath.Glob(filepath.Join(ndpiRulesDir, "*.rules"))
	if len(ruleFiles) == 0 {
		logger.Log.Warn("В папке правил nDPI нет файлов *.rules (пока не фатально)",
			zap.String("path", ndpiRulesDir),
		)
	}

	tpl, err := os.ReadFile(suricataTemplatePath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать шаблон Suricata (%s): %w", suricataTemplatePath, err)
	}

	tplLower := bytes.ToLower(tpl)

	if !bytes.Contains(tplLower, []byte("plugins")) {
		return fmt.Errorf("в шаблоне Suricata не найден блок plugins: (без этого ndpi не подключится)")
	}
	if !bytes.Contains(tplLower, []byte(strings.ToLower(filepath.Base(ndpiPluginPath)))) && !bytes.Contains(tplLower, []byte("ndpi.so")) {
		return fmt.Errorf("в шаблоне Suricata не найдено упоминание ndpi.so в plugins: (плагин не будет загружен)")
	}

	if expectedNdpiRulesPattern != "" {
		if !bytes.Contains(tpl, []byte(expectedNdpiRulesPattern)) {
			logger.Log.Warn("В шаблоне Suricata не найден ожидаемый паттерн правил nDPI. Это не фатал, но может сломать enable/disable через правила.",
				zap.String("pattern", expectedNdpiRulesPattern),
			)
		}
	}

	if err := mustBeFile(suricatascPath, "suricatasc"); err != nil {
		return err
	}
	if strings.TrimSpace(reloadCommand) == "" {
		return fmt.Errorf("reloadCommand пустой: нечего отправлять в suricatasc")
	}
	if reloadTimeout <= 0 {
		return fmt.Errorf("reloadTimeout должен быть > 0")
	}

	logger.Log.Info("Конфигурация nDPI выглядит валидной")
	return nil
}

func mustBeFile(path string, what string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s не найден: %s", what, path)
		}
		return fmt.Errorf("ошибка доступа (%s, %s): %w", what, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s должен быть файлом, а это директория: %s", what, path)
	}
	return nil
}

func mustBeDir(path string, what string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s не найдена: %s", what, path)
		}
		return fmt.Errorf("ошибка доступа (%s, %s): %w", what, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s должен быть директорией: %s", what, path)
	}
	return nil
}
