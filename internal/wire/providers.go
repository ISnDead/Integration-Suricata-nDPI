package wire

import (
	"strings"
	"time"

	"integration-suricata-ndpi/integration"
	"integration-suricata-ndpi/internal/app"
	"integration-suricata-ndpi/internal/config"
	"integration-suricata-ndpi/pkg/fsutil"
	"integration-suricata-ndpi/pkg/hostagent"
	"integration-suricata-ndpi/pkg/logger"
	"integration-suricata-ndpi/pkg/systemd"
)

type HostAgentOptions struct {
	ConfigPath      string
	SocketPath      string
	SuricataConfig  string
	NDPIPluginPath  string
	SystemdUnit     string
	SystemctlPath   string
	RestartTimeout  time.Duration
	ShutdownTimeout time.Duration
}

func newIntegrationService(configPath string) (app.Service, error) {
	return integration.NewRunner(configPath, nil, fsutil.OSFS{}), nil
}

func firstNonEmpty(v []string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func newHostAgentService(opts HostAgentOptions) (app.Service, error) {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return nil, err
	}

	logger.Infow("Host-agent config loaded",
		"config_path", opts.ConfigPath,
		"socket_candidates", cfg.Suricata.SocketCandidates,
		"config_candidates", cfg.Suricata.ConfigCandidates,
		"systemctl", cfg.System.Systemctl,
		"suricata_service", cfg.System.SuricataService,
	)

	suricataCfgPath := opts.SuricataConfig
	if suricataCfgPath == "" {
		p, err := integration.FirstExistingPath(cfg.Suricata.ConfigCandidates)
		if err != nil {
			return nil, err
		}
		suricataCfgPath = p
	}

	ndpiPluginPath := opts.NDPIPluginPath
	if ndpiPluginPath == "" {
		ndpiPluginPath = cfg.Paths.NDPIPluginPath
	}

	unit := opts.SystemdUnit
	if unit == "" {
		unit = cfg.System.SuricataService
	}

	systemctlPath := opts.SystemctlPath
	if systemctlPath == "" {
		systemctlPath = cfg.System.Systemctl
	}

	deps := hostagent.Deps{
		SocketPath:    firstNonEmpty([]string{opts.SocketPath, cfg.HTTP.HostAgentSocket}),
		SystemctlPath: systemctlPath,
		SuricataUnit:  unit,

		SuricataSocketCandidates: append([]string(nil), cfg.Suricata.SocketCandidates...),

		SuricataCfgPath: suricataCfgPath,
		NDPIPluginPath:  ndpiPluginPath,

		RestartTimeout:         opts.RestartTimeout,
		SuricataConnectTimeout: 300 * time.Millisecond,

		FS:      fsutil.OSFS{},
		Systemd: systemd.NewManager(systemctlPath, nil),
	}

	return hostagent.New(deps)
}
