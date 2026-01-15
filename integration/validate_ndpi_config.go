package integration

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"integration-suricata-ndpi/pkg/logger"
)

func ValidateNDPIConfig(opts NDPIValidateOptions) error {
	ndpiPluginPath := opts.NDPIPluginPath
	ndpiRulesDir := opts.NDPIRulesDir
	suricataTemplatePath := opts.SuricataTemplatePath
	suricatascPath := opts.SuricataSCPath
	reloadCommand := opts.ReloadCommand
	reloadTimeout := opts.ReloadTimeout
	expectedNdpiRulesPattern := opts.ExpectedRulesPattern

	logger.Infow("Validating nDPI configuration",
		"ndpi_plugin_path", ndpiPluginPath,
		"ndpi_rules_dir", ndpiRulesDir,
		"suricata_template", suricataTemplatePath,
		"suricatasc_path", suricatascPath,
		"reload_command", reloadCommand,
		"reload_timeout", reloadTimeout,
		"expected_ndpi_rules_pattern", expectedNdpiRulesPattern,
	)

	if err := mustBeFile(ndpiPluginPath, "nDPI plugin (ndpi.so)"); err != nil {
		return err
	}

	if err := mustBeDir(ndpiRulesDir, "nDPI rules directory"); err != nil {
		return err
	}

	ruleFiles, _ := filepath.Glob(filepath.Join(ndpiRulesDir, "*.rules"))
	if len(ruleFiles) == 0 {
		logger.Warnw("No *.rules files found in nDPI rules directory (not fatal)",
			"path", ndpiRulesDir,
		)
	}

	tpl, err := os.ReadFile(suricataTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read Suricata template (%s): %w", suricataTemplatePath, err)
	}

	tplLower := bytes.ToLower(tpl)

	if !bytes.Contains(tplLower, []byte("plugins")) {
		return fmt.Errorf("suricata template has no 'plugins' block (nDPI will not be loaded)")
	}

	// check ndpi.so mention
	base := strings.ToLower(filepath.Base(ndpiPluginPath))
	if !bytes.Contains(tplLower, []byte(base)) && !bytes.Contains(tplLower, []byte("ndpi.so")) {
		return fmt.Errorf("suricata template does not mention ndpi.so in 'plugins' block (plugin won't be loaded)")
	}

	if expectedNdpiRulesPattern != "" && !bytes.Contains(tpl, []byte(expectedNdpiRulesPattern)) {
		logger.Warnw("Expected nDPI rules pattern not found in Suricata template (not fatal, but may affect enable/disable via rules)",
			"pattern", expectedNdpiRulesPattern,
		)
	}

	if err := mustBeFile(suricatascPath, "suricatasc"); err != nil {
		return err
	}

	cmdNormalized := strings.TrimSpace(strings.ToLower(reloadCommand))
	if cmdNormalized == "shutdown" {
		return fmt.Errorf("reload_command=shutdown is forbidden")
	}
	if cmdNormalized == "" || cmdNormalized == "none" {
		logger.Warnw("Reload command is empty/none",
			"reload_command", reloadCommand,
		)
	}

	if reloadTimeout <= 0 {
		return fmt.Errorf("reload_timeout must be > 0")
	}

	logger.Infow("nDPI configuration is valid")
	return nil
}

func mustBeFile(path string, what string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found: %s", what, path)
		}
		return fmt.Errorf("cannot access %s (%s): %w", what, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s must be a file, got directory: %s", what, path)
	}
	return nil
}

func mustBeDir(path string, what string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s not found: %s", what, path)
		}
		return fmt.Errorf("cannot access %s (%s): %w", what, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s must be a directory: %s", what, path)
	}
	return nil
}
