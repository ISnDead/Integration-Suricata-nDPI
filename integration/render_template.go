package integration

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var envVarRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

type RenderReport struct {
	Vars []string
}

func RenderTemplateStrict(in []byte) ([]byte, RenderReport, error) {
	matches := envVarRe.FindAllSubmatch(in, -1)

	set := map[string]struct{}{}
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		set[string(m[1])] = struct{}{}
	}

	var vars []string
	for v := range set {
		vars = append(vars, v)
	}
	sort.Strings(vars)

	if len(vars) == 0 {
		return in, RenderReport{Vars: nil}, nil
	}

	env := make(map[string]string, len(vars))
	var missing []string
	for _, v := range vars {
		val, ok := os.LookupEnv(v)
		if !ok {
			missing = append(missing, v)
			continue
		}
		env[v] = val
	}
	if len(missing) > 0 {
		autoVars, err := autoDetectEnv(missing)
		if err != nil {
			return nil, RenderReport{Vars: vars}, err
		}
		for k, v := range autoVars {
			env[k] = v
		}

		var stillMissing []string
		for _, v := range missing {
			if _, ok := env[v]; !ok {
				stillMissing = append(stillMissing, v)
			}
		}
		missing = stillMissing
	}
	if len(missing) > 0 {
		return nil, RenderReport{Vars: vars},
			fmt.Errorf("template requires env vars not set: %s", strings.Join(missing, ", "))
	}

	out := envVarRe.ReplaceAllStringFunc(string(in), func(s string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(s, "${"), "}")
		return env[name]
	})

	return []byte(out), RenderReport{Vars: vars}, nil
}

func autoDetectEnv(missing []string) (map[string]string, error) {
	out := make(map[string]string)
	needsIface := false
	for _, v := range missing {
		if v == "SURICATA_IFACE" {
			needsIface = true
			break
		}
	}
	if !needsIface {
		return out, nil
	}

	iface, err := detectDefaultIface()
	if err != nil {
		return nil, err
	}
	if iface != "" {
		out["SURICATA_IFACE"] = iface
	}
	return out, nil
}

func detectDefaultIface() (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/net/route: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	first := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if first {
			first = false
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == "00000000" {
			return fields[0], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to scan /proc/net/route: %w", err)
	}
	return "", fmt.Errorf("default interface not found in /proc/net/route")
}
