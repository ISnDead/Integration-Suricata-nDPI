package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"integration-suricata-ndpi/integration"
	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/hostagent"
	"integration-suricata-ndpi/pkg/logger"
)

func main() {
	logger.Init()
	defer logger.Sync()

	cfgPath := flag.String("config", "config/config.yaml", "Path to config file")
	sock := flag.String("sock", "/run/ndpi-agent.sock", "Path to unix socket")
	unit := flag.String("unit", "suricata", "Systemd unit name")
	suricataCfg := flag.String("suricata-config", "", "Override Suricata config path (optional)")
	ndpiPlugin := flag.String("ndpi-plugin", "", "Override ndpi plugin path (optional)")
	restartTimeout := flag.Duration("restart-timeout", 20*time.Second, "systemctl restart timeout")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		logger.Fatalw("Failed to load config", "error", err)
	}

	suricataCfgPath := *suricataCfg
	if suricataCfgPath == "" {
		p, err := integration.FirstExistingPath(cfg.Suricata.ConfigCandidates)
		if err != nil {
			logger.Fatalw("Cannot find Suricata config file", "error", err)
		}
		suricataCfgPath = p
	}

	ndpiPluginPath := *ndpiPlugin
	if ndpiPluginPath == "" {
		ndpiPluginPath = cfg.Paths.NDPIPluginPath
	}

	deps := hostagent.Deps{
		SocketPath:      *sock,
		SuricataCfgPath: suricataCfgPath,
		NDPIPluginPath:  ndpiPluginPath,
		SuricataUnit:    *unit,
		RestartTimeout:  *restartTimeout,
	}

	srv, err := hostagent.New(deps)
	if err != nil {
		logger.Fatalw("Failed to init host agent", "error", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Start(ctx); err != nil {
		logger.Fatalw("Host agent crashed", "error", err)
	}
}
