package main

import (
	"os"

	"integration-suricata-ndpi/internal/cli"
	"integration-suricata-ndpi/pkg/logger"
)

func main() {
	logger.Init()
	defer logger.Sync()

	if err := cli.NewHostAgentApp().Run(os.Args); err != nil {
		logger.Fatalw("Command failed", "error", err)
	}
}
