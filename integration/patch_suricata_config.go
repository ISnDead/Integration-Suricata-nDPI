package integration

import (
	"bytes"
	"fmt"
	"strings"
)

func PatchSuricataConfigFromTemplate(template, current []byte) ([]byte, bool, error) {
	tpl := splitLines(template)
	cur := splitLines(current)

	desiredNDPI, err := extractDesiredNDPIPluginLine(tpl)
	if err != nil {
		return nil, false, err
	}

	desiredUnix, err := extractDesiredUnixCommandKV(tpl)
	if err != nil {
		return nil, false, err
	}

	changed := false

	cur, ch, err := ensureUnixCommandBlock(cur, desiredUnix)
	if err != nil {
		return nil, false, err
	}
	if ch {
		changed = true
	}

	cur, ch, err = ensurePluginsNDPI(cur, desiredNDPI)
	if err != nil {
		return nil, false, err
	}
	if ch {
		changed = true
	}

	out := strings.Join(cur, "\n")
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

func findTopLevelKey(lines []string, key string) (idx int, baseIndent int, ok bool) {
	keyLine := key + ":"
	for i, ln := range lines {
		if normalizeKey(ln) == keyLine {
			return i, indentWidth(ln), true
		}
	}
	return -1, 0, false
}

func findBlockRange(lines []string, keyIdx int) (start int, end int, baseIndent int) {
	baseIndent = indentWidth(lines[keyIdx])
	start = keyIdx
	end = len(lines)
	for i := keyIdx + 1; i < len(lines); i++ {
		if lines[i] == "" {
			continue
		}
		if indentWidth(lines[i]) <= baseIndent {
			end = i
			break
		}
	}
	return start, end, baseIndent
}

func extractDesiredNDPIPluginLine(tpl []string) (string, error) {
	idx, _, ok := findTopLevelKey(tpl, "plugins")
	if !ok {
		return "", fmt.Errorf("template does not contain plugins block")
	}
	_, end, baseIndent := findBlockRange(tpl, idx)
	childIndentMin := baseIndent + 1

	for i := idx + 1; i < end; i++ {
		if tpl[i] == "" {
			continue
		}
		if indentWidth(tpl[i]) < childIndentMin {
			continue
		}
		trim := normalizeKey(tpl[i])
		if strings.Contains(trim, "ndpi.so") {
			s := strings.TrimSpace(tpl[i])
			s = strings.TrimPrefix(s, "#")
			s = strings.TrimSpace(s)
			if strings.HasPrefix(s, "-") {
				s = strings.TrimSpace(strings.TrimPrefix(s, "-"))
				return "- " + s, nil
			}
			return "- " + s, nil
		}
	}
	return "", fmt.Errorf("template does not contain ndpi.so plugin line inside plugins block")
}

func extractDesiredUnixCommandKV(tpl []string) (map[string]string, error) {
	idx, _, ok := findTopLevelKey(tpl, "unix-command")
	if !ok {
		return nil, fmt.Errorf("template does not contain unix-command block")
	}
	_, end, baseIndent := findBlockRange(tpl, idx)

	desired := map[string]string{}

	for i := idx + 1; i < end; i++ {
		ln := tpl[i]
		if ln == "" {
			continue
		}
		if indentWidth(ln) <= baseIndent {
			break
		}

		n := normalizeKey(ln)
		col := strings.Index(n, ":")
		if col <= 0 {
			continue
		}
		k := strings.TrimSpace(n[:col])
		v := strings.TrimSpace(n[col+1:])
		if k == "" {
			continue
		}
		desired[k] = v
	}

	if len(desired) == 0 {
		return nil, fmt.Errorf("template unix-command block is empty (no key: value lines)")
	}
	return desired, nil
}

func ensureUnixCommandBlock(lines []string, desired map[string]string) ([]string, bool, error) {
	idx, baseIndent, ok := findTopLevelKey(lines, "unix-command")
	changed := false

	if !ok {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "unix-command:")
		childIndent := "  "
		for _, k := range []string{"enabled", "filename", "mode"} {
			if v, ok := desired[k]; ok {
				if v == "" {
					lines = append(lines, childIndent+k+":")
				} else {
					lines = append(lines, childIndent+k+": "+v)
				}
			}
		}
		for k, v := range desired {
			if k == "enabled" || k == "filename" || k == "mode" {
				continue
			}
			if v == "" {
				lines = append(lines, childIndent+k+":")
			} else {
				lines = append(lines, childIndent+k+": "+v)
			}
		}
		return lines, true, nil
	}

	if isCommented(lines[idx]) {
		lines[idx] = uncommentLine(lines[idx])
		changed = true
	}

	start, end, _ := findBlockRange(lines, idx)
	_ = start

	childIndentWidth := baseIndent + 2
	childIndent := strings.Repeat(" ", childIndentWidth)

	for _, k := range orderedKeysPreferred(desired) {
		v := desired[k]
		found := false
		for i := idx + 1; i < end; i++ {
			if lines[i] == "" {
				continue
			}
			if indentWidth(lines[i]) <= baseIndent {
				break
			}

			n := normalizeKey(lines[i])
			if !strings.HasPrefix(n, k+":") {
				continue
			}

			found = true

			if isCommented(lines[i]) {
				lines[i] = uncommentLine(lines[i])
				changed = true
			}

			n2 := normalizeKey(lines[i])
			col := strings.Index(n2, ":")
			curV := ""
			if col >= 0 {
				curV = strings.TrimSpace(n2[col+1:])
			}

			if strings.TrimSpace(curV) != strings.TrimSpace(v) {
				if v == "" {
					lines[i] = childIndent + k + ":"
				} else {
					lines[i] = childIndent + k + ": " + v
				}
				changed = true
			}

			break
		}

		if !found {
			insertAt := end
			for insertAt > idx+1 && insertAt <= len(lines) && insertAt-1 < len(lines) {
				if insertAt-1 < len(lines) && lines[insertAt-1] == "" {
					insertAt--
					continue
				}
				break
			}

			newLine := ""
			if v == "" {
				newLine = childIndent + k + ":"
			} else {
				newLine = childIndent + k + ": " + v
			}

			lines = append(lines[:insertAt], append([]string{newLine}, lines[insertAt:]...)...)
			changed = true
			end++
		}
	}

	return lines, changed, nil
}

