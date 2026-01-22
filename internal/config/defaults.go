package config

import "time"

func applyDefaults(cfg *Config) {
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}
	if cfg.HTTP.HostAgentSocket == "" {
		cfg.HTTP.HostAgentSocket = "/run/ndpi-agent.sock"
	}
	if cfg.HTTP.HostAgentTimeout == 0 {
		cfg.HTTP.HostAgentTimeout = 10 * time.Second
	}

	if cfg.Reload.Timeout == 0 {
		cfg.Reload.Timeout = 5 * time.Second
	}
	if cfg.Reload.Command == "" {
		cfg.Reload.Command = "reconfigure"
	}

	if cfg.System.Systemctl == "" {
		cfg.System.Systemctl = "/usr/bin/systemctl"
	}
	if cfg.System.SuricataService == "" {
		cfg.System.SuricataService = "suricata"
	}
}
