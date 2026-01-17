package hostagent

import "time"

type Deps struct {
	SocketPath      string
	SuricataCfgPath string
	NDPIPluginPath  string
	SuricataUnit    string

	RestartTimeout time.Duration
}

type baseResp struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}

type ndpiStatusResp struct {
	baseResp
	Enabled bool   `json:"enabled"`
	Line    string `json:"line,omitempty"`
}

type ndpiToggleResp struct {
	baseResp
	Changed bool `json:"changed"`
	Enabled bool `json:"enabled"`
}
