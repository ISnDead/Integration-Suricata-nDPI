package integration

import (
	"context"
	"fmt"

	"integration-suricata-ndpi/pkg/fsutil"
)

type PlanReport struct {
	TemplatePath     string `json:"template_path"`
	TargetConfigPath string `json:"target_config_path"`

	CurrentSHA256 string `json:"current_sha256"`
	PatchedSHA256 string `json:"patched_sha256"`

	CurrentBytes int `json:"current_bytes"`
	PatchedBytes int `json:"patched_bytes"`

	WouldChange     bool `json:"would_change"`
	RestartRequired bool `json:"restart_required"`
}

func PlanConfig(ctx context.Context, opts ApplyConfigOptions) (PlanReport, error) {
	_ = ctx

	fs := opts.FS
	if fs == nil {
		fs = fsutil.OSFS{}
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

	patched, changed, err := PatchSuricataConfigFromTemplate(rendered, current)
	if err != nil {
		return rep, fmt.Errorf("failed to patch config %s: %w", target, err)
	}

	rep.CurrentBytes = len(current)
	rep.PatchedBytes = len(patched)
	rep.CurrentSHA256 = sha256Hex(current)
	rep.PatchedSHA256 = sha256Hex(patched)

	rep.WouldChange = changed
	rep.RestartRequired = changed

	return rep, nil
}
