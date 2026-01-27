package integration

import (
	"context"
	"errors"
	"fmt"
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
	suricatascPath := opts.SuricataSCPath
	reloadCommand := opts.ReloadCommand
	reloadTimeout := opts.ReloadTimeout

	commandRunner := opts.CommandRunner
	if commandRunner == nil {
		commandRunner = executil.DefaultRunner{}
	}

	_ = opts.FS
	_ = fsutil.OSFS{}

	report := ApplyConfigReport{
		ReloadCommand: reloadCommand,
		ReloadTimeout: reloadTimeout,
	}

	logger.Infow("Applying rules/reload via suricatasc (no YAML changes)",
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
		report.Warnings = append(report.Warnings, "reload_command empty/none: reload skipped")
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
		return report, nil
	}

	report.ReloadStatus = ReloadOK
	return report, nil
}
