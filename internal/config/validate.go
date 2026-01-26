package config

import (
	"fmt"
	"strings"
)

func validate(cfg *Config) error {
	if cfg.Paths.NDPIRulesLocal == "" {
		return fmt.Errorf("config: paths.ndpi_rules_local is required")
	}
	if cfg.Paths.SuricataTemplate == "" {
		return fmt.Errorf("config: paths.suricata_template is required")
	}
	if cfg.Paths.SuricataSC == "" {
		return fmt.Errorf("config: paths.suricatasc is required")
	}
	if len(cfg.Suricata.SocketCandidates) == 0 {
		return fmt.Errorf("config: suricata.socket_candidates is required")
	}
	if len(cfg.Suricata.ConfigCandidates) == 0 {
		return fmt.Errorf("config: suricata.config_candidates is required")
	}
	if cfg.Reload.Timeout <= 0 {
		return fmt.Errorf("config: reload.timeout must be > 0")
	}

	cmd := strings.TrimSpace(strings.ToLower(cfg.Reload.Command))
	if cmd == "shutdown" {
		return fmt.Errorf("config: reload.command=shutdown is forbidden")
	}
	if cfg.Suricata.StartTimeout <= 0 {
		return fmt.Errorf("config: suricata.start_timeout must be > 0")
	}

	return nil
}
