package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"integration-suricata-ndpi/pkg/executil"
	"integration-suricata-ndpi/pkg/fsutil"
	"integration-suricata-ndpi/pkg/logger"
)

const defaultReloadTimeout = 10 * time.Second

func ApplyConfig(opts ApplyConfigOptions) (ApplyConfigReport, error) {
	return ApplyConfigWithContext(context.Background(), opts)
}

func ApplyConfigWithContext(ctx context.Context, opts ApplyConfigOptions) (ApplyConfigReport, error) {
	templatePath := opts.TemplatePath
	configCandidates := opts.ConfigCandidates
	socketCandidates := opts.SocketCandidates
	suricatascPath := opts.SuricataSCPath
	reloadCommand := opts.ReloadCommand
	reloadTimeout := opts.ReloadTimeout
	commandRunner := opts.CommandRunner
	fs := opts.FS

	if fs == nil {
		fs = fsutil.OSFS{}
	}
	if commandRunner == nil {
		commandRunner = executil.DefaultRunner{}
	}

	report := ApplyConfigReport{
		ReloadCommand: reloadCommand,
		ReloadTimeout: reloadTimeout,
	}

	logger.Infow("Applying Suricata config (safe apply, no restart)",
		"template_path", templatePath,
		"config_candidates", configCandidates,
		"socket_candidates", socketCandidates,
		"suricatasc", suricatascPath,
		"reload_command", reloadCommand,
		"reload_timeout", reloadTimeout,
	)

	cmdNormalized := strings.TrimSpace(strings.ToLower(reloadCommand))
	if cmdNormalized == "shutdown" {
		return report, fmt.Errorf("reload_command=shutdown is forbidden")
	}

	tmplData, err := fs.ReadFile(templatePath)
	if err != nil {
		return report, fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	rendered, rr, err := RenderTemplateStrict(tmplData)
	if err != nil {
		return report, fmt.Errorf("failed to render template %s: %w", templatePath, err)
	}
	if len(rr.Vars) > 0 {
		logger.Infow("Template rendered with env vars", "vars", rr.Vars)
	}
	tmplData = rendered

	targetConfigPath, err := FirstExistingPath(configCandidates)
	if err != nil {
		return report, fmt.Errorf("suricata.yaml not found in candidates: %w", err)
	}
	report.TargetConfigPath = targetConfigPath

	currentConfig, err := fs.ReadFile(targetConfigPath)
	if err != nil {
		return report, fmt.Errorf("failed to read current config %s: %w", targetConfigPath, err)
	}

	updatedConfig, changed, err := PatchSuricataConfigFromTemplate(tmplData, currentConfig)
	if err != nil {
		return report, fmt.Errorf("failed to patch config %s: %w", targetConfigPath, err)
	}

	perm := os.FileMode(0o644)
	if st, statErr := fs.Stat(targetConfigPath); statErr == nil {
		perm = st.Mode().Perm()
	}

	if changed {
		if err := writeFileAtomic(targetConfigPath, updatedConfig, perm, fs); err != nil {
			return report, fmt.Errorf("failed to write config %s: %w", targetConfigPath, err)
		}
		logger.Infow("Suricata config updated", "path", targetConfigPath)
	} else {
		logger.Infow("Suricata config already contains required settings", "path", targetConfigPath)
	}

	if cmdNormalized == "" || cmdNormalized == "none" {
		report.ReloadStatus = ReloadOK
		report.Warnings = append(report.Warnings, "reload_command empty/none: reload skipped")
		logger.Warnw("reload_command empty/none: reload skipped", "reload_command", reloadCommand)
		return report, nil
	}

	if reloadTimeout <= 0 {
		reloadTimeout = defaultReloadTimeout
		report.ReloadTimeout = reloadTimeout
	}

	rctx, cancel := context.WithTimeout(ctx, reloadTimeout)
	defer cancel()

	out, err := commandRunner.CombinedOutput(rctx, suricatascPath, "-c", reloadCommand)
	report.ReloadOutput = strings.TrimSpace(string(out))

	if errors.Is(rctx.Err(), context.DeadlineExceeded) {
		report.ReloadStatus = ReloadTimeout
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("suricatasc timeout: command=%q timeout=%s", reloadCommand, reloadTimeout),
		)

		logger.Warnw("suricatasc timed out (restart forbidden). Checking Suricata socket availability",
			"command", reloadCommand,
			"timeout", reloadTimeout,
		)

		if err2 := EnsureSuricataRunning(socketCandidates); err2 != nil {
			return report, fmt.Errorf("suricatasc timeout and Suricata is not reachable via socket: %w", err2)
		}

		logger.Warnw("reload not confirmed due to timeout, but Suricata is reachable via socket; continuing",
			"command", reloadCommand,
		)
		return report, nil
	}

	if err == nil && rctx.Err() != nil {
		return report, rctx.Err()
	}

	if err != nil {
		report.ReloadStatus = ReloadFailed
		report.Warnings = append(report.Warnings,
			fmt.Sprintf("suricatasc error: command=%q err=%v output=%q", reloadCommand, err, report.ReloadOutput),
		)

		logger.Errorw("suricatasc failed (restart forbidden)",
			"command", reloadCommand,
			"output", report.ReloadOutput,
			"error", err,
		)

		if err2 := EnsureSuricataRunning(socketCandidates); err2 != nil {
			return report, fmt.Errorf("suricatasc failed and Suricata is not reachable via socket: %w", err2)
		}

		logger.Warnw("reload failed, but Suricata is reachable via socket; continuing",
			"command", reloadCommand,
		)
		return report, nil
	}

	report.ReloadStatus = ReloadOK
	logger.Infow("Suricata reload/reconfigure succeeded",
		"command", reloadCommand,
		"output", report.ReloadOutput,
	)

	return report, nil
}