func orderedKeysPreferred(m map[string]string) []string {
	var out []string
	for _, k := range []string{"enabled", "filename", "mode"} {
		if _, ok := m[k]; ok {
			out = append(out, k)
		}
	}
	for k := range m {
		if k == "enabled" || k == "filename" || k == "mode" {
			continue
		}
		out = append(out, k)
	}
	return out
}
func ensurePluginsNDPI(lines []string, desiredItem string) ([]string, bool, error) {
	idx, baseIndent, ok := findTopLevelKey(lines, "plugins")
	changed := false

	desiredItem = strings.TrimSpace(desiredItem)
	if !strings.HasPrefix(desiredItem, "-") {
		desiredItem = "- " + strings.TrimSpace(strings.TrimPrefix(desiredItem, "-"))
	}

	if !ok {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "plugins:", "  "+desiredItem)
		return lines, true, nil
	}

	if isCommented(lines[idx]) {
		lines[idx] = uncommentLine(lines[idx])
		changed = true
	}

	_, end, _ := findBlockRange(lines, idx)

	itemIndent := strings.Repeat(" ", baseIndent+2)

	for i := idx + 1; i < end; i++ {
		if lines[i] == "" {
			continue
		}
		if indentWidth(lines[i]) <= baseIndent {
			break
		}
		n := normalizeKey(lines[i])
		if strings.Contains(n, "ndpi.so") {
			if isCommented(lines[i]) {
				lines[i] = uncommentLine(lines[i])
				changed = true
			}
			return lines, changed, nil
		}
	}

	insertAt := end
	for insertAt > idx+1 {
		if insertAt-1 < len(lines) && lines[insertAt-1] == "" {
			insertAt--
			continue
		}
		break
	}

	lines = append(lines[:insertAt], append([]string{itemIndent + desiredItem}, lines[insertAt:]...)...)
	changed = true
	return lines, changed, nil
}
