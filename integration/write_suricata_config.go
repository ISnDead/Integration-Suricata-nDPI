package integration

import (
	"fmt"
	"os"
)

func WriteSuricataConfigFromTemplate(templatePath string, configCandidates []string) (targetPath string, err error) {
	tpl, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("read template: %w", err)
	}

	targetPath, err = FirstExistingPath(configCandidates)
	if err != nil {
		return "", fmt.Errorf("find target suricata.yaml: %w", err)
	}

	mode := os.FileMode(0o644)
	if st, statErr := os.Stat(targetPath); statErr == nil {
		mode = st.Mode().Perm()
	}

	if err := writeFileAtomic(targetPath, tpl, mode); err != nil {
		return "", fmt.Errorf("write config atomically: %w", err)
	}

	return targetPath, nil
}
