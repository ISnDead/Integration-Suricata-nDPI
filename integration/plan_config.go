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
	"integration-suricata-ndpi/pkg/logger"
)

type PlanReport struct {
	TemplatePath     string `json:"template_path"`
	TargetConfigPath string `json:"target_config_path"`

	CurrentSHA256 string `json:"current_sha256"`
	PatchedSHA256 string `json:"patched_sha256"`

	CurrentBytes int `json:"current_bytes"`
	PatchedBytes int `json:"patched_bytes"`

	Changed      bool `json:"changed"`
	Validated    bool `json:"validated"`
	Restarted    bool `json:"restarted"`
	WouldRestart bool `json:"would_restart"`

	Notes []string `json:"notes,omitempty"`
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

	rendered, _, err := RenderTemplateStrict(tpl)
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

	rep.CurrentBytes = len(current)
	rep.CurrentSHA256 = sha256Hex(current)

	patched, changed, err := PatchSuricataConfigFromTemplate(rendered, current)
	if err != nil {
		return rep, fmt.Errorf("failed to patch config %s: %w", target, err)
	}

	rep.PatchedBytes = len(patched)
	rep.PatchedSHA256 = sha256Hex(patched)
	rep.Changed = changed
	rep.WouldRestart = changed

	if !changed {
		rep.Notes = append(rep.Notes, "plugins/unix-command already match template; no write, no restart")
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

	tmpPath := filepath.Clean(target + ".plan.tmp")

	if err := writeFileAtomic(tmpPath, patched, perm, fs); err != nil {
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

	if err := writeFileAtomic(target, patched, perm, fs); err != nil {
		_ = tryRemove(fs, tmpPath)
		return rep, fmt.Errorf("failed to write config %s: %w", target, err)
	}
	_ = tryRemove(fs, tmpPath)

	logger.Infow("plan: suricata.yaml patched & validated; restarting suricata", "path", target)

	systemctl := strings.TrimSpace(opts.SystemctlPath)
	if systemctl == "" {
		systemctl = "systemctl"
	}
	unit := strings.TrimSpace(opts.SuricataService)
	if unit == "" {
		unit = "suricata"
	}

	rctx := ctx
	if rctx == nil {
		rctx = context.Background()
	}
	rctx, rcancel := context.WithTimeout(rctx, 60*time.Second)
	defer rcancel()

	rout, rerr := runner.CombinedOutput(rctx, systemctl, "restart", unit)
	if rerr != nil {
		rep.Restarted = false
		rep.Notes = append(rep.Notes, fmt.Sprintf("restart failed: %v output=%q", rerr, strings.TrimSpace(string(rout))))
		return rep, fmt.Errorf("failed to restart suricata (%s restart %s): %v output=%q",
			systemctl, unit, rerr, strings.TrimSpace(string(rout)))
	}

	rep.Restarted = true
	rep.Notes = append(rep.Notes, "suricata restarted because config changed")

	return rep, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func tryRemove(fs fsutil.FS, path string) error {
	if fs == nil {
		return os.Remove(path)
	}
	if r, ok := any(fs).(interface{ Remove(string) error }); ok {
		return r.Remove(path)
	}
	_ = os.Remove(path)
	return nil
}
