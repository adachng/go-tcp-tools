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
	"io"
	"net"
	"strconv"
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

	// Invalid source IPv4 address used to filter out.
	ErrSrcIP = errors.New("proxy: invalid source IPv4 address (not in form \"123.456.789.012\")")

	// Invalid destination address.
	ErrDstIP = errors.New("proxy: invalid destination address (not in form \"123.456.789.012:1234\")")
)

type Logger interface {
	Debug(v ...any)  // repeated close which should not be possible in here, but harmless
	Info(v ...any)   // bytes relayed
	Notice(v ...any) // new accepted connections and closed connections
	Error(v ...any)  // unexpected errors
}

type Config struct {
	// The port number that the proxy server listens on.
	ListenPort uint

	// Inbound connection filter.
	SrcIP net.IP

	// Outbound connection destination address in the form of "192.168.0.1:1234".
	DstAddr string
}

// Implements [io.Writer] for [io.TeeReader] to log all bytes relayed in hex.
type hexWriter struct {
	a    *App
	uuid string // UUID of the source and destination connections instance

	srcAddr net.Addr // not to be confused with source connection; the source of bytes to write
	dstAddr net.Addr // not to be confused with source connection; the destination of bytes to write to
}

func newHexWriter(a *App, uuid string, srcA net.Addr, dstA net.Addr) hexWriter {
	return hexWriter{
		a:    a,
		uuid: uuid,

		srcAddr: srcA,
		dstAddr: dstA,
	}
}

// [App] structure of the entire [proxy] package.
type App struct {
	logMu sync.Mutex // enable hot-swapping logger
	l     Logger     // interface
	c     Config

	countMu   sync.Mutex
	connCount uint
}

func (h hexWriter) Write(b []byte) (n int, err error) {
	h.a.logInf("[", h.uuid, "] ", len(b), " bytes written from ", h.srcAddr, " to ", h.dstAddr)
	return len(b), nil
}

// Returns the total number of unclosed connections (excluding listener) in this [App].
func (a *App) GetConnCount() uint {
	a.countMu.Lock()
	defer a.countMu.Unlock()

	return a.connCount
}

func (a *App) incConnCount() {
	a.countMu.Lock()
	defer a.countMu.Unlock()

	a.connCount++
}

func (a *App) decConnCount() {
	a.countMu.Lock()
	defer a.countMu.Unlock()

	a.connCount--
}

// Hot-swaps the [Logger] of this [App].
func (a *App) SetLogger(l Logger) {
	a.logMu.Lock()
	defer a.logMu.Unlock()

	a.l = l
}

func (a *App) logDeb(v ...any) {
	a.logMu.Lock()
	defer a.logMu.Unlock()

	if a.l != nil {
		a.l.Debug(v...)
	}
}

func (a *App) logInf(v ...any) {
	a.logMu.Lock()
	defer a.logMu.Unlock()

	if a.l != nil {
		a.l.Info(v...)
	}
}

func (a *App) logNot(v ...any) {
	a.logMu.Lock()
	defer a.logMu.Unlock()

	if a.l != nil {
		a.l.Notice(v...)
	}
}

func (a *App) logErr(v ...any) {
	a.logMu.Lock()
	defer a.logMu.Unlock()

	if a.l != nil {
		a.l.Error(v...)
	}
}

// Return an [App] with specified [Config] and [Logger] interface (can be [nil]).
func New(c Config, l Logger) (*App, error) {
	var err error = nil

	if c.ListenPort <= 0 {
		err = errors.Join(err, ErrInvalidPort)
	}

	if c.SrcIP == nil || c.DstAddr == "" {
		err = errors.Join(err, ErrSrcIP)
	}

	if c.DstAddr == "" {
		err = errors.Join(err, ErrDstIP)
	}

	if err == nil {
		return &App{
			c:         c,
			l:         l,
			connCount: 0,
		}, nil
	}

	return nil, err
}

// Convenience function to call [App.logErr] and then join 2 errors.
func (a *App) joinErr(err error, newErr error) error {
	a.logErr(newErr)
	return errors.Join(err, newErr)
}

