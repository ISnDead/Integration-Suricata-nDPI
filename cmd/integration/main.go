package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"integration-suricata-ndpi/pkg/logger"
	"integration-suricata-ndpi/pkg/runner"
)

func main() {
	logger.Init()
	defer logger.Sync()

	configPath := flag.String("config", "config/config.yaml", "Path to config file")
	flag.Parse()

	logger.Infow("Starting Suricata + nDPI integration service",
		"config", *configPath,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := runner.NewRunner()
	if err := srv.Run(ctx, *configPath); err != nil {
		logger.Fatalw("Service exited with an error", "error", err)
	}

	logger.Infow("Service stopped")
}
