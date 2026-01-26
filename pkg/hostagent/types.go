package hostagent

import (
	"context"
	"time"

	"integration-suricata-ndpi/pkg/fsutil"
)

type SystemdManager interface {
	Restart(ctx context.Context, unit string, timeout time.Duration) error
}

type Deps struct {
	SocketPath string

	SuricataCfgPath string
	NDPIPluginPath  string

	SuricataUnit string

	SuricataSocketCandidates []string
	SuricataConnectTimeout   time.Duration

	SuricataSCPath string
	ReloadCommand  string
	ReloadTimeout  time.Duration

	RestartTimeout time.Duration
	SystemctlPath  string
	Systemd        SystemdManager
	FS             fsutil.FS
}
