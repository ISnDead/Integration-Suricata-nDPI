package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

// ApplyConfig выполняет рендеринг конфигурации и синхронизацию изменений с системным файлом Suricata.
// После успешного обновления отправляет команду на горячую перезагрузку (hot reload).
func ApplyConfig(client *SuricataClient) error {
	logger.Log.Info("Запуск процесса применения конфигурации nDPI")

	// 1. Чтение шаблона конфигурации из локальных ресурсов проекта.
	tmplData, err := os.ReadFile(SuricataTemplatePath)
	if err != nil {
		logger.Log.Error("Ошибка чтения шаблона конфигурации",
			zap.String("path", SuricataTemplatePath),
			zap.Error(err))
		return fmt.Errorf("не удалось прочитать шаблон: %w", err)
	}

	// 2. Генерация финального конфига (пока просто копия шаблона).
	finalConfig := tmplData

	// 3. Определяем, где лежит рабочий suricata.yaml (смешанная установка).
	targetPath, err := FirstExistingPath(SuricataConfigCandidates)
	if err != nil {
		return fmt.Errorf("не найден suricata.yaml (ни /etc, ни /usr/local): %w", err)
	}

	// 4. Атомарная запись (tmp -> rename), чтобы не получить битый конфиг.
	tmpPath := filepath.Join(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp")
	if err := os.WriteFile(tmpPath, finalConfig, 0644); err != nil {
		logger.Log.Error("Критическая ошибка при записи временного конфига",
			zap.String("tmp", tmpPath),
			zap.Error(err))
		return fmt.Errorf("ошибка записи временного файла: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		logger.Log.Error("Критическая ошибка при замене системного конфига",
			zap.String("tmp", tmpPath),
			zap.String("target", targetPath),
			zap.Error(err))
		return fmt.Errorf("ошибка обновления системного файла: %w", err)
	}

	logger.Log.Info("Системный файл конфигурации успешно обновлен",
		zap.String("path", targetPath))

	// 5. Hot reload
	_ = client

	if err := hotReloadSuricata(); err != nil {
		return err
	}

	logger.Log.Info("Конфигурация nDPI успешно применена и активирована")
	return nil
}

func hotReloadSuricata() error {
	path, err := exec.LookPath("suricatasc")
	if err != nil {
		return fmt.Errorf("suricatasc не найден в PATH: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "-c", "reconfigure")
	out, err := cmd.CombinedOutput()

	logger.Log.Info("suricatasc output", zap.ByteString("out", out))

	if err != nil {
		return fmt.Errorf("suricatasc reconfigure failed: %w", err)
	}
	return nil
}
