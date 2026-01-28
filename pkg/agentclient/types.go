package agentclient

type ToggleResponse struct {
	OK      bool   `json:"ok"`
	Changed bool   `json:"changed"`
	Message string `json:"message"`
	Enabled bool   `json:"enabled,omitempty"`
}

type EnsureSuricataResponse struct {
	OK      bool   `json:"ok"`
	Started bool   `json:"started"`
	Socket  string `json:"socket,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
