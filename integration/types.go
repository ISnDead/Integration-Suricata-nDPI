package integration

import (
	"net"
	"time"

	"integration-suricata-ndpi/pkg/executil"
	"integration-suricata-ndpi/pkg/fsutil"
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

	SuricataSCPath string
	ReloadCommand  string
	ReloadTimeout  time.Duration

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
	Apply        ApplyConfigOptions
	NDPIValidate NDPIValidateOptions
}
