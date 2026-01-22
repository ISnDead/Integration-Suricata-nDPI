package integration

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/executil"
	"integration-suricata-ndpi/pkg/fsutil"
	"integration-suricata-ndpi/pkg/logger"
)

type Runner struct {
	configPath    string
	commandRunner executil.Runner
	fs            fsutil.FS
	cfg           *config.Config
	opts          RunnerOptions
	httpServer    *http.Server
	httpErrCh     chan error
	mu            sync.Mutex
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
		httpErrCh:     make(chan error, 1),
	}
}

func (r *Runner) Start(ctx context.Context) error {
	logger.Infow("Starting integration workflow")

	cfg, err := config.Load(r.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config.yaml: %w", err)
	}
	r.cfg = cfg
	r.opts = buildRunnerOptions(cfg, r.commandRunner, r.fs)

	if err := r.checkContext(ctx); err != nil {
		return err
	}
	if err := ValidateLocalResources(cfg.Paths.NDPIRulesLocal, cfg.Paths.SuricataTemplate, r.fs); err != nil {
		return fmt.Errorf("step 1 (validate local resources) failed: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}

	if err := ValidateNDPIConfig(r.opts.NDPIValidate); err != nil {
		return fmt.Errorf("step 2 (validate ndpi config) failed: %w", err)
	}

	if err := r.checkContext(ctx); err != nil {
		return err
	}

	if err := r.startHTTPServer(ctx); err != nil {
		return err
	}

	logger.Infow("Waiting for shutdown signal")

	select {
	case <-ctx.Done():
		return nil
	case err := <-r.httpErrCh:
		return fmt.Errorf("http server failed: %w", err)
	}
}

func (r *Runner) Stop(ctx context.Context) error {
	logger.Infow("Stopping integration workflow")
	if r.httpServer != nil {
		return r.httpServer.Shutdown(ctx)
	}
	return nil
}

func (r *Runner) checkContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		logger.Warnw("Run aborted: context canceled")
		return err
	}
	return nil
}
