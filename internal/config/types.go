package config

import "time"

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type PathsConfig struct {
	NDPIRulesLocal string `yaml:"ndpi_rules_local"`

	NDPIPluginPath string `yaml:"ndpi_plugin_path"`

	SuricataTemplate string `yaml:"suricata_template"`

	SuricataSC string `yaml:"suricatasc"`
}

type NDPIConfig struct {
	ExpectedRulesPattern string `yaml:"expected_rules_pattern"`

	Enabled bool `yaml:"enabled"`
}

type SuricataConfig struct {
	SocketCandidates []string `yaml:"socket_candidates"`
	ConfigCandidates []string `yaml:"config_candidates"`
}

type ApplyConfig struct {
	OverwriteSuricataYAML bool `yaml:"overwrite_suricata_yaml"`

	RestartIfYAMLChanged bool `yaml:"restart_if_yaml_changed"`
}

type ReloadConfig struct {
	Timeout time.Duration `yaml:"timeout"`
	Command string        `yaml:"command"`
}

type SystemConfig struct { // В текущей версии ApplyConfig не использует systemctl/systemd.
	Systemctl       string `yaml:"systemctl"`
	SuricataService string `yaml:"suricata_service"`
}

type Config struct {
	HTTP     HTTPConfig     `yaml:"http"`
	Paths    PathsConfig    `yaml:"paths"`
	NDPI     NDPIConfig     `yaml:"ndpi"`
	Suricata SuricataConfig `yaml:"suricata"`
	Apply    ApplyConfig    `yaml:"apply"`
	Reload   ReloadConfig   `yaml:"reload"`
	System   SystemConfig   `yaml:"system"`
}
