package integration

import "net"

type SuricataClient struct {
	Conn net.Conn
	Path string
}

const (
	// SocketPath берем из: /var/run/suricata/suricata-command.socket
	SocketPath = "/var/run/suricata/suricata-command.socket"
)
