package runner

import (
	"context"
	"fmt"

	"integration-suricata-ndpi/integration"
	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/logger"

	"go.uber.org/zap"
)

type Runner struct{}

func NewRunner() *Runner { return &Runner{} }

func (r *Runner) Start(ctx context.Context, configPath string) error {
	logger.Log.Info("Старт процесса интеграции")

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("не удалось загрузить config.yaml: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := integration.ValidateLocalResources(cfg.Paths.NDPIRulesLocal, cfg.Paths.SuricataTemplate); err != nil {
		return fmt.Errorf("шаг 1 (валидация локальных ресурсов) не пройден: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}

	if err := integration.ValidateNDPIConfig(integration.NDPIValidateOptions{
		NDPIPluginPath:       cfg.Paths.NDPIPluginPath,
		NDPIRulesDir:         cfg.Paths.NDPIRulesLocal,
		SuricataTemplatePath: cfg.Paths.SuricataTemplate,
		SuricataSCPath:       cfg.Paths.SuricataSC,
		ReloadCommand:        cfg.Reload.Command,
		ReloadTimeout:        cfg.Reload.Timeout,
		ExpectedRulesPattern: cfg.NDPI.ExpectedRulesPattern,
	}); err != nil {
		return fmt.Errorf("шаг 2 (валидация конфигурации nDPI) не пройден: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}

	if err := integration.EnsureSuricataRunning(cfg.Suricata.SocketCandidates); err != nil {
		return fmt.Errorf("шаг 3 (suricata socket) не пройден: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}

	report, err := integration.ApplyConfig(integration.ApplyConfigOptions{
		TemplatePath:     cfg.Paths.SuricataTemplate,
		ConfigCandidates: cfg.Suricata.ConfigCandidates,
		SocketCandidates: cfg.Suricata.SocketCandidates,
		SuricataSCPath:   cfg.Paths.SuricataSC,
		ReloadCommand:    cfg.Reload.Command,
		ReloadTimeout:    cfg.Reload.Timeout,
	})

	if err != nil {
		return fmt.Errorf("шаг 4 (apply config) не пройден: %w", err)
	}

	if report.ReloadStatus != integration.ReloadOK {
		logger.Log.Warn("Конфиг записан, но reload/reconfigure не подтверждён (best-effort)",
			zap.String("status", string(report.ReloadStatus)),
			zap.String("config_path", report.TargetConfigPath),
			zap.String("command", report.ReloadCommand),
			zap.Duration("timeout", report.ReloadTimeout),
			zap.Strings("warnings", report.Warnings),
			zap.String("output", report.ReloadOutput),
		)
	} else {
		logger.Log.Info("Конфиг записан и reload/reconfigure успешен",
			zap.String("config_path", report.TargetConfigPath),
			zap.String("command", report.ReloadCommand),
			zap.String("output", report.ReloadOutput),
		)
	}

	logger.Log.Info("Интеграция запущена, ожидание сигнала остановки")
	<-ctx.Done()

	r.Stop()
	return nil
}

func (r *Runner) Stop() {
	logger.Log.Info("Остановка процесса интеграции: ресурсов для освобождения нет")
}

func (r *Runner) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		logger.Log.Warn("Запуск прерван: контекст отменён")
		return err
	}
	return nil
}
