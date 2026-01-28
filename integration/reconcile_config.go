package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/executil"
	"integration-suricata-ndpi/pkg/fsutil"
	"integration-suricata-ndpi/pkg/logger"
)

type ReconcileReport struct {
	TemplatePath     string `json:"template_path"`
	TargetConfigPath string `json:"target_config_path"`

	CurrentSHA256 string `json:"current_sha256"`
	PatchedSHA256 string `json:"patched_sha256"`

	CurrentBytes int `json:"current_bytes"`
	PatchedBytes int `json:"patched_bytes"`

	WouldChange      bool `json:"would_change"`
	Applied          bool `json:"applied"`
	Validated        bool `json:"validated"`
	RestartRequired  bool `json:"restart_required"`
	RestartPerformed bool `json:"restart_performed"`

	RestartCommand string `json:"restart_command,omitempty"`
	RestartOutput  string `json:"restart_output,omitempty"`
}

func ReconcileConfig(ctx context.Context, opts ApplyConfigOptions) (ReconcileReport, error) {
	fs := opts.FS
	if fs == nil {
		fs = fsutil.OSFS{}
	}
	runner := opts.CommandRunner
	if runner == nil {
		runner = executil.DefaultRunner{}
	}

	rep := ReconcileReport{
		TemplatePath: opts.TemplatePath,
	}

	tpl, err := fs.ReadFile(opts.TemplatePath)
	if err != nil {
		return rep, fmt.Errorf("read template %s: %w", opts.TemplatePath, err)
	}
	rendered, _, err := RenderTemplateStrict(tpl)
	if err != nil {
		return rep, fmt.Errorf("render template %s: %w", opts.TemplatePath, err)
	}

	target, err := FirstExistingPath(opts.ConfigCandidates)
	if err != nil {
		return rep, fmt.Errorf("suricata.yaml not found: %w", err)
	}
	rep.TargetConfigPath = target

	current, err := fs.ReadFile(target)
	if err != nil {
		return rep, fmt.Errorf("read current config %s: %w", target, err)
	}

	patched, changed, err := PatchSuricataConfigFromTemplate(rendered, current)
	if err != nil {
		return rep, fmt.Errorf("patch config %s: %w", target, err)
	}

	rep.CurrentBytes = len(current)
	rep.PatchedBytes = len(patched)
	rep.CurrentSHA256 = sha256Hex(current)
	rep.PatchedSHA256 = sha256Hex(patched)
	rep.WouldChange = changed
	rep.RestartRequired = changed

	if !changed {
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

	tmpPath := filepath.Clean(target + ".integration.tmp")
	if err := writeFileAtomic(tmpPath, patched, perm, fs); err != nil {
		return rep, fmt.Errorf("write tmp config %s: %w", tmpPath, err)
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
		_ = fs.Remove(tmpPath)
		return rep, fmt.Errorf("suricata -T failed; config NOT applied. err=%v output=%q", verr, vout)
	}
	rep.Validated = true

	if err := writeFileAtomic(target, patched, perm, fs); err != nil {
		_ = fs.Remove(tmpPath)
		return rep, fmt.Errorf("write config %s: %w", target, err)
	}
	_ = fs.Remove(tmpPath)
	rep.Applied = true

	systemctl := strings.TrimSpace(opts.SystemctlPath)
	if systemctl == "" {
		systemctl = "/usr/bin/systemctl"
	}
	unit := strings.TrimSpace(opts.SuricataService)
	if unit == "" {
		unit = "suricata"
	}

	rep.RestartCommand = fmt.Sprintf("%s restart %s", systemctl, unit)

	logger.Infow("Suricata YAML patched & validated (-T), restarting service",
		"path", target,
		"cmd", rep.RestartCommand,
	)

	rctx, rcancel := context.WithTimeout(ctx, 60*time.Second)
	defer rcancel()

	rout, rerr := runner.CombinedOutput(rctx, systemctl, "restart", unit)
	rep.RestartOutput = strings.TrimSpace(string(rout))
	if rerr != nil {
		return rep, fmt.Errorf("suricata restart failed: err=%v output=%q", rerr, rep.RestartOutput)
	}
	rep.RestartPerformed = true

	return rep, nil
}
