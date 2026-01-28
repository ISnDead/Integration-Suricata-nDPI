package config

import (
	"os"
	"path/filepath"
	"strings"
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
				SuricataBin:      "/usr/bin/suricata",
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
			name: "missing suricata_bin",
			cfg: func() *Config {
				c := base()
				c.Paths.SuricataBin = ""
				return c
			}(),
			wantErr: "config: paths.suricata_bin is required",
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
			name: "invalid start_timeout",
			cfg: func() *Config {
				c := base()
				c.Suricata.StartTimeout = 0
				return c
			}(),
			wantErr: "config: suricata.start_timeout must be > 0",
		},
		{
			name: "invalid reload_timeout",
			cfg: func() *Config {
				c := base()
				c.Reload.Timeout = 0
				return c
			}(),
			wantErr: "config: reload.timeout must be > 0",
		},
		{
			name: "reload_command_shutdown_forbidden",
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
			if err == nil {
				t.Fatalf("want error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("want %q, got %q", tc.wantErr, err.Error())
			}
		})
	}

	t.Run("none allowed", func(t *testing.T) {
		c := base()
		c.Reload.Command = " none "
		if err := validate(c); err != nil {
			t.Fatalf("want nil, got %v", err)
		}
	})
}

func TestValidate_OK_HappyPath(t *testing.T) {
	cfg := &Config{
		Paths: PathsConfig{
			NDPIRulesLocal:   "rules/ndpi/",
			SuricataTemplate: "config/suricata.yaml.tpl",
			SuricataSC:       "/bin/true",
			SuricataBin:      "/usr/bin/suricata",
		},
		Suricata: SuricataConfig{
			SocketCandidates: []string{"/tmp/s.sock"},
			ConfigCandidates: []string{"/tmp/suricata.yaml"},
			StartTimeout:     2 * time.Second,
		},
		Reload: ReloadConfig{
			Timeout: 2 * time.Second,
			Command: "reload-rules",
		},
	}
	if err := validate(cfg); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestLoad_ParsesYAML_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.yaml")

	// специально задаём минимально нужные поля, а остальное пусть заполнит applyDefaults()
	y := `
paths:
  ndpi_rules_local: "rules/ndpi/"
  ndpi_plugin_path: "/usr/local/lib/suricata/ndpi.so"
  suricata_template: "config/suricata.yaml.tpl"
  suricatasc: "/usr/local/bin/suricatasc"
  suricata_bin: "/usr/bin/suricata"
suricata:
  socket_candidates: ["/run/suricata/suricata-command.socket"]
  config_candidates: ["/etc/suricata/suricata.yaml"]
  start_timeout: 1s
reload:
  timeout: 2s
  command: "reload-rules"
`
	if err := os.WriteFile(p, []byte(strings.TrimSpace(y)), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	// дефолты должны примениться
	if cfg.HTTP.Addr == "" {
		t.Fatalf("want default http.addr, got empty")
	}
	if cfg.System.Systemctl == "" {
		t.Fatalf("want default system.systemctl, got empty")
	}
	if cfg.System.SuricataService == "" {
		t.Fatalf("want default system.suricata_service, got empty")
	}
}

func TestLoad_MissingFile_Error(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil {
		t.Fatal("expected error")
	}
}