func (a *App) Run(ctx context.Context) error {
	ret := error(nil)

	// Start the listener.
	lPort := uint64(a.c.ListenPort)

	l, err := net.Listen("tcp", ":"+strconv.FormatUint(lPort, 10))
	if err != nil {
		ret = a.joinErr(ret, err)
		return ret
	}

	a.logNot("proxy: listener started successfully at ", l.Addr().String())

	// [sync.Once] for closing the listener.
	var closeLOnce sync.Once

	// Remove code duplication of using [sync.Once.Go].
	closeListener := func() {
		err := l.Close()
		if errors.Is(err, net.ErrClosed) {
			ret = a.joinErr(ret, ErrLRepeatedClose)
		} else if err != nil {
			ret = a.joinErr(ret, err)
		} else {
			a.logNot("proxy: listener closed successfully")
		}
	}

	// Defer closing of the listener.
	defer closeLOnce.Do(closeListener)

	// [sync.WaitGroup] for the listener main loop which includes the connection handlers.
	var rootWg sync.WaitGroup

	rootWg.Go(func() {
		// Close listener in case of accept failure.
		defer closeLOnce.Do(closeListener)

		for {
			// Non-blocking select for context cancellation.
			select {
			case <-ctx.Done():
				break
			default:
			}

			// Blocking accept.
			srcConn, err := l.Accept()
			if errors.Is(err, net.ErrClosed) {
				// Benign case of listener closed from somewhere else.
				return
			} else if err != nil {
				// Actual unexpected error.
				ret = a.joinErr(ret, err)
				return
			}

			a.incConnCount()

			// Use this function to close both the source and destination connections.
			closeConn := func(conn net.Conn) {
				err := conn.Close()
				if errors.Is(err, net.ErrClosed) {
					ret = a.joinErr(ret, ErrCRepeatedClose)
				} else if errors.Is(err, io.EOF) {
					ret = a.joinErr(ret, ErrPeerClose)
				} else if err != nil {
					ret = a.joinErr(ret, err)
				} else {
					a.decConnCount()
					a.logNot("Connection with ", conn.RemoteAddr().String(), " closed")
				}
			}

			// Close source connection only once.
			var closeSrcCOnce sync.Once
			closeSrcConn := func() {
				a.logNot("Closing source connection")
				closeConn(srcConn)
			}

			// Defer closing the connection if listener closes by another goroutine.
			defer closeSrcCOnce.Do(closeSrcConn)

			// Validate connection source.
			if srcConn.RemoteAddr().String() != a.c.SrcIP.String() {
				closeSrcCOnce.Do(closeSrcConn)
				a.logNot("New connection from ", srcConn.RemoteAddr().String(), " rejected")
				continue
			}

			a.logNot("New connection from ", srcConn.RemoteAddr().String(), " accepted, connecting to ", a.c.DstAddr)

			// Connect to destination address.
			dstConn, err := net.Dial("tcp", a.c.DstAddr)

			if err != nil {
				ret = a.joinErr(ret, err)
				closeSrcCOnce.Do(closeSrcConn)
				continue
			}

			// Assign a UUID to this successful proxy connection.
			connUUID := uuid.New().String()

			a.logNot("Successfully connected ", srcConn.RemoteAddr().String(), " to ", a.c.DstAddr, " as ", connUUID)

			// Close destination connection only once.
			var closeDstCOnce sync.Once
			closeDstConn := func() {
				a.logNot("Closing destination connection")
				closeConn(dstConn)
			}

			defer closeDstCOnce.Do(closeDstConn)

			rootWg.Go(func() {
				// Defer closing the connections if any of the connections closes by remote peer or another goroutine.
				defer closeSrcCOnce.Do(closeSrcConn)
				defer closeDstCOnce.Do(closeDstConn)

				// One non-blocking select for context cancellation.
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Note that [io.Copy] will never return error [io.EOF].
				var wg sync.WaitGroup

				// Relay all bytes from source connection to destination connection.
				wg.Go(func() {
					defer closeSrcCOnce.Do(closeSrcConn)
					defer closeDstCOnce.Do(closeDstConn)

					hW := newHexWriter(a, connUUID, srcConn.RemoteAddr(), dstConn.RemoteAddr())
					teeR := io.TeeReader(dstConn, hW)
					bytesWritten, err := io.Copy(dstConn, teeR)
					// bytesWritten, err := io.Copy(srcConn, srcConn) // TODO: delete

					if errors.Is(err, net.ErrClosed) {
						ret = a.joinErr(ret, ErrCRepeatedClose)
					} else if errors.Is(err, io.EOF) {
						panic("io.Copy returned EOF error") // never happens according to official docs
					} else if err != nil {
						ret = a.joinErr(ret, err)
					}

					a.logInf("[", connUUID, "]: ", bytesWritten, " total bytes written from ", srcConn.RemoteAddr().String(), " to ", dstConn.RemoteAddr().String())
				})

				// Relay all bytes from destination connection to source connection.
				wg.Go(func() {
					defer closeSrcCOnce.Do(closeSrcConn)
					defer closeDstCOnce.Do(closeDstConn)

					hW := newHexWriter(a, connUUID, dstConn.RemoteAddr(), srcConn.RemoteAddr())
					teeR := io.TeeReader(srcConn, hW)
					bytesWritten, err := io.Copy(srcConn, teeR)
					// bytesWritten, err := io.Copy(srcConn, dstConn) // TODO: delete

					if errors.Is(err, net.ErrClosed) {
						ret = a.joinErr(ret, ErrCRepeatedClose)
					} else if errors.Is(err, io.EOF) {
						panic("io.Copy returned EOF error") // never happens according to official docs
					} else if err != nil {
						ret = a.joinErr(ret, err)
					}

					a.logInf("[", connUUID, "]: ", bytesWritten, " total bytes written from ", dstConn.RemoteAddr().String(), " to ", srcConn.RemoteAddr().String())
				})

				// Wait for both byte-relaying goroutines to complete.
				//
				// If one completes, it closes both connections, hence the other goroutine completes as well.
				wg.Wait()
			})
		}
	})

	// Wait for context cancellation.
	<-ctx.Done()

	// Close listener which triggers closing of all the connections associated with the listener, which should close the child goroutines.
	closeLOnce.Do(closeListener)

	// Wait for the listener and the source + destination pairing goroutines to complete.
	rootWg.Wait()

	return ret
}
