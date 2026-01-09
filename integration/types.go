package integration

import "net"

type SuricataClient struct {
	Conn net.Conn
	Path string
}
