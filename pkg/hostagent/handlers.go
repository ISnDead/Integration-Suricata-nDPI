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

func (h *Handlers) NDPIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	enabled, line, err := integration.NDPIStatus(h.deps.SuricataCfgPath, h.deps.NDPIPluginPath)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ndpiStatusResp{
		baseResp: baseResp{OK: true},
		Enabled:  enabled,
		Line:     line,
	})
}

func (h *Handlers) NDPIEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	changed, enabledAfter, err := integration.SetNDPIEnabled(h.deps.SuricataCfgPath, h.deps.NDPIPluginPath, true)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	if changed {
		if err := RestartUnit(r.Context(), h.deps.SuricataUnit, h.deps.RestartTimeout); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	logger.Infow("NDPI enabled", "changed", changed)
	writeJSON(w, http.StatusOK, ndpiToggleResp{
		baseResp: baseResp{OK: true},
		Changed:  changed,
		Enabled:  enabledAfter,
	})
}

func (h *Handlers) NDPIDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	changed, enabledAfter, err := integration.SetNDPIEnabled(h.deps.SuricataCfgPath, h.deps.NDPIPluginPath, false)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	if changed {
		if err := RestartUnit(r.Context(), h.deps.SuricataUnit, h.deps.RestartTimeout); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	logger.Infow("NDPI disabled", "changed", changed)
	writeJSON(w, http.StatusOK, ndpiToggleResp{
		baseResp: baseResp{OK: true},
		Changed:  changed,
		Enabled:  enabledAfter,
	})
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, baseResp{OK: false, Message: msg})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
