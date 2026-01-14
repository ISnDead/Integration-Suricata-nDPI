package integration

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeExecutable(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatalf("write exe: %v", err)
	}
	if err := os.Chmod(p, 0o755); err != nil {
		t.Fatalf("chmod exe: %v", err)
	}
	return p
}

func TestApplyConfig_ReloadOK(t *testing.T) {
	dir := t.TempDir()

	// template + target config
	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	cfg := filepath.Join(dir, "suricata.yaml")
	_ = os.WriteFile(tpl, []byte("newcfg\n"), 0o644)
	_ = os.WriteFile(cfg, []byte("oldcfg\n"), 0o644)

	// unix socket listener
	sock := filepath.Join(dir, "suricata-command.socket")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		for {
			c, e := l.Accept()
			if e != nil {
				return
			}
			_ = c.Close()
		}
	}()

	// fake suricatasc: success
	suricatasc := writeExecutable(t, dir, "suricatasc", "#!/bin/sh\necho OK\nexit 0\n")

	rep, err := ApplyConfig(tpl, []string{cfg}, []string{sock}, suricatasc, "reconfigure", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rep.ReloadStatus != ReloadOK {
		t.Fatalf("want ReloadOK, got %s", rep.ReloadStatus)
	}
}

func TestApplyConfig_ReloadFailed_ButSocketAlive(t *testing.T) {
	dir := t.TempDir()

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	cfg := filepath.Join(dir, "suricata.yaml")
	_ = os.WriteFile(tpl, []byte("newcfg\n"), 0o644)
	_ = os.WriteFile(cfg, []byte("oldcfg\n"), 0o644)

	sock := filepath.Join(dir, "suricata-command.socket")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		for {
			c, e := l.Accept()
			if e != nil {
				return
			}
			_ = c.Close()
		}
	}()

	// fake suricatasc: fail
	suricatasc := writeExecutable(t, dir, "suricatasc", "#!/bin/sh\necho FAIL\nexit 1\n")

	rep, err := ApplyConfig(tpl, []string{cfg}, []string{sock}, suricatasc, "reconfigure", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rep.ReloadStatus != ReloadFailed {
		t.Fatalf("want ReloadFailed, got %s", rep.ReloadStatus)
	}
}
func TestApplyConfig_ReloadTimeout_ButSocketAlive(t *testing.T) {
	dir := t.TempDir()

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	cfg := filepath.Join(dir, "suricata.yaml")
	_ = os.WriteFile(tpl, []byte("newcfg\n"), 0o644)
	_ = os.WriteFile(cfg, []byte("oldcfg\n"), 0o644)

	sock := filepath.Join(dir, "suricata-command.socket")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go func() {
		for {
			c, e := l.Accept()
			if e != nil {
				return
			}
			_ = c.Close()
		}
	}()

	// fake suricatasc: sleep longer than timeout
	suricatasc := writeExecutable(t, dir, "suricatasc", "#!/bin/sh\nsleep 0.2\necho LATE\nexit 0\n")

	rep, err := ApplyConfig(tpl, []string{cfg}, []string{sock}, suricatasc, "reconfigure", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rep.ReloadStatus != ReloadTimeout {
		t.Fatalf("want ReloadTimeout, got %s", rep.ReloadStatus)
	}
}
func TestApplyConfig_ReloadFailed_AndSocketDown_ReturnsError(t *testing.T) {
	dir := t.TempDir()

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	cfg := filepath.Join(dir, "suricata.yaml")
	_ = os.WriteFile(tpl, []byte("newcfg\n"), 0o644)
	_ = os.WriteFile(cfg, []byte("oldcfg\n"), 0o644)

	// сокет путь, но listener не поднимаем
	sock := filepath.Join(dir, "suricata-command.socket")

	suricatasc := writeExecutable(t, dir, "suricatasc", "#!/bin/sh\necho FAIL\nexit 1\n")

	_, err := ApplyConfig(tpl, []string{cfg}, []string{sock}, suricatasc, "reconfigure", 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected error")
	}
}
func TestWriteFileAtomic_DirMissing_Error(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "no_such_dir", "x.txt")

	err := writeFileAtomic(target, []byte("data"), 0o644)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteFileAtomic_PermApplied(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")

	if err := writeFileAtomic(p, []byte("new"), 0o600); err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("want perm 0600, got %v", info.Mode().Perm())
	}
}
func TestSuricataClientClose_NilReceiver_OK(t *testing.T) {
	var c *SuricataClient
	if err := c.Close(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestSuricataClientClose_NilConn_OK(t *testing.T) {
	c := &SuricataClient{Conn: nil, Path: "x"}
	if err := c.Close(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestMustBeFile_OK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mustBeFile(p, "file"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestMustBeFile_WhenDir_Error(t *testing.T) {
	dir := t.TempDir()
	if err := mustBeFile(dir, "file"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMustBeDir_OK(t *testing.T) {
	dir := t.TempDir()
	if err := mustBeDir(dir, "dir"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestMustBeDir_WhenFile_Error(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mustBeDir(p, "dir"); err == nil {
		t.Fatal("expected error")
	}
}
func TestMustBeFile_Missing_Error(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nope.txt")
	if err := mustBeFile(p, "file"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMustBeDir_Missing_Error(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nope-dir")
	if err := mustBeDir(p, "dir"); err == nil {
		t.Fatal("expected error")
	}
}
func TestValidateNDPIConfig_NDPISOMissing_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "missing_ndpi.so") // не создаём
	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	_ = os.WriteFile(tpl, []byte("plugins:\n  - "+ndpiSo+"\n"), 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	_ = os.WriteFile(suricatasc, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	err := ValidateNDPIConfig(
		ndpiSo,
		rulesDir,
		tpl,
		suricatasc,
		"reload-rules",
		2*time.Second,
		"var/lib/suricata/rules/ndpi/*.rules",
	)
	if err == nil {
		t.Fatal("expected error")
	}
}
func TestValidateNDPIConfig_RulesDirMissing_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	_ = os.WriteFile(ndpiSo, []byte("fake"), 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi") // не создаём

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	_ = os.WriteFile(tpl, []byte("plugins:\n  - "+ndpiSo+"\n"), 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	_ = os.WriteFile(suricatasc, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	err := ValidateNDPIConfig(
		ndpiSo,
		rulesDir,
		tpl,
		suricatasc,
		"reload-rules",
		2*time.Second,
		"var/lib/suricata/rules/ndpi/*.rules",
	)
	if err == nil {
		t.Fatal("expected error")
	}
}
func TestValidateNDPIConfig_TemplateMissing_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	_ = os.WriteFile(ndpiSo, []byte("fake"), 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "missing.tpl") // не создаём

	suricatasc := filepath.Join(dir, "suricatasc")
	_ = os.WriteFile(suricatasc, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	err := ValidateNDPIConfig(
		ndpiSo,
		rulesDir,
		tpl,
		suricatasc,
		"reload-rules",
		2*time.Second,
		"var/lib/suricata/rules/ndpi/*.rules",
	)
	if err == nil {
		t.Fatal("expected error")
	}
}
func TestValidateNDPIConfig_SuricatascMissing_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	_ = os.WriteFile(ndpiSo, []byte("fake"), 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	_ = os.WriteFile(tpl, []byte("plugins:\n  - "+ndpiSo+"\n"), 0o644)

	suricatasc := filepath.Join(dir, "missing_suricatasc") // не создаём

	err := ValidateNDPIConfig(
		ndpiSo,
		rulesDir,
		tpl,
		suricatasc,
		"reload-rules",
		2*time.Second,
		"var/lib/suricata/rules/ndpi/*.rules",
	)
	if err == nil {
		t.Fatal("expected error")
	}
}
func TestValidateNDPIConfig_ReloadCommandEmpty_OK(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	_ = os.WriteFile(ndpiSo, []byte("fake"), 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	_ = os.WriteFile(tpl, []byte("plugins:\n  - "+ndpiSo+"\n"), 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	_ = os.WriteFile(suricatasc, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	err := ValidateNDPIConfig(
		ndpiSo,
		rulesDir,
		tpl,
		suricatasc,
		"", // допустимо по текущей логике
		2*time.Second,
		"var/lib/suricata/rules/ndpi/*.rules",
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
func TestWriteFileAtomic_OverwriteExisting_OK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")

	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(p, []byte("new"), 0o644); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "new" {
		t.Fatalf("want new, got %q", string(b))
	}
}
func TestValidateNDPIConfig_TemplateWithoutRulePattern_OK(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	_ = os.WriteFile(ndpiSo, []byte("fake"), 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	// шаблон без ожидаемого rulePattern
	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	_ = os.WriteFile(tpl, []byte("plugins:\n  - "+ndpiSo+"\nrule-files:\n  - /some/other/*.rules\n"), 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	_ = os.WriteFile(suricatasc, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	err := ValidateNDPIConfig(
		ndpiSo, rulesDir, tpl, suricatasc,
		"reload-rules",
		2*time.Second,
		"var/lib/suricata/rules/ndpi/*.rules",
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
func TestValidateNDPIConfig_RulePatternEmpty_OK(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	_ = os.WriteFile(ndpiSo, []byte("fake"), 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	_ = os.WriteFile(tpl, []byte("plugins:\n  - "+ndpiSo+"\n"), 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	_ = os.WriteFile(suricatasc, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	err := ValidateNDPIConfig(
		ndpiSo,
		rulesDir,
		tpl,
		suricatasc,
		"reload-rules",
		2*time.Second,
		"", // допустимо по текущей логике
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
func TestValidateLocalResources_TemplateIsDir_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(ndpiDir, 0o755)

	tplDir := filepath.Join(dir, "tpldir")
	_ = os.MkdirAll(tplDir, 0o755) // template path = dir

	if err := ValidateLocalResources(ndpiDir, tplDir); err == nil {
		t.Fatal("expected error")
	}
}
func TestWriteFileAtomic_RenameToDir_Error(t *testing.T) {
	dir := t.TempDir()

	// target path будет директорией, а не файлом
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	err := writeFileAtomic(target, []byte("data"), 0o644)
	if err == nil {
		t.Fatal("expected error")
	}
}
func TestConnectSuricata_SocketNotFound_Error(t *testing.T) {
	_, err := ConnectSuricata([]string{"/tmp/definitely-not-exists.sock"}, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnsureSuricataRunning_PathNotSocket_Error(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "not_socket")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureSuricataRunning([]string{p}); err == nil {
		t.Fatal("expected error")
	}
}
func TestValidateLocalResources_OK(t *testing.T) {
	dir := t.TempDir()

	ndpiDir := filepath.Join(dir, "rules", "ndpi")
	if err := os.MkdirAll(ndpiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	if err := os.WriteFile(tpl, []byte("plugins:\n  - /usr/local/lib/suricata/ndpi.so\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ValidateLocalResources(ndpiDir, tpl); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateLocalResources_Missing(t *testing.T) {
	dir := t.TempDir()
	tpl := filepath.Join(dir, "x.tpl")
	_ = os.WriteFile(tpl, []byte("x"), 0o644)

	if err := ValidateLocalResources(filepath.Join(dir, "nope"), tpl); err == nil {
		t.Fatal("expected error for missing ndpi dir")
	}
	if err := ValidateLocalResources(dir, filepath.Join(dir, "nope.tpl")); err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestEnsureSuricataRunning_OK_UnixSocket(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "suricata-command.socket")

	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// accept чтобы dial не падал
	go func() {
		for {
			c, e := l.Accept()
			if e != nil {
				return
			}
			_ = c.Close()
		}
	}()

	if err := EnsureSuricataRunning([]string{sock}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestConnectSuricata_OK(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "suricata-command.socket")

	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	go func() {
		c, e := l.Accept()
		if e == nil {
			_ = c.Close()
		}
	}()

	client, err := ConnectSuricata([]string{sock}, 1*time.Second)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if client == nil || client.Conn == nil || client.Path != sock {
		t.Fatalf("bad client: %+v", client)
	}
	_ = client.Close()
}

func TestValidateNDPIConfig_OK(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	if err := os.WriteFile(ndpiSo, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(rulesDir, "test.rules"), []byte("alert any any -> any any (msg:\"x\";)"), 0o644)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	tplBody := `
plugins:
  - ` + ndpiSo + `
rule-files:
  - var/lib/suricata/rules/ndpi/*.rules
`
	if err := os.WriteFile(tpl, []byte(tplBody), 0o644); err != nil {
		t.Fatal(err)
	}

	// fake suricatasc
	suricatasc := filepath.Join(dir, "suricatasc")
	if err := os.WriteFile(suricatasc, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := ValidateNDPIConfig(
		ndpiSo,
		rulesDir,
		tpl,
		suricatasc,
		"reload-rules",
		2*time.Second,
		"var/lib/suricata/rules/ndpi/*.rules",
	)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestApplyConfig_RejectShutdown(t *testing.T) {
	_, err := ApplyConfig("x", []string{"y"}, []string{"/tmp/no.sock"}, "/bin/true", "shutdown", 1*time.Second)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyConfig_EmptyCommand_SafeNoReload(t *testing.T) {
	dir := t.TempDir()
	tpl := filepath.Join(dir, "s.tpl")
	cfg := filepath.Join(dir, "suricata.yaml")

	_ = os.WriteFile(tpl, []byte("new"), 0o644)
	_ = os.WriteFile(cfg, []byte("old"), 0o644)

	rep, err := ApplyConfig(tpl, []string{cfg}, []string{"/tmp/no.sock"}, "/bin/true", "", 1*time.Second)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rep.ReloadStatus != ReloadOK {
		t.Fatalf("want ReloadOK, got %s", rep.ReloadStatus)
	}
	if len(rep.Warnings) == 0 || !strings.Contains(rep.Warnings[0], "reload_command пустой") {
		t.Fatalf("expected warning about empty command, got %+v", rep.Warnings)
	}
}

func TestWriteFileAtomic_WritesNewContent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")

	if err := os.WriteFile(p, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(p, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "new" {
		t.Fatalf("want new, got %q", string(b))
	}
}
