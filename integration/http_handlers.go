package integration

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"integration-suricata-ndpi/pkg/agentclient"
)

func (r *Runner) handleHealth(w http.ResponseWriter, req *http.Request) {
	if !requireMethod(w, req, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (r *Runner) handlePlan(w http.ResponseWriter, req *http.Request) {
	if !requireMethod(w, req, http.MethodGet) {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	report, err := PlanConfig(r.opts.Apply)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (r *Runner) handleApply(w http.ResponseWriter, req *http.Request) {
	if !requireMethod(w, req, http.MethodPost) {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	report, err := ApplyConfig(r.opts.Apply)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (r *Runner) handleNDPIEnable(w http.ResponseWriter, req *http.Request) {
	if !requireMethod(w, req, http.MethodPost) {
		return
	}
	resp, err := r.callHostAgent(req.Context(), true)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (r *Runner) handleNDPIDisable(w http.ResponseWriter, req *http.Request) {
	if !requireMethod(w, req, http.MethodPost) {
		return
	}
	resp, err := r.callHostAgent(req.Context(), false)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (r *Runner) callHostAgent(ctx context.Context, enable bool) (*agentclient.ToggleResponse, error) {
	if r.cfg == nil {
		return nil, fmt.Errorf("config is not loaded")
	}

	socket := r.cfg.HTTP.HostAgentSocket
	if socket == "" {
		return nil, fmt.Errorf("http.host_agent_socket is empty")
	}

	timeout := r.cfg.HTTP.HostAgentTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	client := agentclient.New(socket, timeout)

	if enable {
		return client.EnableNDPI(ctx)
	}
	return client.DisableNDPI(ctx)
}
