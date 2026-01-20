package integration

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

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
	Notes      []string `json:"notes,omitempty"`
}

func PlanConfig(opts ApplyConfigOptions) (PlanReport, error) {
	fs := opts.FS
	if fs == nil {
		fs = fsutil.OSFS{}
	}

	rep := PlanReport{
		TemplatePath: opts.TemplatePath,
	}

	rendered, err := fs.ReadFile(opts.TemplatePath)
	if err != nil {
		return rep, fmt.Errorf("failed to read template %s: %w", opts.TemplatePath, err)
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
	rep.RenderedBytes = len(rendered)

	rep.CurrentSHA256 = sha256Hex(current)
	rep.RenderedSHA256 = sha256Hex(rendered)
	rep.WouldWrite = rep.CurrentSHA256 != rep.RenderedSHA256

	return rep, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
