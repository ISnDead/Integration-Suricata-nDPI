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

type ndpiToggleResp struct {
	Changed bool `json:"changed"`
	Enabled bool `json:"enabled"`
}

func (h *Handlers) NDPIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	enabled, line, err := integration.NDPIStatus(h.deps.SuricataCfgPath, h.deps.NDPIPluginPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, ndpiStatusResp{Enabled: enabled, Line: line})
}

func (h *Handlers) NDPIEnable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	changed, enabledAfter, err := integration.SetNDPIEnabled(h.deps.SuricataCfgPath, h.deps.NDPIPluginPath, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if changed {
		if err := RestartUnit(r.Context(), h.deps.SuricataUnit, h.deps.RestartTimeout); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	logger.Infow("NDPI enabled", "changed", changed)
	writeJSON(w, ndpiToggleResp{Changed: changed, Enabled: enabledAfter})
}

func (h *Handlers) NDPIDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	changed, enabledAfter, err := integration.SetNDPIEnabled(h.deps.SuricataCfgPath, h.deps.NDPIPluginPath, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if changed {
		if err := RestartUnit(r.Context(), h.deps.SuricataUnit, h.deps.RestartTimeout); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	logger.Infow("NDPI disabled", "changed", changed)
	writeJSON(w, ndpiToggleResp{Changed: changed, Enabled: enabledAfter})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
