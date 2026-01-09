package integration

import (
	"fmt"
	"os"
	"os/exec"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ApplyConfig:
// 1) читает шаблон suricata.yaml.tpl из проекта
// 2) пишет итоговый suricata.yaml в системный конфиг Suricata (выбранный из кандидатов)
// 3) делает reload/reconfigure через suricatasc
func ApplyConfig(templatePath string, configCandidates []string, suricatascPath string, reloadCommand string) error {
	logger.Log.Info("Применение конфигурации Suricata",
		zap.String("template_path", templatePath),
		zap.String("reload_command", reloadCommand),
	)

	// 1) Читаем шаблон из репозитория
	tmplData, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать шаблон %s: %w", templatePath, err)
	}

	// 2) Выбираем реальный системный suricata.yaml (под /etc и /usr/local)
	targetConfigPath, err := FirstExistingPath(configCandidates)
	if err != nil {
		return fmt.Errorf("не найден системный suricata.yaml среди кандидатов: %w", err)
	}

	// 3) Пишем конфиг (пока просто копируем шаблон как есть)
	if err := os.WriteFile(targetConfigPath, tmplData, 0644); err != nil {
		return fmt.Errorf("не удалось записать конфиг %s: %w", targetConfigPath, err)
	}

	logger.Log.Info("Конфиг Suricata обновлён", zap.String("path", targetConfigPath))

	// 4) Reload / Reconfigure через suricatasc
	cmd := exec.Command(suricatascPath, "-c", reloadCommand)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logger.Log.Error("Команда suricatasc завершилась с ошибкой",
			zap.String("suricatasc", suricatascPath),
			zap.String("command", reloadCommand),
			zap.String("output", string(out)),
			zap.Error(err),
		)
		return fmt.Errorf("ошибка suricatasc (%s): %w", string(out), err)
	}

	logger.Log.Info("Suricata успешно применила изменения",
		zap.String("command", reloadCommand),
		zap.String("output", string(out)),
	)

	return nil
}
