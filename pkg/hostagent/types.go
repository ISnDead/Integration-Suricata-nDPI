package hostagent

import "time"

type Deps struct {
	SocketPath      string
	SuricataCfgPath string
	NDPIPluginPath  string
	SuricataUnit    string

	RestartTimeout time.Duration
}
