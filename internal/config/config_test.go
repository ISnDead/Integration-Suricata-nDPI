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
	c := &Config{}
	if err := validate(c); err == nil || err.Error() != "config: paths.ndpi_rules_local is required" {
		t.Fatalf("want missing ndpi_rules_local, got %v", err)
	}

	c.Paths.NDPIRulesLocal = "rules/ndpi/"
	if err := validate(c); err == nil || err.Error() != "config: paths.suricata_template is required" {
		t.Fatalf("want missing suricata_template, got %v", err)
	}

	c.Paths.SuricataTemplate = "config/suricata.yaml.tpl"
	if err := validate(c); err == nil || err.Error() != "config: paths.suricatasc is required" {
		t.Fatalf("want missing suricatasc, got %v", err)
	}

	c.Paths.SuricataSC = "/bin/true"
	if err := validate(c); err == nil || err.Error() != "config: suricata.socket_candidates is required" {
		t.Fatalf("want missing socket_candidates, got %v", err)
	}

	c.Suricata.SocketCandidates = []string{"/tmp/s.sock"}
	if err := validate(c); err == nil || err.Error() != "config: suricata.config_candidates is required" {
		t.Fatalf("want missing config_candidates, got %v", err)
	}

	c.Suricata.ConfigCandidates = []string{"/tmp/suricata.yaml"}
	if err := validate(c); err == nil || err.Error() != "config: reload.timeout must be > 0" {
		t.Fatalf("want invalid timeout, got %v", err)
	}

	c.Reload.Timeout = time.Second
	c.Reload.Command = "shutdown"
	if err := validate(c); err == nil || err.Error() != "config: reload.command=shutdown is forbidden" {
		t.Fatalf("want shutdown forbidden, got %v", err)
	}

	c.System.Systemctl = "/bin/systemctl"
	c.System.SuricataService = "suricata"
	c.Reload.Timeout = time.Second
	c.Reload.Command = "none"
	if err := validate(c); err != nil {
		t.Fatalf("want none allowed, got %v", err)
	}

	c.Reload.Command = "reload-rules"
	c.System.Systemctl = ""
	c.System.SuricataService = ""
	if err := validate(c); err != nil {
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
	// defaults должны заполнить system.*
	if cfg.System.Systemctl == "" || cfg.System.SuricataService == "" {
		t.Fatalf("system defaults not applied: %+v", cfg.System)
	}
}
