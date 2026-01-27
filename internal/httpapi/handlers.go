package httpapi

import (
	"context"
	"net/http"

	"integration-suricata-ndpi/pkg/logger"
)

type Deps struct {
	Plan      func(ctx context.Context) (any, error)
	Reconcile func(ctx context.Context) (any, error)
	Apply     func(ctx context.Context) (any, error)

	EnsureSuricata func(ctx context.Context) error
	EnableNDPI     func(ctx context.Context) (any, error)
	DisableNDPI    func(ctx context.Context) (any, error)
}

type Handlers struct {
	deps Deps
}

func NewHandlers(deps Deps) *Handlers {
	return &Handlers{deps: deps}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) Plan(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		resp, err := h.deps.Plan(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return

	case http.MethodPost:
		if h.deps.Reconcile == nil {
			writeJSONError(w, http.StatusInternalServerError, "reconcile is not configured")
			return
		}
		resp, err := h.deps.Reconcile(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
}

func (h *Handlers) Apply(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	if err := h.deps.EnsureSuricata(r.Context()); err != nil {
		logger.Errorw("HTTP apply: suricata ensure failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	resp, err := h.deps.Apply(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) NDPIEnable(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	resp, err := h.deps.EnableNDPI(r.Context())
	if err != nil {
		logger.Errorw("HTTP ndpi enable: failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) NDPIDisable(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	resp, err := h.deps.DisableNDPI(r.Context())
	if err != nil {
		logger.Errorw("HTTP ndpi disable: failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
