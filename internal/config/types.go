package config

import "time"

type Config struct {
	HTTP     HTTPConfig     `yaml:"http"`
	Paths    PathsConfig    `yaml:"paths"`
	Suricata SuricataConfig `yaml:"suricata"`
	Reload   ReloadConfig   `yaml:"reload"`
	System   SystemConfig   `yaml:"system"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type PathsConfig struct {
	NDPIRulesLocal   string `yaml:"ndpi_rules_local"`
	SuricataTemplate string `yaml:"suricata_template"`
	SuricataSC       string `yaml:"suricatasc"`
}

type SuricataConfig struct {
	SocketCandidates []string `yaml:"socket_candidates"`
	ConfigCandidates []string `yaml:"config_candidates"`
}

type ReloadConfig struct {
	Timeout time.Duration `yaml:"timeout"`
	Command string        `yaml:"command"`
}

type SystemConfig struct {
	Systemctl string `yaml:"systemctl"`

	SuricataService string `yaml:"suricata_service"`
}
