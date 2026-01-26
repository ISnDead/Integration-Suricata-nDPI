package integration

import (
	"bytes"
	"fmt"
	"strings"
)

func PatchSuricataConfigFromTemplate(template, current []byte) ([]byte, bool, error) {
	templateLines := splitLines(template)
	currentLines := splitLines(current)

	pluginsBlock, pluginsIndent, ok := findBlock(templateLines, "plugins")
	if !ok {
		return nil, false, fmt.Errorf("template does not contain plugins block")
	}
	ndpiLine, ok := findNDPIPluginLine(pluginsBlock)
	if !ok {
		return nil, false, fmt.Errorf("template does not contain ndpi.so plugin line")
	}

	unixBlock, _, unixBlockOk := findBlock(templateLines, "unix-command")

	changed := false
	currentLines, unixChanged, err := ensureBlockPresent(currentLines, unixBlock, "unix-command", unixBlockOk)
	if err != nil {
		return nil, false, err
	}
	if unixChanged {
		changed = true
	}

	currentLines, pluginsChanged, err := ensurePluginsNDPI(currentLines, pluginsIndent, ndpiLine)
	if err != nil {
		return nil, false, err
	}
	if pluginsChanged {
		changed = true
	}

	out := strings.Join(currentLines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}

	return []byte(out), changed, nil
}

func splitLines(data []byte) []string {
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	return strings.Split(string(data), "\n")
}

func normalizeKey(line string) string {
	trim := strings.TrimSpace(line)
	if strings.HasPrefix(trim, "#") {
		trim = strings.TrimSpace(strings.TrimPrefix(trim, "#"))
	}
	return trim
}

func indentWidth(line string) int {
	count := 0
	for _, r := range line {
		if r == ' ' || r == '\t' {
			count++
			continue
		}
		break
	}
	return count
}

func findBlock(lines []string, key string) ([]string, int, bool) {
	keyLine := key + ":"
	for i, line := range lines {
		if normalizeKey(line) != keyLine {
			continue
		}
		baseIndent := indentWidth(line)
		block := []string{line}
		for j := i + 1; j < len(lines); j++ {
			if lines[j] == "" {
				block = append(block, lines[j])
				continue
			}
			if indentWidth(lines[j]) <= baseIndent {
				break
			}
			block = append(block, lines[j])
		}
		return block, baseIndent, true
	}
	return nil, 0, false
}

func findNDPIPluginLine(block []string) (string, bool) {
	for _, line := range block {
		trim := normalizeKey(line)
		if strings.Contains(trim, "ndpi.so") {
			return strings.TrimSpace(line), true
		}
	}
	return "", false
}

func ensureBlockPresent(lines []string, block []string, key string, hasTemplate bool) ([]string, bool, error) {
	keyLine := key + ":"
	for i, line := range lines {
		if normalizeKey(line) != keyLine {
			continue
		}
		baseIndent := indentWidth(line)
		if isCommented(line) {
			lines[i] = uncommentLine(line)
			return uncommentBlock(lines, i+1, baseIndent)
		}
		return uncommentBlock(lines, i+1, baseIndent)
	}

	if !hasTemplate {
		return lines, false, nil
	}

	if len(lines) > 0 && lines[len(lines)-1] != "" {
		lines = append(lines, "")
	}
	lines = append(lines, block...)
	return lines, true, nil
}

func uncommentBlock(lines []string, start int, baseIndent int) ([]string, bool, error) {
	changed := false
	for i := start; i < len(lines); i++ {
		if lines[i] == "" {
			continue
		}
		if indentWidth(lines[i]) <= baseIndent {
			break
		}
		if isCommented(lines[i]) {
			lines[i] = uncommentLine(lines[i])
			changed = true
		}
	}
	return lines, changed, nil
}

func ensurePluginsNDPI(lines []string, pluginsIndent int, ndpiLine string) ([]string, bool, error) {
	keyLine := "plugins:"
	ndpiLine = strings.TrimSpace(ndpiLine)
	if !strings.HasPrefix(strings.TrimSpace(ndpiLine), "-") {
		ndpiLine = "- " + ndpiLine
	}
	for i, line := range lines {
		if normalizeKey(line) != keyLine {
			continue
		}

		if isCommented(line) {
			lines[i] = uncommentLine(line)
		}

		insertAt := i + 1
		found := false
		changed := false
		for j := i + 1; j < len(lines); j++ {
			if lines[j] == "" {
				insertAt = j + 1
				continue
			}
			if indentWidth(lines[j]) <= pluginsIndent {
				insertAt = j
				break
			}
			insertAt = j + 1
			if strings.Contains(normalizeKey(lines[j]), "ndpi.so") {
				found = true
				if isCommented(lines[j]) {
					lines[j] = uncommentLine(lines[j])
					changed = true
				}
				break
			}
		}

		if !found {
			indent := strings.Repeat(" ", pluginsIndent+2)
			lines = append(lines[:insertAt], append([]string{indent + ndpiLine}, lines[insertAt:]...)...)
			changed = true
		}
		return lines, changed, nil
	}

	if len(lines) > 0 && lines[len(lines)-1] != "" {
		lines = append(lines, "")
	}
	lines = append(lines, "plugins:", "  "+ndpiLine)
	return lines, true, nil
}
