package integration

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"integration-suricata-ndpi/pkg/fsutil"
)

func NDPIStatus(suricataCfgPath, ndpiPluginPath string) (bool, string, error) {
	return NDPIStatusWithFS(suricataCfgPath, ndpiPluginPath, nil)
}

func NDPIStatusWithFS(suricataCfgPath, ndpiPluginPath string, fs fsutil.FS) (bool, string, error) {
	lines, err := readLines(suricataCfgPath, fs)
	if err != nil {
		return false, "", err
	}

	for _, ln := range lines {
		if matchNDPIPluginLine(ln, ndpiPluginPath) {
			enabled := !isCommented(ln)
			return enabled, strings.TrimRight(ln, "\r\n"), nil
		}
	}
	return false, "", fmt.Errorf("ndpi plugin line not found in %s", suricataCfgPath)
}

func SetNDPIEnabled(suricataCfgPath, ndpiPluginPath string, enable bool) (bool, bool, error) {
	return SetNDPIEnabledWithFS(suricataCfgPath, ndpiPluginPath, enable, nil)
}

func SetNDPIEnabledWithFS(suricataCfgPath, ndpiPluginPath string, enable bool, fs fsutil.FS) (bool, bool, error) {
	lines, err := readLines(suricataCfgPath, fs)
	if err != nil {
		return false, false, err
	}

	found := false
	changed := false
	enabledAfter := false

	for i, ln := range lines {
		if !matchNDPIPluginLine(ln, ndpiPluginPath) {
			continue
		}

		found = true
		curEnabled := !isCommented(ln)

		if enable {
			enabledAfter = true
			if curEnabled {
				break
			}
			lines[i] = uncommentLine(ln)
			changed = true
			break
		}

		enabledAfter = false
		if !curEnabled {
			break
		}
		lines[i] = commentLine(ln)
		changed = true
		break
	}

	if !found {
		return false, false, fmt.Errorf("ndpi plugin line not found in %s", suricataCfgPath)
	}

	if !changed {
		return false, enabledAfter, nil
	}

	perm := os.FileMode(0o644)
	if fs == nil {
		fs = fsutil.OSFS{}
	}

	if st, statErr := fs.Stat(suricataCfgPath); statErr == nil {
		perm = st.Mode().Perm()
	}

	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	if err := writeFileAtomic(suricataCfgPath, []byte(out), perm, fs); err != nil {
		return false, false, fmt.Errorf("failed to write suricata config: %w", err)
	}

	return true, enabledAfter, nil
}

func readLines(path string, fs fsutil.FS) ([]string, error) {
	if fs == nil {
		fs = fsutil.OSFS{}
	}

	b, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))

	var lines []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return lines, nil
}

func matchNDPIPluginLine(line, ndpiPluginPath string) bool {
	base := filepath.Base(ndpiPluginPath)

	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "#")
	s = strings.TrimSpace(s)

	if !strings.Contains(s, "-") {
		return false
	}

	return strings.Contains(s, ndpiPluginPath) || (base != "" && strings.Contains(s, base))
}

func isCommented(line string) bool {
	trim := strings.TrimLeft(line, " \t")
	return strings.HasPrefix(trim, "#")
}

func commentLine(line string) string {
	if isCommented(line) {
		return line
	}
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	rest := strings.TrimLeft(line, " \t")
	return indent + "# " + rest
}

func uncommentLine(line string) string {
	if !isCommented(line) {
		return line
	}

	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	rest := strings.TrimLeft(line, " \t")

	rest = strings.TrimPrefix(rest, "#")
	rest = strings.TrimLeft(rest, " \t")

	return indent + rest
}
