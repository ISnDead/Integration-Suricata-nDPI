package hostagent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func runSuricataSC(ctx context.Context, scPath, cmdName, socketPath string) (string, error) {
	cmd := exec.CommandContext(ctx, scPath, "-c", cmdName, socketPath)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return output, fmt.Errorf("suricatasc failed: %w; output=%q", err, output)
	}
	return output, nil
}
