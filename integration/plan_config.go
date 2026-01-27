package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/executil"
	"integration-suricata-ndpi/pkg/fsutil"
)

type PlanReport struct {
	TemplatePath     string `json:"template_path"`
	TargetConfigPath string `json:"target_config_path"`

	CurrentSHA256  string `json:"current_sha256"`
	RenderedSHA256 string `json:"rendered_sha256"`

	CurrentBytes  int `json:"current_bytes"`
	RenderedBytes int `json:"rendered_bytes"`

	WouldWrite bool     `json:"would_write"`
	Applied    bool     `json:"applied"`
	Validated  bool     `json:"validated"`
	Notes      []string `json:"notes,omitempty"`
}

func PlanConfig(ctx context.Context, opts ApplyConfigOptions) (PlanReport, error) {
	fs := opts.FS
	if fs == nil {
		fs = fsutil.OSFS{}
	}
	runner := opts.CommandRunner
	if runner == nil {
		runner = executil.DefaultRunner{}
	}

	rep := PlanReport{
		TemplatePath: opts.TemplatePath,
	}

	tpl, err := fs.ReadFile(opts.TemplatePath)
	if err != nil {
		return rep, fmt.Errorf("failed to read template %s: %w", opts.TemplatePath, err)
	}
	tplRendered, _, err := RenderTemplateStrict(tpl)
	if err != nil {
		return rep, fmt.Errorf("failed to render template %s: %w", opts.TemplatePath, err)
	}

	target, err := FirstExistingPath(opts.ConfigCandidates)
	if err != nil {
		return rep, fmt.Errorf("suricata.yaml not found in candidates: %w", err)
	}
	rep.TargetConfigPath = target

	current, err := fs.ReadFile(target)
	if err != nil {
		return rep, fmt.Errorf("failed to read current config %s: %w", target, err)
	}

	updated, changed, err := PatchSuricataConfigFromTemplate(tplRendered, current)
	if err != nil {
		return rep, fmt.Errorf("failed to patch config %s: %w", target, err)
	}

	rep.CurrentBytes = len(current)
	rep.RenderedBytes = len(updated)
	rep.CurrentSHA256 = sha256Hex(current)
	rep.RenderedSHA256 = sha256Hex(updated)
	rep.WouldWrite = changed

	if !changed {
		rep.Notes = append(rep.Notes, "config already matches template blocks (plugins/unix-command); nothing to do")
		return rep, nil
	}

	suricataBin := strings.TrimSpace(opts.SuricataBinPath)
	if suricataBin == "" {
		suricataBin = "suricata"
	}

	perm := os.FileMode(0o644)
	if st, statErr := fs.Stat(target); statErr == nil {
		perm = st.Mode().Perm()
	}

	tmpPath := filepath.Clean(target + ".integration.plan.tmp")
	if err := writeFileAtomic(tmpPath, updated, perm, fs); err != nil {
		return rep, fmt.Errorf("failed to write temp config %s: %w", tmpPath, err)
	}

	vctx := ctx
	if vctx == nil {
		vctx = context.Background()
	}
	vctx, cancel := context.WithTimeout(vctx, 30*time.Second)
	defer cancel()

	out, verr := runner.CombinedOutput(vctx, suricataBin, "-T", "-c", tmpPath)
	vout := strings.TrimSpace(string(out))
	if verr != nil {
		_ = tryRemove(fs, tmpPath)
		rep.Validated = false
		return rep, fmt.Errorf("suricata -T failed; config NOT applied. err=%v output=%q", verr, vout)
	}
	rep.Validated = true

	if err := writeFileAtomic(target, updated, perm, fs); err != nil {
		_ = tryRemove(fs, tmpPath)
		return rep, fmt.Errorf("failed to write config %s: %w", target, err)
	}

	_ = tryRemove(fs, tmpPath)
	rep.Applied = true
	rep.Notes = append(rep.Notes, "patched plugins/unix-command blocks and validated with suricata -T")

	return rep, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func tryRemove(fs fsutil.FS, path string) error {
	if fs == nil {
		fs = fsutil.OSFS{}
	}
	_ = fs.Remove(path)
	return nil
}
