package hostagent

import (
	"encoding/json"
	"net/http"

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
	Enabled bool   `json:"enabled"`
	Line    string `json:"line"`
}

type toggleResp struct {
	OK      bool   `json:"ok"`
	Changed bool   `json:"changed"`
	Enabled bool   `json:"enabled,omitempty"`
	Message string `json:"message,omitempty"`
}

func (h *Handlers) NDPIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	enabled, line, err := integration.NDPIStatusWithFS(
		h.deps.SuricataCfgPath,
		h.deps.NDPIPluginPath,
		h.deps.FS,
	)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, ndpiStatusResp{Enabled: enabled, Line: line})
}

func (h *Handlers) NDPIEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	changed, enabledAfter, err := integration.SetNDPIEnabledWithFS(
		h.deps.SuricataCfgPath,
		h.deps.NDPIPluginPath,
		true,
		h.deps.FS,
	)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	if changed {
		if err := h.deps.Systemd.Restart(r.Context(), h.deps.SuricataUnit, h.deps.RestartTimeout); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
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
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	changed, enabledAfter, err := integration.SetNDPIEnabledWithFS(
		h.deps.SuricataCfgPath,
		h.deps.NDPIPluginPath,
		false,
		h.deps.FS,
	)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	if changed {
		if err := h.deps.Systemd.Restart(r.Context(), h.deps.SuricataUnit, h.deps.RestartTimeout); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
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

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSONWithStatus(w, status, toggleResp{
		OK:      false,
		Changed: false,
		Message: msg,
	})
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
