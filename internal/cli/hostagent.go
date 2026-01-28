package cli

import (
	"context"
	"time"

	"github.com/urfave/cli/v2"

	"integration-suricata-ndpi/internal/app"
	"integration-suricata-ndpi/internal/wire"
)

func NewHostAgentApp() *cli.App {
	return &cli.App{
		Name:  "host-agent",
		Usage: "Host agent for toggling nDPI in Suricata",
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start host agent HTTP server on unix socket",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Value: "config/config.yaml",
						Usage: "Path to config file",
					},
					&cli.StringFlag{
						Name:  "sock",
						Value: "/run/ndpi-agent.sock",
						Usage: "Path to unix socket",
					},
					&cli.StringFlag{
						Name:  "unit",
						Value: "",
						Usage: "Systemd unit name (overrides config)",
					},
					&cli.StringFlag{
						Name:  "suricata-config",
						Value: "",
						Usage: "Override Suricata config path (optional)",
					},
					&cli.StringFlag{
						Name:  "ndpi-plugin",
						Value: "",
						Usage: "Override ndpi plugin path (optional)",
					},
					&cli.StringFlag{
						Name:  "systemctl",
						Value: "",
						Usage: "Path to systemctl (overrides config)",
					},
					&cli.DurationFlag{
						Name:  "restart-timeout",
						Value: 20 * time.Second,
						Usage: "systemctl restart timeout",
					},
					&cli.DurationFlag{
						Name:  "shutdown-timeout",
						Value: 10 * time.Second,
						Usage: "Graceful shutdown timeout",
					},
				},
				Action: func(c *cli.Context) error {
					opts := wire.HostAgentOptions{
						ConfigPath:      c.String("config"),
						SocketPath:      c.String("sock"),
						SuricataConfig:  c.String("suricata-config"),
						NDPIPluginPath:  c.String("ndpi-plugin"),
						SystemdUnit:     c.String("unit"),
						SystemctlPath:   c.String("systemctl"),
						RestartTimeout:  c.Duration("restart-timeout"),
						ShutdownTimeout: c.Duration("shutdown-timeout"),
					}

					svc, err := wire.InitializeHostAgentService(opts)
					if err != nil {
						return err
					}
					return app.RunWithSignals(context.Background(), svc, opts.ShutdownTimeout)
				},
			},
		},
	}
}
