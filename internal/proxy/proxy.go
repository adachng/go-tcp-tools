// MIT License
//
// Copyright (c) 2026-present adachng (github.com/adachng)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package proxy

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Benign "errors" that should be impossible. Okay for client code to treat as error and report.
var (
	// Not really an error, but indicates listener closed more than once.
	ErrLRepeatedClose = errors.New("proxy: listener closed successfully (repeated)")

	// Not really an error, but indicates connection closed more than once.
	ErrCRepeatedClose = errors.New("proxy: connection closed successfully (repeated)")

	// Not really an error, but indicates connection was closed by remote peer.
	ErrPeerClose = errors.New("proxy: connection closed by remote peer")
)

// Actual errors that are beneficial for the client code to know.
var (
	// Invalid port number specified.
	ErrInvalidPort = errors.New("proxy: listener port is invalid")

	// Invalid inbound IPv4 remote address used to filter out.
	ErrInbIP = errors.New("proxy: invalid inbound IPv4 address (not in form \"123.456.789.012\")")

	// Invalid outbound address.
	ErrOutbAddr = errors.New("proxy: invalid outbound address with port (not in form \"123.456.789.012:1234\")")
)

type Config struct {
	// Inbound connection filter.
	SrcIP net.IP

	// Outbound connection address in the form of "192.168.0.1:1234".
	DstAddr string

	// The port number that the proxy server listens on.
	ListenPort uint
}

// [App] structure of the entire [proxy] package.
type App struct {
	c Config

	subscriber EventListener // TODO: make atomic to make hot-swappable

	rootWg     sync.WaitGroup // listener main loop and all paired connection instances
	closeLOnce sync.Once      // close the listener only once
}

// Return an [App] with specified [Config] and [EventListener] interface (can be [nil]).
func New(c Config, s EventListener) (*App, error) {
	var err error = nil

	if c.ListenPort <= 0 {
		err = errors.Join(err, ErrInvalidPort)
	}

	if c.SrcIP == nil {
		err = errors.Join(err, ErrInbIP)
	}

	strs := strings.Split(c.DstAddr, ":")
	ip := strs[0]
	if c.DstAddr == "" || len(strs) != 2 || net.ParseIP(ip) == nil {
		err = errors.Join(err, ErrOutbAddr)
	}

	if err == nil {
		return &App{
			c:          c,
			subscriber: s,
		}, nil
	}

	return nil, err
}

func (a *App) Run(ctx context.Context) {
	// Validate config.
	if a.c.SrcIP == nil {
		a.gotError("", ErrInbIP)
		return
	}
	if a.c.ListenPort <= 0 {
		a.gotError("", ErrInvalidPort)
		return
	}

	// Validate a.c.DstAddr.
	{
		strs := strings.Split(a.c.DstAddr, ":")
		if len(strs) != 2 {
			a.gotError("", ErrOutbAddr)
			return
		}

		ip := strs[0]
		if net.ParseIP(ip) == nil {
			a.gotError("", ErrOutbAddr)
			return
		}
	}

	// Start the listener.
	lPort := uint64(a.c.ListenPort)

	l, err := net.Listen("tcp", ":"+strconv.FormatUint(lPort, 10))

	if l != nil {
		a.attemptedListen(l.Addr(), err)
	} else {
		a.attemptedListen(nil, err)
	}

	if err != nil {
		return
	}

	// Remove code duplication of using [sync.Once.Go].
	closeListener := func() {
		err := l.Close()
		if l != nil {
			a.closedListener(l.Addr(), err)
		} else {
			a.closedListener(nil, err)
		}
	}

	// Defer closing of the listener.
	defer a.closeLOnce.Do(closeListener)

	a.rootWg.Go(func() {
		// Close listener in case of accept failure.
		defer a.closeLOnce.Do(closeListener)

		for {
			// Non-blocking select for context cancellation.
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Blocking accept.
			inbConn, err := l.Accept()

			// If err is not nil, inbConn is nil.
			if inbConn != nil {
				a.attemptedAccept(inbConn.LocalAddr(), inbConn.RemoteAddr(), err)
			} else {
				a.attemptedAccept(nil, nil, err)
			}

			if err != nil {
				return
			}

			// Use this function to close both the inbound and outbound connections.
			closeConn := func(uuid string, conn net.Conn) {
				err := conn.Close()
				if conn != nil {
					a.closedConn(uuid, conn.LocalAddr(), conn.RemoteAddr(), err)
				} else {
					a.closedConn(uuid, nil, nil, err)
				}
			}

			// Validate inbound connection's remote address.
			if a.c.SrcIP.String() != "0.0.0.0" && // do not validate if "0.0.0.0"
				strings.Split(inbConn.RemoteAddr().String(), ":")[0] != a.c.SrcIP.String() {
				a.failedInbConn(inbConn.LocalAddr(), a.c.SrcIP.String())
				closeConn("", inbConn)
				continue
			}

			a.validatedInbConn(inbConn.LocalAddr(), a.c.SrcIP.String())

			// Connect to specified address (outbound connection).
			outbConn, err := net.Dial("tcp", a.c.DstAddr)

			if outbConn != nil {
				a.attemptedDial(outbConn.LocalAddr(), outbConn.RemoteAddr(), err)
			} else {
				a.attemptedDial(nil, nil, err)
			}

			if err != nil {
				closeConn("", inbConn)
				continue
			}

			// Assign a UUID to this successful proxy connection.
			connUUID := uuid.New().String()

			// Instantiate hexWriter instances for connPair.
			getFuncForHW := func() func(uuid string, b []byte, srcAddr net.Addr, dstAddr net.Addr) {
				// TODO: atomic load here.
				if a.subscriber != nil {
					return a.subscriber.RelayedBytes
				}
				return nil
			}

			getAttemptedIOCopyFunc := func() func(uuid string, bytesWritten int64, err error, srcLAddr net.Addr, srcRAddr net.Addr, dstLAddr net.Addr, dstRAddr net.Addr) {
				// TODO: atomic load here.
				return a.attemptedIOCopy
			}

			inbToOutbHW := newHexWriter(getFuncForHW, connUUID, inbConn.RemoteAddr(), outbConn.RemoteAddr())
			OutbToInbHW := newHexWriter(getFuncForHW, connUUID, outbConn.RemoteAddr(), inbConn.RemoteAddr())

			// Instantiate connPair instances.
			connPair := newConnPair(closeConn, getAttemptedIOCopyFunc, connUUID, inbConn, &inbToOutbHW, outbConn, &OutbToInbHW)
			a.gotConnPair(connUUID, inbConn.LocalAddr(), inbConn.RemoteAddr(), outbConn.LocalAddr(), outbConn.RemoteAddr())

			// Defer the closing of inbound and outbound connections here to increase reactivity of shutting down proxy.
			closeInbOnce, closeOutbOnce := connPair.getSyncOnce()
			defer closeInbOnce.Do(func() { closeConn(connUUID, inbConn) })
			defer closeOutbOnce.Do(func() { closeConn(connUUID, outbConn) })

			a.rootWg.Go(func() { connPair.run(ctx) })
		}
	})

	// Wait for context cancellation.
	<-ctx.Done()

	// Close listener which triggers closing of all the connections associated with the listener, which should close the child goroutines.
	a.closeLOnce.Do(closeListener)

	// Wait for the listener and the inbound + outbound pair goroutines to complete.
	a.rootWg.Wait()
}
