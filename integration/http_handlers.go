package integration

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"integration-suricata-ndpi/pkg/agentclient"
	"integration-suricata-ndpi/pkg/logger"
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

	logger.Infow("HTTP ndpi enable: request received", "remote", req.RemoteAddr)

	resp, err := r.callHostAgent(req.Context(), true)
	if err != nil {
		logger.Errorw("HTTP ndpi enable: host-agent call failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	logger.Infow("HTTP ndpi enable: done",
		"ok", resp.OK,
		"changed", resp.Changed,
		"enabled", resp.Enabled,
		"message", resp.Message,
	)

	writeJSON(w, http.StatusOK, resp)
}

func (r *Runner) handleNDPIDisable(w http.ResponseWriter, req *http.Request) {
	if !requireMethod(w, req, http.MethodPost) {
		return
	}

	logger.Infow("HTTP ndpi disable: request received", "remote", req.RemoteAddr)

	resp, err := r.callHostAgent(req.Context(), false)
	if err != nil {
		logger.Errorw("HTTP ndpi disable: host-agent call failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	logger.Infow("HTTP ndpi disable: done",
		"ok", resp.OK,
		"changed", resp.Changed,
		"enabled", resp.Enabled,
		"message", resp.Message,
	)

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

	action := "disable"
	if enable {
		action = "enable"
	}

	logger.Infow("Calling host-agent",
		"action", action,
		"socket", socket,
		"timeout", timeout,
	)

	client := agentclient.New(socket, timeout)

	var (
		resp *agentclient.ToggleResponse
		err  error
	)

	if enable {
		resp, err = client.EnableNDPI(ctx)
	} else {
		resp, err = client.DisableNDPI(ctx)
	}

	if err != nil {
		logger.Errorw("Host-agent request failed",
			"action", action,
			"socket", socket,
			"error", err,
		)
		return nil, err
	}

	logger.Infow("Host-agent response received",
		"action", action,
		"ok", resp.OK,
		"changed", resp.Changed,
		"enabled", resp.Enabled,
		"message", resp.Message,
	)

	return resp, nil
}
