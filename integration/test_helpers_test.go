package integration

import (
	"net"
	"os"
	"path/filepath"
	"testing"
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
