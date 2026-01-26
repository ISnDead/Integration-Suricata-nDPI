package hostagent

import (
	"context"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"integration-suricata-ndpi/integration"
)

type suricataReloadResp struct {
	OK      bool   `json:"ok"`
	Socket  string `json:"socket,omitempty"`
	Command string `json:"command,omitempty"`
	Output  string `json:"output,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (h *Handlers) SuricataReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrPublic(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	if strings.TrimSpace(h.deps.SuricataSCPath) == "" {
		writeErrPublic(w, http.StatusInternalServerError, "SURICATASC_NOT_CONFIGURED", "paths.suricatasc is empty", nil)
		return
	}

	cmdName := strings.TrimSpace(h.deps.ReloadCommand)
	if cmdName == "" {
		cmdName = "reload-rules"
	}

	timeout := h.deps.ReloadTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	socketPath, err := integration.FirstExistingSocket(h.deps.SuricataSocketCandidates)
	if err != nil {
		writeErrPublic(w, http.StatusGatewayTimeout, "SURICATA_NOT_READY", "suricata control socket not reachable", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.deps.SuricataSCPath, "-c", cmdName, socketPath)
	out, runErr := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if runErr != nil {
		writeErrPublic(w, http.StatusInternalServerError, "RELOAD_FAILED", "suricatasc reload failed", runErr)
		return
	}

	writeJSONWithStatus(w, http.StatusOK, suricataReloadResp{
		OK:      true,
		Socket:  socketPath,
		Command: cmdName,
		Output:  output,
		Message: "ok",
	})
}
