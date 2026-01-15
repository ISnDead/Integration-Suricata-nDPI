package integration

import (
	"context"
	"fmt"

	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/logger"
)

type Runner struct{}

func NewRunner() *Runner { return &Runner{} }

func (r *Runner) Run(ctx context.Context, configPath string) error {
	logger.Infow("Starting integration workflow")

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config.yaml: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := ValidateLocalResources(cfg.Paths.NDPIRulesLocal, cfg.Paths.SuricataTemplate); err != nil {
		return fmt.Errorf("step 1 (validate local resources) failed: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := ValidateNDPIConfig(NDPIValidateOptions{
		NDPIPluginPath:       cfg.Paths.NDPIPluginPath,
		NDPIRulesDir:         cfg.Paths.NDPIRulesLocal,
		SuricataTemplatePath: cfg.Paths.SuricataTemplate,
		SuricataSCPath:       cfg.Paths.SuricataSC,
		ReloadCommand:        cfg.Reload.Command,
		ReloadTimeout:        cfg.Reload.Timeout,
		ExpectedRulesPattern: cfg.NDPI.ExpectedRulesPattern,
	}); err != nil {
		return fmt.Errorf("step 2 (validate ndpi config) failed: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := EnsureSuricataRunning(cfg.Suricata.SocketCandidates); err != nil {
		return fmt.Errorf("step 3 (check suricata socket) failed: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}
	_, err = ApplyConfig(ApplyConfigOptions{
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

	logger.Infow("Waiting for shutdown signal")
	<-ctx.Done()

	return r.Stop()
}

func (r *Runner) Stop() error {
	logger.Infow("Stopping integration workflow")
	return nil
}

func (r *Runner) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		logger.Warnw("Run aborted: context canceled")
		return err
	}
	return nil
}
