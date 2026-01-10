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

// ApplyConfig применяет конфигурацию Suricata:
//  1. Читает шаблон suricata.yaml.tpl из репозитория.
//  2. Определяет реальный системный suricata.yaml (из списка кандидатов).
//  3. Записывает конфиг атомарно (через временный файл + rename).
//  4. Выполняет reload/reconfigure через suricatasc с таймаутом.
//     Если suricatasc завис/не ответил за timeout — делаем fallback: systemctl restart suricata.
func ApplyConfig(
	templatePath string,
	configCandidates []string,
	suricatascPath string,
	reloadCommand string,
	reloadTimeout time.Duration,
	systemctlPath string,
	suricataService string,
) error {
	logger.Log.Info("Применение конфигурации Suricata",
		zap.String("template_path", templatePath),
		zap.Strings("config_candidates", configCandidates),
		zap.String("suricatasc", suricatascPath),
		zap.String("reload_command", reloadCommand),
		zap.Duration("reload_timeout", reloadTimeout),
		zap.String("systemctl", systemctlPath),
		zap.String("suricata_service", suricataService),
	)

	// 1) Читаем шаблон из репозитория
	tmplData, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать шаблон %s: %w", templatePath, err)
	}

	// 2) Находим системный путь suricata.yaml
	targetConfigPath, err := FirstExistingPath(configCandidates)
	if err != nil {
		return fmt.Errorf("не найден системный suricata.yaml среди кандидатов: %w", err)
	}

	// 3) Атомарно пишем конфиг на место
	if err := writeFileAtomic(targetConfigPath, tmplData, 0644); err != nil {
		return fmt.Errorf("не удалось записать конфиг %s: %w", targetConfigPath, err)
	}

	logger.Log.Info("Конфиг Suricata обновлён", zap.String("path", targetConfigPath))

	// 4) Делаем reload через suricatasc, но ограничиваем по времени.
	ctx, cancel := context.WithTimeout(context.Background(), reloadTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, suricatascPath, "-c", reloadCommand)
	out, err := cmd.CombinedOutput()

	// Если вышли по таймауту — пробуем аварийный перезапуск через systemctl.
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		logger.Log.Warn("suricatasc не ответил за таймаут — fallback на systemctl restart",
			zap.Duration("timeout", reloadTimeout),
			zap.String("command", reloadCommand),
		)

		// Ограничиваем ожидание рестарта, чтобы микросервис не висел бесконечно.
		restartTimeout := 90 * time.Second
		if reloadTimeout > 0 && reloadTimeout*2 > restartTimeout {
			restartTimeout = reloadTimeout * 2
		}

		if err := restartServiceAndWaitActive(systemctlPath, suricataService, restartTimeout); err != nil {
			return fmt.Errorf("suricatasc завис по таймауту (%s), затем fallback restart %s не удался: %w",
				reloadCommand, suricataService, err)
		}

		return nil
	}

	// Обычная ошибка suricatasc (не таймаут) — возвращаем как есть.
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

// restartServiceAndWaitActive:
// 1) Делает systemctl restart --no-block (чтобы не висеть на долгом stop)
// 2) Ждёт пока сервис реально станет active (с таймаутом)
func restartServiceAndWaitActive(systemctlPath, service string, timeout time.Duration) error {
	logger.Log.Warn("Пробуем перезапустить Suricata через systemctl (no-block) и дождаться active",
		zap.String("service", service),
		zap.Duration("timeout", timeout),
	)

	// Запускаем рестарт неблокирующе: это убирает зависание на stop/start job в systemd.
	startCmd := exec.Command(systemctlPath, "--no-block", "restart", service)
	startOut, startErr := startCmd.CombinedOutput()
	if startErr != nil {
		logger.Log.Error("systemctl restart --no-block завершился с ошибкой",
			zap.String("service", service),
			zap.String("output", string(startOut)),
			zap.Error(startErr),
		)
		return fmt.Errorf("systemctl restart --no-block %s error (%s): %w", service, string(startOut), startErr)
	}

	// Ждём active с таймаутом.
	waitCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastState string

	for {
		// Быстрая проверка: is-active.
		activeCmd := exec.CommandContext(waitCtx, systemctlPath, "is-active", "--quiet", service)
		if err := activeCmd.Run(); err == nil {
			logger.Log.Info("Suricata перезапущена и находится в состоянии active (fallback)",
				zap.String("service", service),
				zap.String("output", string(startOut)),
			)
			return nil
		}

		// Если сервис стал failed — сразу вываливаемся с диагностикой.
		failedCmd := exec.CommandContext(waitCtx, systemctlPath, "is-failed", "--quiet", service)
		if err := failedCmd.Run(); err == nil {
			st, _ := getServiceState(systemctlPath, service)
			logger.Log.Error("Сервис перешёл в состояние failed",
				zap.String("service", service),
				zap.String("state", st),
			)
			return fmt.Errorf("service %s is failed (state=%s)", service, st)
		}

		// Периодически сохраняем ActiveState/SubState для ошибки по таймауту.
		if st, err := getServiceState(systemctlPath, service); err == nil && st != "" {
			lastState = st
		}

		select {
		case <-waitCtx.Done():
			logger.Log.Error("Не дождались active после systemctl restart",
				zap.String("service", service),
				zap.Duration("timeout", timeout),
				zap.String("last_state", lastState),
			)
			return fmt.Errorf("timeout waiting for %s to become active (last_state=%s)", service, lastState)
		case <-ticker.C:
		}
	}
}

// getServiceState возвращает "ActiveState/SubState" через systemctl show (для логов/диагностики).
func getServiceState(systemctlPath, service string) (string, error) {
	cmd := exec.Command(systemctlPath, "show", service, "-p", "ActiveState", "-p", "SubState")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	s := string(out)
	active := pickSystemdValue(s, "ActiveState")
	sub := pickSystemdValue(s, "SubState")
	if active == "" && sub == "" {
		return strings.TrimSpace(s), nil
	}
	if sub == "" {
		return active, nil
	}
	return active + "/" + sub, nil
}

func pickSystemdValue(text, key string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}
	return ""
}

// writeFileAtomic безопасно записывает файл:
// пишет во временный файл в той же директории и затем делает os.Rename.
// Это защищает от ситуаций, когда процесс упал и оставил "обрубок" конфига.
func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".suricata.yaml.*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	// На случай ошибки — удаляем временный файл.
	defer func() {
		_ = os.Remove(tmpName)
	}()

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
