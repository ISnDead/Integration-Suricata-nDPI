package integration

import (
	"context"
	"encoding/json"
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

func writeErrPublic(w http.ResponseWriter, code string, status int, err error) {
	logger.Errorw("host-agent request failed", "code", code, "status", status, "error", err)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      false,
		"code":    code,
		"message": err.Error(),
	})
}

func (r *Runner) handleApply(w http.ResponseWriter, req *http.Request) {
	if !requireMethod(w, req, http.MethodPost) {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.ensureSuricataViaHostAgent(req.Context()); err != nil {
		logger.Errorw("HTTP apply: suricata ensure failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	report, err := ApplyConfigWithContext(req.Context(), r.opts.Apply)
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
		logger.Errorw("HTTP ndpi enable: host-agent call failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	logger.Infow("HTTP ndpi enable: done",
		"ok", resp.OK,
		"changed", resp.Changed,
		"message", resp.Message,
	)
	writeJSON(w, http.StatusOK, resp)
}

func (r *Runner) handleNDPIDisable(w http.ResponseWriter, req *http.Request) {
	if !requireMethod(w, req, http.MethodPost) {
		return
	}
	resp, err := r.callHostAgent(req.Context(), false)
	if err != nil {
		logger.Errorw("HTTP ndpi disable: host-agent call failed", "error", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	logger.Infow("HTTP ndpi disable: done",
		"ok", resp.OK,
		"changed", resp.Changed,
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

	client := agentclient.New(socket, timeout)

	if enable {
		return client.EnableNDPI(ctx)
	}
	return client.DisableNDPI(ctx)
}

func (r *Runner) ensureSuricataViaHostAgent(ctx context.Context) error {
	if r.cfg == nil {
		return fmt.Errorf("config is not loaded")
	}

	socket := r.cfg.HTTP.HostAgentSocket
	if socket == "" {
		return fmt.Errorf("http.host_agent_socket is empty")
	}

	timeout := r.cfg.HTTP.HostAgentTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	client := agentclient.New(socket, timeout)

	resp, err := client.EnsureSuricataStarted(ctx)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("host-agent ensure suricata failed: %s (%s)", resp.Message, resp.Code)
	}

	logger.Infow("Suricata ensured via host-agent",
		"started", resp.Started,
		"socket", resp.Socket,
	)
	return nil
}
