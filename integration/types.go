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

	ReloadStatus ReloadStatus
	ReloadOutput string

	// Warnings — например: "suricatasc timeout", "reload failed"
	Warnings []string
}

type ReloadStatus string

const (
	ReloadOK      ReloadStatus = "ok"
	ReloadTimeout ReloadStatus = "timeout"
	ReloadFailed  ReloadStatus = "failed"
)
