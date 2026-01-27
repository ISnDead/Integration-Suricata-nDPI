package integration

import (
	"net"
	"time"

	"integration-suricata-ndpi/pkg/executil"
	"integration-suricata-ndpi/pkg/fsutil"
	"integration-suricata-ndpi/pkg/netutil"
	"integration-suricata-ndpi/pkg/systemd"
)

type SuricataClient struct {
	Conn net.Conn
	Path string
}

type ApplyConfigReport struct {
	TargetConfigPath string
	ReloadCommand    string
	ReloadTimeout    time.Duration
	ReloadStatus     ReloadStatus
	ReloadOutput     string
	Warnings         []string
}

type ApplyConfigOptions struct {
	TemplatePath     string
	ConfigCandidates []string
	SocketCandidates []string

	SuricataSCPath  string
	SuricataBinPath string
	ReloadCommand   string
	ReloadTimeout   time.Duration

	CommandRunner executil.Runner
	FS            fsutil.FS
}

type NDPIValidateOptions struct {
	NDPIPluginPath       string
	NDPIRulesDir         string
	SuricataTemplatePath string
	SuricataSCPath       string
	ReloadCommand        string
	ReloadTimeout        time.Duration

	ExpectedRulesPattern string
	FS                   fsutil.FS
}

type NDPIToggleOptions struct {
	TemplatePath   string
	NDPIPluginPath string
	Enable         bool
}

type ReloadStatus string

const (
	ReloadOK      ReloadStatus = "ok"
	ReloadTimeout ReloadStatus = "timeout"
	ReloadFailed  ReloadStatus = "failed"
)

type RunnerOptions struct {
	Apply         ApplyConfigOptions
	NDPIValidate  NDPIValidateOptions
	SuricataStart SuricataStartOptions
}

type SuricataStartOptions struct {
	SocketCandidates []string
	SystemctlPath    string
	SystemdUnit      string
	StartTimeout     time.Duration
	Dialer           netutil.Dialer
	Systemd          systemd.Manager
}
