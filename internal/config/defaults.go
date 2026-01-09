package config

import "time"

func applyDefaults(cfg *Config) {
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}

	if cfg.Reload.Timeout == 0 {
		cfg.Reload.Timeout = 5 * time.Second
	}
	if cfg.Reload.Command == "" {
		cfg.Reload.Command = "reconfigure"
	}
}
