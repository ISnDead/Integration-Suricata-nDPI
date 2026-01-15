package runner

import (
	"context"
	"fmt"

	"integration-suricata-ndpi/integration"
	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/logger"
)

type Runner struct{}

func NewRunner() *Runner { return &Runner{} }

func (r *Runner) Run(ctx context.Context, configPath string) error {
	logger.Infow("Starting integration workflow")

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}

	if err := integration.ValidateLocalResources(cfg.Paths.NDPIRulesLocal, cfg.Paths.SuricataTemplate); err != nil {
		return fmt.Errorf("step 1 (validate local resources) failed: %w", err)
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
		return fmt.Errorf("step 2 (validate nDPI config) failed: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}

	if err := integration.EnsureSuricataRunning(cfg.Suricata.SocketCandidates); err != nil {
		return fmt.Errorf("step 3 (check Suricata socket) failed: %w", err)
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
		return fmt.Errorf("step 4 (apply config) failed: %w", err)
	}

	if report.ReloadStatus != integration.ReloadOK {
		logger.Warnw("Config written, but reload/reconfigure was not confirmed (best-effort)",
			"status", string(report.ReloadStatus),
			"config_path", report.TargetConfigPath,
			"command", report.ReloadCommand,
			"timeout", report.ReloadTimeout,
			"warnings", report.Warnings,
			"output", report.ReloadOutput,
		)
	} else {
		logger.Infow("Config written and reload/reconfigure succeeded",
			"config_path", report.TargetConfigPath,
			"command", report.ReloadCommand,
			"output", report.ReloadOutput,
		)
	}

	logger.Infow("Integration workflow completed; waiting for shutdown signal")
	<-ctx.Done()

	r.Stop()
	return nil
}

func (r *Runner) Stop() {
	logger.Infow("Stopping integration workflow (no resources to release)")
}

func (r *Runner) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		logger.Warnw("Startup aborted: context canceled", "error", err)
		return err
	}
	return nil
}
