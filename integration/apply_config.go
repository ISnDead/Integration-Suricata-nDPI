package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ApplyConfig применяет конфигурацию Suricata максимально безопасно (без дропа сервиса):
//
//  1) Читает шаблон suricata.yaml.tpl из репозитория.
//  2) Определяет реальный системный suricata.yaml (из списка кандидатов).
//  3) Записывает конфиг атомарно (через временный файл + rename).
//  4) Делает best-effort reload/reconfigure через suricatasc.

func ApplyConfig(
	templatePath string,
	configCandidates []string,
	suricatascPath string,
	reloadCommand string,
	reloadTimeout time.Duration,
	systemctlPath string,
	suricataService string,
) (ApplyConfigReport, error) {
	report := ApplyConfigReport{
		ReloadCommand: reloadCommand,
		ReloadTimeout: reloadTimeout,
	}

	logger.Log.Info("Применение конфигурации Suricata (safe apply, no restart)",
		zap.String("template_path", templatePath),
		zap.Strings("config_candidates", configCandidates),
		zap.String("suricatasc", suricatascPath),
		zap.String("reload_command", reloadCommand),
		zap.Duration("reload_timeout", reloadTimeout),
		zap.String("systemctl", systemctlPath),
		zap.String("suricata_service", suricataService),
	)

	cmdNormalized := strings.TrimSpace(strings.ToLower(reloadCommand))
	if cmdNormalized == "shutdown" {
		return report, fmt.Errorf("reload_command=shutdown запрещён: микросервис не должен останавливать Suricata")
	}
	if cmdNormalized == "" {
		report.ReloadStatus = ReloadOK
		report.Warnings = append(report.Warnings, "reload_command пустой: конфиг записан, reload не выполнялся")
		logger.Log.Warn("reload_command пустой — reload не выполняем (это безопасно)")
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

	if err := writeFileAtomic(targetConfigPath, tmplData, 0644); err != nil {
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

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		report.ReloadStatus = ReloadTimeout
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("suricatasc timeout: command=%q timeout=%s", reloadCommand, reloadTimeout),
		)

		logger.Log.Warn("suricatasc не ответил за таймаут (restart запрещён). Проверяем что Suricata active",
			zap.String("command", reloadCommand),
			zap.Duration("timeout", reloadTimeout),
		)

		active, state, checkErr := getServiceActiveState(systemctlPath, suricataService)
		if checkErr != nil {
			return report, fmt.Errorf("suricatasc timeout, а проверить systemctl is-active не удалось: %w", checkErr)
		}
		if !active {
			return report, fmt.Errorf("suricatasc timeout и Suricata сейчас НЕ active (state=%s)", state)
		}

		logger.Log.Warn("reload не подтверждён из-за timeout, но Suricata active — продолжаем",
			zap.String("state", state),
		)
		return report, nil
	}

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

		active, state, checkErr := getServiceActiveState(systemctlPath, suricataService)
		if checkErr != nil {
			return report, fmt.Errorf("ошибка suricatasc + не удалось проверить systemctl is-active: %w", checkErr)
		}
		if !active {
			return report, fmt.Errorf("ошибка suricatasc и Suricata сейчас НЕ active (state=%s)", state)
		}

		logger.Log.Warn("reload завершился с ошибкой, но Suricata active — продолжаем",
			zap.String("state", state),
		)
		return report, nil
	}

	report.ReloadStatus = ReloadOK
	logger.Log.Info("Suricata успешно применила изменения",
		zap.String("command", reloadCommand),
		zap.String("output", report.ReloadOutput),
	)

	return report, nil
}

// getServiceActiveState проверяет состояние systemd-сервиса.
// Возвращает:
//   - active=true/false
//   - state="active"/"inactive"/"failed"/...
//   - error только если проверить невозможно (systemctl недоступен, нет прав, и т.п.)
func getServiceActiveState(systemctlPath, service string) (bool, string, error) {
	cmd := exec.Command(systemctlPath, "is-active", service)
	out, err := cmd.CombinedOutput()
	state := strings.TrimSpace(string(out))

	if err == nil {
		return state == "active", state, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return state == "active", state, nil
	}

	return false, state, fmt.Errorf("systemctl is-active failed: %w (output=%q)", err, state)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".suricata.yaml.*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}
