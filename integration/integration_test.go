package integration

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"integration-suricata-ndpi/pkg/fsutil"
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

func writeFile(t *testing.T, path, body string, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), perm); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func setupTemplateAndConfig(t *testing.T, dir string) (string, string) {
	t.Helper()

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	cfg := filepath.Join(dir, "suricata.yaml")

	templateBody := `
plugins:
  - /usr/local/lib/suricata/ndpi.so

unix-command:
  enabled: yes
  filename: /run/suricata/suricata-command.socket
  mode: 0660
`
	writeFile(t, tpl, templateBody, 0o644)
	writeFile(t, cfg, "oldcfg\n", 0o644)

	return tpl, cfg
}

func startUnixSocketListener(t *testing.T, sock string) net.Listener {
	t.Helper()

	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			c, e := l.Accept()
			if e != nil {
				return
			}
			_ = c.Close()
		}
	}()

	return l
}

func TestApplyConfig_ReloadOK(t *testing.T) {
	dir := t.TempDir()

	tpl, cfg := setupTemplateAndConfig(t, dir)

	sock := filepath.Join(dir, "suricata-command.socket")
	l := startUnixSocketListener(t, sock)
	t.Cleanup(func() { _ = l.Close() })

	suricatasc := writeExecutable(t, dir, "suricatasc", "#!/bin/sh\necho OK\nexit 0\n")

	rep, err := ApplyConfig(ApplyConfigOptions{
		TemplatePath:     tpl,
		ConfigCandidates: []string{cfg},
		SocketCandidates: []string{sock},
		SuricataSCPath:   suricatasc,
		ReloadCommand:    "reconfigure",
		ReloadTimeout:    200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rep.ReloadStatus != ReloadOK {
		t.Fatalf("want ReloadOK, got %s", rep.ReloadStatus)
	}
}

func TestApplyConfig_ReloadFailed_ButSocketAlive(t *testing.T) {
	dir := t.TempDir()

	tpl, cfg := setupTemplateAndConfig(t, dir)

	sock := filepath.Join(dir, "suricata-command.socket")
	l := startUnixSocketListener(t, sock)
	t.Cleanup(func() { _ = l.Close() })

	suricatasc := writeExecutable(t, dir, "suricatasc", "#!/bin/sh\necho FAIL\nexit 1\n")

	rep, err := ApplyConfig(ApplyConfigOptions{
		TemplatePath:     tpl,
		ConfigCandidates: []string{cfg},
		SocketCandidates: []string{sock},
		SuricataSCPath:   suricatasc,
		ReloadCommand:    "reconfigure",
		ReloadTimeout:    200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rep.ReloadStatus != ReloadFailed {
		t.Fatalf("want ReloadFailed, got %s", rep.ReloadStatus)
	}
}

func TestApplyConfig_ReloadTimeout_ButSocketAlive(t *testing.T) {
	dir := t.TempDir()

	tpl, cfg := setupTemplateAndConfig(t, dir)

	sock := filepath.Join(dir, "suricata-command.socket")
	l := startUnixSocketListener(t, sock)
	t.Cleanup(func() { _ = l.Close() })

	suricatasc := writeExecutable(t, dir, "suricatasc", "#!/bin/sh\nsleep 0.2\necho LATE\nexit 0\n")

	rep, err := ApplyConfig(ApplyConfigOptions{
		TemplatePath:     tpl,
		ConfigCandidates: []string{cfg},
		SocketCandidates: []string{sock},
		SuricataSCPath:   suricatasc,
		ReloadCommand:    "reconfigure",
		ReloadTimeout:    50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rep.ReloadStatus != ReloadTimeout {
		t.Fatalf("want ReloadTimeout, got %s", rep.ReloadStatus)
	}
}

func TestApplyConfig_ReloadFailed_AndSocketDown_ReturnsError(t *testing.T) {
	dir := t.TempDir()

	tpl, cfg := setupTemplateAndConfig(t, dir)

	sock := filepath.Join(dir, "suricata-command.socket")
	suricatasc := writeExecutable(t, dir, "suricatasc", "#!/bin/sh\necho FAIL\nexit 1\n")

	_, err := ApplyConfig(ApplyConfigOptions{
		TemplatePath:     tpl,
		ConfigCandidates: []string{cfg},
		SocketCandidates: []string{sock},
		SuricataSCPath:   suricatasc,
		ReloadCommand:    "reconfigure",
		ReloadTimeout:    200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyConfig_RejectShutdown(t *testing.T) {
	_, err := ApplyConfig(ApplyConfigOptions{
		TemplatePath:     "x",
		ConfigCandidates: []string{"y"},
		SocketCandidates: []string{"/tmp/no.sock"},
		SuricataSCPath:   "/bin/true",
		ReloadCommand:    "shutdown",
		ReloadTimeout:    1 * time.Second,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyConfig_EmptyCommand_SafeNoReload(t *testing.T) {
	dir := t.TempDir()
	tpl := filepath.Join(dir, "s.tpl")
	cfg := filepath.Join(dir, "suricata.yaml")
	writeFile(t, tpl, "new", 0o644)
	writeFile(t, cfg, "old", 0o644)

	rep, err := ApplyConfig(ApplyConfigOptions{
		TemplatePath:     tpl,
		ConfigCandidates: []string{cfg},
		SocketCandidates: []string{"/tmp/no.sock"},
		SuricataSCPath:   "/bin/true",
		ReloadCommand:    "",
		ReloadTimeout:    1 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if rep.ReloadStatus != ReloadOK {
		t.Fatalf("want ReloadOK, got %s", rep.ReloadStatus)
	}
	if len(rep.Warnings) == 0 || !strings.Contains(rep.Warnings[0], "reload_command empty/none") {
		t.Fatalf("expected warning about empty command, got %+v", rep.Warnings)
	}
}

func TestWriteFileAtomic_DirMissing_Error(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "no_such_dir", "x.txt")

	err := writeFileAtomic(target, []byte("data"), 0o644, fsutil.OSFS{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteFileAtomic_PermApplied(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")

	if err := writeFileAtomic(p, []byte("new"), 0o600, fsutil.OSFS{}); err != nil {
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

func TestWriteFileAtomic_OverwriteExisting_OK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")

	writeFile(t, p, "old", 0o644)
	if err := writeFileAtomic(p, []byte("new"), 0o600, fsutil.OSFS{}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "new" {
		t.Fatalf("want new, got %q", string(b))
	}
}

func TestWriteFileAtomic_RenameToDir_Error(t *testing.T) {
	dir := t.TempDir()

	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	err := writeFileAtomic(target, []byte("data"), 0o644, fsutil.OSFS{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMustBeFile_OK(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	writeFile(t, p, "x", 0o644)
	if err := mustBeFile(p, "file", fsutil.OSFS{}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestMustBeFile_WhenDir_Error(t *testing.T) {
	dir := t.TempDir()
	if err := mustBeFile(dir, "file", fsutil.OSFS{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestMustBeFile_Missing_Error(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nope.txt")
	if err := mustBeFile(p, "file", fsutil.OSFS{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestMustBeDir_OK(t *testing.T) {
	dir := t.TempDir()
	if err := mustBeDir(dir, "dir", fsutil.OSFS{}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestMustBeDir_WhenFile_Error(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	writeFile(t, p, "x", 0o644)
	if err := mustBeDir(p, "dir", fsutil.OSFS{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestMustBeDir_Missing_Error(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nope-dir")
	if err := mustBeDir(p, "dir", fsutil.OSFS{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNDPIConfig_OK(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	writeFile(t, ndpiSo, "fake", 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(rulesDir, "test.rules"), "alert any any -> any any (msg:\"x\";)", 0o644)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	tplBody := `
plugins:
  - ` + ndpiSo + `
rule-files:
  - var/lib/suricata/rules/ndpi/*.rules
`
	writeFile(t, tpl, tplBody, 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	writeFile(t, suricatasc, "#!/bin/sh\nexit 0\n", 0o755)

	err := ValidateNDPIConfig(NDPIValidateOptions{
		NDPIPluginPath:       ndpiSo,
		NDPIRulesDir:         rulesDir,
		SuricataTemplatePath: tpl,
		SuricataSCPath:       suricatasc,
		ReloadCommand:        "reload-rules",
		ReloadTimeout:        2 * time.Second,
		ExpectedRulesPattern: "var/lib/suricata/rules/ndpi/*.rules",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateNDPIConfig_NDPISOMissing_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "missing_ndpi.so")
	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	writeFile(t, tpl, "plugins:\n  - "+ndpiSo+"\n", 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	writeFile(t, suricatasc, "#!/bin/sh\nexit 0\n", 0o755)

	err := ValidateNDPIConfig(NDPIValidateOptions{
		NDPIPluginPath:       ndpiSo,
		NDPIRulesDir:         rulesDir,
		SuricataTemplatePath: tpl,
		SuricataSCPath:       suricatasc,
		ReloadCommand:        "reload-rules",
		ReloadTimeout:        2 * time.Second,
		ExpectedRulesPattern: "var/lib/suricata/rules/ndpi/*.rules",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNDPIConfig_RulesDirMissing_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	writeFile(t, ndpiSo, "fake", 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	writeFile(t, tpl, "plugins:\n  - "+ndpiSo+"\n", 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	writeFile(t, suricatasc, "#!/bin/sh\nexit 0\n", 0o755)

	err := ValidateNDPIConfig(NDPIValidateOptions{
		NDPIPluginPath:       ndpiSo,
		NDPIRulesDir:         rulesDir,
		SuricataTemplatePath: tpl,
		SuricataSCPath:       suricatasc,
		ReloadCommand:        "reload-rules",
		ReloadTimeout:        2 * time.Second,
		ExpectedRulesPattern: "var/lib/suricata/rules/ndpi/*.rules",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNDPIConfig_TemplateMissing_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	writeFile(t, ndpiSo, "fake", 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "missing.tpl")

	suricatasc := filepath.Join(dir, "suricatasc")
	writeFile(t, suricatasc, "#!/bin/sh\nexit 0\n", 0o755)

	err := ValidateNDPIConfig(NDPIValidateOptions{
		NDPIPluginPath:       ndpiSo,
		NDPIRulesDir:         rulesDir,
		SuricataTemplatePath: tpl,
		SuricataSCPath:       suricatasc,
		ReloadCommand:        "reload-rules",
		ReloadTimeout:        2 * time.Second,
		ExpectedRulesPattern: "var/lib/suricata/rules/ndpi/*.rules",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNDPIConfig_SuricatascMissing_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	writeFile(t, ndpiSo, "fake", 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	writeFile(t, tpl, "plugins:\n  - "+ndpiSo+"\n", 0o644)

	suricatasc := filepath.Join(dir, "missing_suricatasc")

	err := ValidateNDPIConfig(NDPIValidateOptions{
		NDPIPluginPath:       ndpiSo,
		NDPIRulesDir:         rulesDir,
		SuricataTemplatePath: tpl,
		SuricataSCPath:       suricatasc,
		ReloadCommand:        "reload-rules",
		ReloadTimeout:        2 * time.Second,
		ExpectedRulesPattern: "var/lib/suricata/rules/ndpi/*.rules",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateNDPIConfig_ReloadCommandEmpty_OK(t *testing.T) {
	dir := t.TempDir()

	ndpiSo := filepath.Join(dir, "ndpi.so")
	writeFile(t, ndpiSo, "fake", 0o644)

	rulesDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(rulesDir, 0o755)

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	writeFile(t, tpl, "plugins:\n  - "+ndpiSo+"\n", 0o644)

	suricatasc := filepath.Join(dir, "suricatasc")
	writeFile(t, suricatasc, "#!/bin/sh\nexit 0\n", 0o755)

	err := ValidateNDPIConfig(NDPIValidateOptions{
		NDPIPluginPath:       ndpiSo,
		NDPIRulesDir:         rulesDir,
		SuricataTemplatePath: tpl,
		SuricataSCPath:       suricatasc,
		ReloadCommand:        "",
		ReloadTimeout:        2 * time.Second,
		ExpectedRulesPattern: "var/lib/suricata/rules/ndpi/*.rules",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
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
	writeFile(t, p, "x", 0o644)

	if err := EnsureSuricataRunning([]string{p}); err == nil {
		t.Fatal("expected error")
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

	t.Cleanup(func() {
		if client != nil && client.Conn != nil {
			_ = client.Conn.Close()
		}
	})
}

func TestValidateLocalResources_OK(t *testing.T) {
	dir := t.TempDir()

	ndpiDir := filepath.Join(dir, "rules", "ndpi")
	if err := os.MkdirAll(ndpiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tpl := filepath.Join(dir, "suricata.yaml.tpl")
	writeFile(t, tpl, "plugins:\n  - /usr/local/lib/suricata/ndpi.so\n", 0o644)

	if err := ValidateLocalResources(ndpiDir, tpl, fsutil.OSFS{}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestValidateLocalResources_Missing(t *testing.T) {
	dir := t.TempDir()
	tpl := filepath.Join(dir, "x.tpl")
	writeFile(t, tpl, "x", 0o644)

	if err := ValidateLocalResources(filepath.Join(dir, "nope"), tpl, fsutil.OSFS{}); err == nil {
		t.Fatal("expected error for missing ndpi dir")
	}
	if err := ValidateLocalResources(dir, filepath.Join(dir, "nope.tpl"), fsutil.OSFS{}); err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestValidateLocalResources_TemplateIsDir_Error(t *testing.T) {
	dir := t.TempDir()

	ndpiDir := filepath.Join(dir, "rules", "ndpi")
	_ = os.MkdirAll(ndpiDir, 0o755)

	tplDir := filepath.Join(dir, "tpldir")
	_ = os.MkdirAll(tplDir, 0o755)

	if err := ValidateLocalResources(ndpiDir, tplDir, fsutil.OSFS{}); err == nil {
		t.Fatal("expected error")
	}
}
