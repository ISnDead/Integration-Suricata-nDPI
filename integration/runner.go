package integration

import (
	"context"
	"fmt"

	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/executil"
	"integration-suricata-ndpi/pkg/fsutil"
	"integration-suricata-ndpi/pkg/logger"
)

type Runner struct {
	configPath    string
	commandRunner executil.Runner
	fs            fsutil.FS
}

func NewRunner(configPath string, commandRunner executil.Runner, fs fsutil.FS) *Runner {
	if commandRunner == nil {
		commandRunner = executil.DefaultRunner{}
	}
	if fs == nil {
		fs = fsutil.OSFS{}
	}

	return &Runner{
		configPath:    configPath,
		commandRunner: commandRunner,
		fs:            fs,
	}
}

func (r *Runner) Start(ctx context.Context) error {
	logger.Infow("Starting integration workflow")

	cfg, err := config.Load(r.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config.yaml: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := ValidateLocalResources(cfg.Paths.NDPIRulesLocal, cfg.Paths.SuricataTemplate, r.fs); err != nil {
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
		FS:                   r.fs,
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
		CommandRunner:    r.commandRunner,
		FS:               r.fs,
	})
	if err != nil {
		return fmt.Errorf("step 4 (apply config) failed: %w", err)
	}

	logger.Infow("Waiting for shutdown signal")
	<-ctx.Done()

	return nil
}

func (r *Runner) Stop(ctx context.Context) error {
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
