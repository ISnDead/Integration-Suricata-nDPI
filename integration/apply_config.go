package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ApplyConfig применяет конфигурацию Suricata без рестарта службы:
//
//  1. Читает шаблон suricata.yaml.tpl из репозитория.
//  2. Определяет системный suricata.yaml (из списка кандидатов).
//  3. Записывает конфиг атомарно.
//  4. Делает best-effort reload/reconfigure через suricatasc.
//  5. При ошибке/таймауте проверяет, что Suricata остаётся доступной через unix-socket.
func ApplyConfig(
	templatePath string,
	configCandidates []string,
	socketCandidates []string,
	suricatascPath string,
	reloadCommand string,
	reloadTimeout time.Duration,
) (ApplyConfigReport, error) {
	report := ApplyConfigReport{
		ReloadCommand: reloadCommand,
		ReloadTimeout: reloadTimeout,
	}

	logger.Log.Info("Применение конфигурации Suricata (safe apply, no restart)",
		zap.String("template_path", templatePath),
		zap.Strings("config_candidates", configCandidates),
		zap.Strings("socket_candidates", socketCandidates),
		zap.String("suricatasc", suricatascPath),
		zap.String("reload_command", reloadCommand),
		zap.Duration("reload_timeout", reloadTimeout),
	)

	cmdNormalized := strings.TrimSpace(strings.ToLower(reloadCommand))
	if cmdNormalized == "shutdown" {
		return report, fmt.Errorf("reload_command=shutdown запрещён")
	}
	if cmdNormalized == "" || cmdNormalized == "none" {
		report.ReloadStatus = ReloadOK
		report.Warnings = append(report.Warnings, "reload_command пустой/none: конфиг записан, reload не выполнялся")
		logger.Log.Warn("reload_command пустой/none",
			zap.String("reload_command", reloadCommand),
		)
		return report, nil
	}

	tmplData, err := os.ReadFile(templatePath)
	if err != nil {
		return report, fmt.Errorf("не удалось прочитать шаблон %s: %w", templatePath, err)
	}

	targetConfigPath, err := FirstExistingPath(configCandidates)
	if err != nil {
		return report, fmt.Errorf("не найден системный suricata.yaml среди кандидатов: %w", err)
	}
	report.TargetConfigPath = targetConfigPath

	if err := writeFileAtomic(targetConfigPath, tmplData, 0o644); err != nil {
		return report, fmt.Errorf("не удалось записать конфиг %s: %w", targetConfigPath, err)
	}
	logger.Log.Info("Конфиг Suricata обновлён", zap.String("path", targetConfigPath))

	var ctx context.Context
	var cancel func()
	if reloadTimeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), reloadTimeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, suricatascPath, "-c", reloadCommand)
	out, err := cmd.CombinedOutput()
	report.ReloadOutput = strings.TrimSpace(string(out))

	// Таймаут suricatasc
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		report.ReloadStatus = ReloadTimeout
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("suricatasc timeout: command=%q timeout=%s", reloadCommand, reloadTimeout),
		)

		logger.Log.Warn("suricatasc не ответил за таймаут (restart запрещён). Проверяем что Suricata доступна по сокету",
			zap.String("command", reloadCommand),
			zap.Duration("timeout", reloadTimeout),
		)

		if err2 := EnsureSuricataRunning(socketCandidates); err2 != nil {
			return report, fmt.Errorf("suricatasc timeout и Suricata недоступна по сокету: %w", err2)
		}

		logger.Log.Warn("reload не подтверждён из-за timeout, но Suricata доступна по сокету — продолжаем")
		return report, nil
	}

	// Ошибка выполнения suricatasc
	if err != nil {
		report.ReloadStatus = ReloadFailed
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("suricatasc error: command=%q err=%v output=%q", reloadCommand, err, report.ReloadOutput),
		)

		logger.Log.Error("suricatasc завершился с ошибкой (restart запрещён)",
			zap.String("command", reloadCommand),
			zap.String("output", report.ReloadOutput),
			zap.Error(err),
		)

		if err2 := EnsureSuricataRunning(socketCandidates); err2 != nil {
			return report, fmt.Errorf("ошибка suricatasc и Suricata недоступна по сокету: %w", err2)
		}

		logger.Log.Warn("reload завершился с ошибкой, но Suricata доступна по сокету — продолжаем")
		return report, nil
	}

	report.ReloadStatus = ReloadOK
	logger.Log.Info("Suricata успешно применила изменения",
		zap.String("command", reloadCommand),
		zap.String("output", report.ReloadOutput),
	)

	return report, nil
}
