package cli

import (
	"context"
	"time"

	"github.com/urfave/cli/v2"

	"integration-suricata-ndpi/internal/app"
	"integration-suricata-ndpi/internal/wire"
)

func NewIntegrationApp() *cli.App {
	return &cli.App{
		Name:  "integration",
		Usage: "Suricata nDPI integration workflow",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run integration workflow and wait for shutdown",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "config",
						Value: "config/config.yaml",
						Usage: "Path to config file",
					},
					&cli.DurationFlag{
						Name:  "shutdown-timeout",
						Value: 10 * time.Second,
						Usage: "Graceful shutdown timeout",
					},
				},
				Action: func(c *cli.Context) error {
					svc, err := wire.InitializeIntegrationService(c.String("config"))
					if err != nil {
						return err
					}
					return app.RunWithSignals(context.Background(), svc, c.Duration("shutdown-timeout"))
				},
			},
		},
	}
}
