package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"integration-suricata-ndpi/integration"
	"integration-suricata-ndpi/pkg/logger"
	"integration-suricata-ndpi/pkg/runner"
)

type integrationService struct {
	runner     *integration.Runner
	configPath string
}

func (s *integrationService) Run(ctx context.Context) error {
	return s.runner.Run(ctx, s.configPath)
}

func (s *integrationService) Stop() error {
	return s.runner.Stop()
}

func main() {
	logger.Init()
	defer logger.Sync()

	configPath := flag.String("config", "config/config.yaml", "Path to config file")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	intRunner := integration.NewRunner()

	svc := &integrationService{
		runner:     intRunner,
		configPath: *configPath,
	}

	r := runner.New(svc)

	logger.Infow("Starting service", "config", *configPath)

	if err := r.Run(ctx); err != nil {
		logger.Fatalw("Service exited with error", "error", err)
	}

	_ = r.Stop()
	logger.Infow("Service stopped")
}
