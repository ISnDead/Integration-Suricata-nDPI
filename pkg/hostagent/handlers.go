package hostagent

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strings"

	"integration-suricata-ndpi/integration"
	"integration-suricata-ndpi/pkg/logger"
)

type Handlers struct {
	deps Deps
}

func NewHandlers(deps Deps) *Handlers {
	return &Handlers{deps: deps}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

type ndpiStatusResp struct {
	OK      bool   `json:"ok"`
	Enabled bool   `json:"enabled"`
	Line    string `json:"line,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type toggleResp struct {
	OK      bool   `json:"ok"`
	Changed bool   `json:"changed"`
	Enabled bool   `json:"enabled,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (h *Handlers) NDPIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrPublic(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	enabled, line, err := integration.NDPIStatusWithFS(
		h.deps.SuricataCfgPath,
		h.deps.NDPIPluginPath,
		h.deps.FS,
	)
	if err != nil {
		writeErrFromErr(w, err)
		return
	}

	writeJSON(w, ndpiStatusResp{OK: true, Enabled: enabled, Line: line})
}

func (h *Handlers) NDPIEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrPublic(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	changed, enabledAfter, err := integration.SetNDPIEnabledWithFS(
		h.deps.SuricataCfgPath,
		h.deps.NDPIPluginPath,
		true,
		h.deps.FS,
	)
	if err != nil {
		writeErrFromErr(w, err)
		return
	}

	if changed {
		ctx, cancel := context.WithTimeout(context.Background(), h.deps.RestartTimeout)
		defer cancel()

		if err := h.deps.Systemd.Restart(ctx, h.deps.SuricataUnit, h.deps.RestartTimeout); err != nil {
			writeErrPublic(w, http.StatusInternalServerError, "RESTART_FAILED", "failed to restart suricata", err)
			return
		}
	}

	logger.Infow("NDPI enabled", "changed", changed)
	writeJSON(w, toggleResp{
		OK:      true,
		Changed: changed,
		Enabled: enabledAfter,
		Message: "ok",
	})
}

func (h *Handlers) NDPIDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrPublic(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", nil)
		return
	}

	changed, enabledAfter, err := integration.SetNDPIEnabledWithFS(
		h.deps.SuricataCfgPath,
		h.deps.NDPIPluginPath,
		false,
		h.deps.FS,
	)
	if err != nil {
		writeErrFromErr(w, err)
		return
	}

	if changed {
		ctx, cancel := context.WithTimeout(context.Background(), h.deps.RestartTimeout)
		defer cancel()

		if err := h.deps.Systemd.Restart(ctx, h.deps.SuricataUnit, h.deps.RestartTimeout); err != nil {
			writeErrPublic(w, http.StatusInternalServerError, "RESTART_FAILED", "failed to restart suricata", err)
			return
		}
	}

	logger.Infow("NDPI disabled", "changed", changed)
	writeJSON(w, toggleResp{
		OK:      true,
		Changed: changed,
		Enabled: enabledAfter,
		Message: "ok",
	})
}

func writeErrFromErr(w http.ResponseWriter, err error) {
	status, code, msg := classifyErr(err)
	writeErrPublic(w, status, code, msg, err)
}

func writeErrPublic(w http.ResponseWriter, status int, code, msg string, err error) {
	if err != nil {
		logger.Errorw("host-agent request failed", "code", code, "status", status, "error", err)
	}
	writeJSONWithStatus(w, status, toggleResp{
		OK:      false,
		Changed: false,
		Code:    code,
		Message: msg,
	})
}

func classifyErr(err error) (status int, code, msg string) {
	if err == nil {
		return http.StatusInternalServerError, "INTERNAL", "internal error"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, "TIMEOUT", "operation timed out"
	}
	if errors.Is(err, fs.ErrNotExist) {
		return http.StatusInternalServerError, "NOT_FOUND", "required file/path not found on host"
	}
	if errors.Is(err, fs.ErrPermission) {
		return http.StatusInternalServerError, "PERMISSION", "permission denied on host"
	}
	if strings.Contains(err.Error(), "ndpi plugin line not found") {
		return http.StatusConflict, "NDPI_NOT_CONFIGURED", "ndpi plugin line not found in suricata config"
	}

	return http.StatusInternalServerError, "INTERNAL", "internal error"
}

func writeJSON(w http.ResponseWriter, v any) {
	writeJSONWithStatus(w, http.StatusOK, v)
}

func writeJSONWithStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
