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
	closeConnFunc       func(uuid string, conn net.Conn)
	attemptedIOCopyFunc func() func(uuid string, bytesWritten int64, err error, srcLAddr net.Addr, srcRAddr net.Addr, dstLAddr net.Addr, dstRAddr net.Addr)

	uuid string
	wg   sync.WaitGroup

	closeInbOnce  sync.Once
	closeOutbOnce sync.Once

	inbConn     net.Conn
	inbToOutbHW *hexWriter

	outbConn    net.Conn
	outbToInbHW *hexWriter
}

func newConnPair(
	closeConnFunc func(uuid string, conn net.Conn),
	attemptedIOCopyFunc func() func(uuid string, bytesWritten int64, err error, srcLAddr net.Addr, srcRAddr net.Addr, dstLAddr net.Addr, dstRAddr net.Addr),
	uuid string,
	inbConn net.Conn,
	inbToOutbHW *hexWriter,
	outbConn net.Conn,
	outbToInbHW *hexWriter,
) *connPair {
	return &connPair{
		closeConnFunc:       closeConnFunc,
		attemptedIOCopyFunc: attemptedIOCopyFunc,
		uuid:                uuid,
		inbConn:             inbConn,
		inbToOutbHW:         inbToOutbHW,
		outbConn:            outbConn,
		outbToInbHW:         outbToInbHW,
	}
}

func (c *connPair) getSyncOnce() (*sync.Once, *sync.Once) {
	return &c.closeInbOnce, &c.closeOutbOnce
}

func (c *connPair) run(ctx context.Context) {
	// Defer closing the connections if any of the connections closes by remote peer or another goroutine.
	defer c.closeInbOnce.Do(func() { c.closeConnFunc(c.uuid, c.inbConn) })
	defer c.closeOutbOnce.Do(func() { c.closeConnFunc(c.uuid, c.outbConn) })

	// One non-blocking select for context cancellation.
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Note that [io.Copy] will never return error [io.EOF].
	var wg sync.WaitGroup

	// Relay all bytes from inbound connection to outbound connection.
	wg.Go(func() {
		defer c.closeInbOnce.Do(func() { c.closeConnFunc(c.uuid, c.inbConn) })
		defer c.closeOutbOnce.Do(func() { c.closeConnFunc(c.uuid, c.outbConn) })

		teeR := io.TeeReader(c.inbConn, c.inbToOutbHW)
		bytesWritten, err := io.Copy(c.outbConn, teeR)
		if c.attemptedIOCopyFunc != nil && c.attemptedIOCopyFunc() != nil {
			c.attemptedIOCopyFunc()(c.uuid, bytesWritten, err, c.inbConn.LocalAddr(), c.inbConn.RemoteAddr(), c.outbConn.LocalAddr(), c.outbConn.RemoteAddr())
		}
	})

	// Relay all bytes from outbound connection to inbound connection.
	wg.Go(func() {
		defer c.closeInbOnce.Do(func() { c.closeConnFunc(c.uuid, c.inbConn) })
		defer c.closeOutbOnce.Do(func() { c.closeConnFunc(c.uuid, c.outbConn) })

		teeR := io.TeeReader(c.outbConn, c.outbToInbHW)
		bytesWritten, err := io.Copy(c.inbConn, teeR)
		if c.attemptedIOCopyFunc != nil && c.attemptedIOCopyFunc() != nil {
			c.attemptedIOCopyFunc()(c.uuid, bytesWritten, err, c.outbConn.LocalAddr(), c.outbConn.RemoteAddr(), c.inbConn.LocalAddr(), c.inbConn.RemoteAddr())
		}
	})

	// Wait for both byte-relaying goroutines to complete.
	//
	// If one completes, it closes both connections, hence the other goroutine completes as well.
	wg.Wait()
}
