package integration

import (
	"net"
	"time"
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
}

type NDPIValidateOptions struct {
	NDPIPluginPath       string
	NDPIRulesDir         string
	SuricataTemplatePath string
	SuricataSCPath       string
	ReloadCommand        string
	ReloadTimeout        time.Duration

	ExpectedRulesPattern string
}

type ReloadStatus string

const (
	ReloadOK      ReloadStatus = "ok"
	ReloadTimeout ReloadStatus = "timeout"
	ReloadFailed  ReloadStatus = "failed"
)
