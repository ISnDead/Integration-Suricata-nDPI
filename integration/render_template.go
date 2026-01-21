package integration

import (
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
		return nil, RenderReport{Vars: vars},
			fmt.Errorf("template requires env vars not set: %s", strings.Join(missing, ", "))
	}

	out := envVarRe.ReplaceAllStringFunc(string(in), func(s string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(s, "${"), "}")
		return env[name]
	})

	return []byte(out), RenderReport{Vars: vars}, nil
}
