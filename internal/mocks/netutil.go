package mocks

import (
	"net"
	"time"
)

type Dialer struct {
	DialTimeoutFunc func(network, address string, timeout time.Duration) (net.Conn, error)
}

func (m *Dialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	if m.DialTimeoutFunc != nil {
		return m.DialTimeoutFunc(network, address, timeout)
	}
	return nil, nil
}
