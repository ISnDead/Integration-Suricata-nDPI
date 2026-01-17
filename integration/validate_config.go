package integration

import (
	"fmt"
	"os"
	"path/filepath"

	"integration-suricata-ndpi/pkg/fsutil"
	"integration-suricata-ndpi/pkg/logger"
)

func ValidateLocalResources(ndpiRulesDir string, templatePath string, fs fsutil.FS) error {
	if fs == nil {
		fs = fsutil.OSFS{}
	}

	logger.Infow("Validating local resources",
		"ndpi_rules_dir", ndpiRulesDir,
		"template_path", templatePath,
	)

	info, err := fs.Stat(ndpiRulesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("nDPI rules directory not found: %s", ndpiRulesDir)
		}
		return fmt.Errorf("cannot access nDPI rules directory (%s): %w", ndpiRulesDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("nDPI rules path is not a directory: %s", ndpiRulesDir)
	}

	files, err := fs.Glob(filepath.Join(ndpiRulesDir, "*"))
	if err != nil {
		return fmt.Errorf("failed to list files in rules directory (%s): %w", ndpiRulesDir, err)
	}
	if len(files) == 0 {
		logger.Warnw("nDPI rules directory is empty",
			"path", ndpiRulesDir,
		)
	}

	tmplInfo, err := fs.Stat(templatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("suricata template not found: %s", templatePath)
		}
		return fmt.Errorf("cannot access template (%s): %w", templatePath, err)
	}
	if tmplInfo.IsDir() {
		return fmt.Errorf("suricata template must be a file, got a directory: %s", templatePath)
	}

	logger.Infow("Local resources are valid")
	return nil
}
