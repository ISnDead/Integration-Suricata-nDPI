package config

import "fmt"

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
	if cfg.Reload.Command == "" {
		return fmt.Errorf("config: reload.command is required")
	}

	if cfg.System.Systemctl == "" {
		return fmt.Errorf("system.systemctl не задан")
	}
	if cfg.System.SuricataService == "" {
		return fmt.Errorf("system.suricata_service не задан")
	}

	return nil

}
