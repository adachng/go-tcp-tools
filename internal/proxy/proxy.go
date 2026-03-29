package proxy

import (
	"errors"
	"net"
	"sync"
)

type Logger interface {
	Debug(v ...any)
	Info(v ...any)
	Warn(v ...any)
	Error(v ...any)
	Panic(v ...any)
	Fatal(v ...any)
}

type Stats struct {
	mu sync.Mutex

	Conns []net.Conn
}

type Config struct {
	mu sync.Mutex

	// The port number that the proxy server listens on.
	ListenPort uint

	// Inbound connection filter.
	SrcAddr net.IP

	// Outbound connection destination address.
	DstAddr net.IP
}

type App struct {
	l Logger
	c *Config
	s *Stats
}

func New(c *Config, l Logger) (*App, error) {
	if c.ListenPort <= 0 {
		return nil, errors.New("listen port cannot be 0")
	}

	if c.SrcAddr == nil || c.DstAddr == nil {
		return nil, errors.New("missing IP")
	}

	return &App{
		c: c,
		l: l,
	}, nil
}

func (a *App) Run() {

}
