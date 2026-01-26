package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyDefaults(t *testing.T) {
	var cfg Config
	applyDefaults(&cfg)

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("http.addr: want :8080, got %q", cfg.HTTP.Addr)
	}
	if cfg.HTTP.HostAgentSocket != "/run/ndpi-agent.sock" {
		t.Fatalf("http.host_agent_socket: want /run/ndpi-agent.sock, got %q", cfg.HTTP.HostAgentSocket)
	}
	if cfg.HTTP.HostAgentTimeout != 10*time.Second {
		t.Fatalf("http.host_agent_timeout: want 10s, got %v", cfg.HTTP.HostAgentTimeout)
	}
	if cfg.Reload.Timeout != 5*time.Second {
		t.Fatalf("reload.timeout: want 5s, got %v", cfg.Reload.Timeout)
	}
	if cfg.Reload.Command != "reconfigure" {
		t.Fatalf("reload.command: want reconfigure, got %q", cfg.Reload.Command)
	}
	if cfg.System.Systemctl != "/usr/bin/systemctl" {
		t.Fatalf("system.systemctl: want /usr/bin/systemctl, got %q", cfg.System.Systemctl)
	}
	if cfg.System.SuricataService != "suricata" {
		t.Fatalf("system.suricata_service: want suricata, got %q", cfg.System.SuricataService)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	base := func() *Config {
		return &Config{
			Paths: PathsConfig{
				NDPIRulesLocal:   "rules/ndpi/",
				SuricataTemplate: "config/suricata.yaml.tpl",
				SuricataSC:       "/bin/true",
			},
			Suricata: SuricataConfig{
				SocketCandidates: []string{"/tmp/s.sock"},
				ConfigCandidates: []string{"/tmp/suricata.yaml"},
				StartTimeout:     time.Second,
			},
			Reload: ReloadConfig{
				Timeout: time.Second,
				Command: "reload-rules",
			},
		}
	}

	cases := []struct {
		name    string
		cfg     *Config
		wantErr string
	}{
		{
			name:    "missing ndpi_rules_local",
			cfg:     &Config{},
			wantErr: "config: paths.ndpi_rules_local is required",
		},
		{
			name: "missing suricata_template",
			cfg: func() *Config {
				c := base()
				c.Paths.SuricataTemplate = ""
				return c
			}(),
			wantErr: "config: paths.suricata_template is required",
		},
		{
			name: "missing suricatasc",
			cfg: func() *Config {
				c := base()
				c.Paths.SuricataSC = ""
				return c
			}(),
			wantErr: "config: paths.suricatasc is required",
		},
		{
			name: "missing socket_candidates",
			cfg: func() *Config {
				c := base()
				c.Suricata.SocketCandidates = nil
				return c
			}(),
			wantErr: "config: suricata.socket_candidates is required",
		},
		{
			name: "missing config_candidates",
			cfg: func() *Config {
				c := base()
				c.Suricata.ConfigCandidates = nil
				return c
			}(),
			wantErr: "config: suricata.config_candidates is required",
		},
		{
			name: "invalid start timeout",
			cfg: func() *Config {
				c := base()
				c.Suricata.StartTimeout = 0
				return c
			}(),
			wantErr: "config: suricata.start_timeout must be > 0",
		},
		{
			name: "invalid reload timeout",
			cfg: func() *Config {
				c := base()
				c.Reload.Timeout = 0
				return c
			}(),
			wantErr: "config: reload.timeout must be > 0",
		},
		{
			name: "reload command shutdown forbidden",
			cfg: func() *Config {
				c := base()
				c.Reload.Command = "shutdown"
				return c
			}(),
			wantErr: "config: reload.command=shutdown is forbidden",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.cfg)
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("want %q, got %v", tc.wantErr, err)
			}
		})
	}

	okCfg := base()
	okCfg.Reload.Command = "none"
	if err := validate(okCfg); err != nil {
		t.Fatalf("want none allowed, got %v", err)
	}

	okCfg.System.Systemctl = ""
	okCfg.System.SuricataService = ""
	okCfg.Reload.Command = "reload-rules"
	if err := validate(okCfg); err != nil {
		t.Fatalf("want system.* optional, got %v", err)
	}
}

func TestLoad_OK_WithDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")

	yml := `
paths:
  ndpi_rules_local: "rules/ndpi/"
  suricata_template: "config/suricata.yaml.tpl"
  suricatasc: "/bin/true"

suricata:
  socket_candidates: ["/tmp/sock"]
  config_candidates: ["/tmp/suricata.yaml"]

reload:
  timeout: "2s"
  command: "reload-rules"
`
	if err := os.WriteFile(p, []byte(yml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg.Reload.Timeout != 2*time.Second {
		t.Fatalf("timeout: want 2s, got %v", cfg.Reload.Timeout)
	}
	if cfg.System.Systemctl == "" || cfg.System.SuricataService == "" {
		t.Fatalf("system defaults not applied: %+v", cfg.System)
	}
}
