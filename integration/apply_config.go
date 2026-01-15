package integration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"integration-suricata-ndpi/pkg/logger"
)

func ApplyConfig(opts ApplyConfigOptions) (ApplyConfigReport, error) {
	templatePath := opts.TemplatePath
	configCandidates := opts.ConfigCandidates
	socketCandidates := opts.SocketCandidates
	suricatascPath := opts.SuricataSCPath
	reloadCommand := opts.ReloadCommand
	reloadTimeout := opts.ReloadTimeout

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
	if cmdNormalized == "" || cmdNormalized == "none" {
		report.ReloadStatus = ReloadOK
		report.Warnings = append(report.Warnings, "reload_command empty/none: config written, reload skipped")
		logger.Warnw("reload_command empty/none: reload skipped",
			"reload_command", reloadCommand,
		)
		return report, nil
	}

	tmplData, err := os.ReadFile(templatePath)
	if err != nil {
		return report, fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	targetConfigPath, err := FirstExistingPath(configCandidates)
	if err != nil {
		return report, fmt.Errorf("suricata.yaml not found in candidates: %w", err)
	}
	report.TargetConfigPath = targetConfigPath

	if err := writeFileAtomic(targetConfigPath, tmplData, 0o644); err != nil {
		return report, fmt.Errorf("failed to write config %s: %w", targetConfigPath, err)
	}
	logger.Infow("Suricata config updated", "path", targetConfigPath)

	var ctx context.Context
	var cancel func()
	if reloadTimeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), reloadTimeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, suricatascPath, "-c", reloadCommand)
	out, err := cmd.CombinedOutput()
	report.ReloadOutput = strings.TrimSpace(string(out))

	// suricatasc timeout
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
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

	// suricatasc error
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
