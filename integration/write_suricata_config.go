package integration

import (
	"fmt"
	"os"

	"integration-suricata-ndpi/pkg/fsutil"
)

func WriteSuricataConfigFromTemplate(templatePath string, configCandidates []string) (targetPath string, err error) {
	return WriteSuricataConfigFromTemplateWithFS(templatePath, configCandidates, nil)
}

func WriteSuricataConfigFromTemplateWithFS(templatePath string, configCandidates []string, fs fsutil.FS) (targetPath string, err error) {
	if fs == nil {
		fs = fsutil.OSFS{}
	}

	tpl, err := fs.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("read template: %w", err)
	}

	targetPath, err = FirstExistingPath(configCandidates)
	if err != nil {
		return "", fmt.Errorf("find target suricata.yaml: %w", err)
	}

	mode := os.FileMode(0o644)
	if st, statErr := fs.Stat(targetPath); statErr == nil {
		mode = st.Mode().Perm()
	}

	if err := writeFileAtomic(targetPath, tpl, mode, fs); err != nil {
		return "", fmt.Errorf("write config atomically: %w", err)
	}

	return targetPath, nil
}
