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
	"io"
	"net"
	"sync"
)

type connPair struct {
	h             *eventHandle
	closeConnFunc func(uuid string, conn net.Conn)

	uuid string

	inbConn     net.Conn
	inbToOutbHW *hexWriter

	outbConn    net.Conn
	outbToInbHW *hexWriter
}

func newConnPair(
	h *eventHandle,
	closeConnFunc func(uuid string, conn net.Conn),
	uuid string,
	inbConn net.Conn,
	inbToOutbHW *hexWriter,
	outbConn net.Conn,
	outbToInbHW *hexWriter,
) *connPair {
	return &connPair{
		h:             h,
		closeConnFunc: closeConnFunc,
		uuid:          uuid,
		inbConn:       inbConn,
		inbToOutbHW:   inbToOutbHW,
		outbConn:      outbConn,
		outbToInbHW:   outbToInbHW,
	}
}

func (c *connPair) run(ctx context.Context) {
	// Note that [io.Copy] will never return error [io.EOF].
	var wg sync.WaitGroup

	// Derive a context with cancel function to call upon EOF in any of the endpoints.
	ctx, cancelFunc := context.WithCancel(ctx)

	// Close sockets upon context cancellation.
	wg.Go(func() {
		<-ctx.Done()
		// Close all connections in this pair based on the derived context:
		c.closeConnFunc(c.uuid, c.inbConn)
		c.closeConnFunc(c.uuid, c.outbConn)
	})

	// Relay all bytes from inbound connection to outbound connection.
	wg.Go(func() {
		teeR := io.TeeReader(c.inbConn, c.inbToOutbHW)

		bytesWritten, err := io.Copy(c.outbConn, teeR)
		c.h.listener().AttemptedIOCopy(c.uuid, bytesWritten, err, c.inbConn.LocalAddr(), c.inbConn.RemoteAddr(), c.outbConn.LocalAddr(), c.outbConn.RemoteAddr())

		cancelFunc()
	})

	// Relay all bytes from outbound connection to inbound connection.
	wg.Go(func() {
		teeR := io.TeeReader(c.outbConn, c.outbToInbHW)

		bytesWritten, err := io.Copy(c.inbConn, teeR)
		c.h.listener().AttemptedIOCopy(c.uuid, bytesWritten, err, c.outbConn.LocalAddr(), c.outbConn.RemoteAddr(), c.inbConn.LocalAddr(), c.inbConn.RemoteAddr())

		cancelFunc()
	})

	// Wait for both byte-relaying goroutines to complete.
	//
	// If one completes, it closes both connections, hence the other goroutine completes as well.
	wg.Wait()
}
