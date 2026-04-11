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

package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/adachng/go-tcp-tools/internal/logx"
)

type hexWriter struct{ addr net.Addr }

func (h hexWriter) Write(b []byte) (n int, err error) {
	logx.Default().Info("Reflected bytes with connection", h.addr.String(), ":\n\t[", hex.EncodeToString(b), "]")
	return len(b), nil
}

func run(ctx context.Context, port uint64) {
	// Initialise the listener.
	l, err := net.Listen("tcp", ":"+strconv.FormatUint(port, 10))
	if err != nil {
		logx.Default().Error(err)
		return
	}

	ctx, cancelFunc := context.WithCancel(ctx)

	var wg sync.WaitGroup

	wg.Go(func() {
		<-ctx.Done()
		err := l.Close()
		if err != nil {
			logx.Default().Error(err)
		} else {
			logx.Default().Notice("Successfully closed listener")
		}
	})

	wg.Go(func() {
		defer cancelFunc()
		for {
			conn, err := l.Accept()
			if errors.Is(err, net.ErrClosed) {
				logx.Default().Notice(err)
				return
			} else if err != nil {
				logx.Default().Error(err)
				continue
			}

			logx.Default().Notice("Accepted connection from ", conn.RemoteAddr().String())
			wg.Go(func() {
				ctx, cancelFunc := context.WithCancel(ctx)

				var wg sync.WaitGroup

				wg.Go(func() {
					<-ctx.Done()
					err := conn.Close()
					if err != nil {
						logx.Default().Error(err)
					} else {
						logx.Default().Notice("Connection with ", conn.RemoteAddr().String(), " successfully closed")
					}
				})

				wg.Go(func() {
					defer cancelFunc()

					reader := io.TeeReader(conn, hexWriter{addr: conn.RemoteAddr()})
					bytesWritten, err := io.Copy(conn, reader)
					if err != nil {
						logx.Default().Error(err)
					} else {
						logx.Default().Notice("Read EOF from connection with ", conn.RemoteAddr().String())
					}
					logx.Default().Notice("Total bytes reflected with connection ", conn.RemoteAddr().String(), " = ", bytesWritten)
				})

				wg.Wait()
			})
		}
	})

	wg.Wait()
}

func printUsage(err error) {
	if err != nil {
		fmt.Println("Error:\n\t", err)
	}

	fmt.Println("Usage:\n\t", os.Args[0], " <PORT>")
}

func main() {
	if len(os.Args) != 2 {
		printUsage(errors.New("main: invalid argument count"))
		return
	}

	// Parse the port number string.
	portStr := os.Args[1]
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		printUsage(err)
		return
	}

	// Initialise the signal context.
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)

	// Start the server.
	run(ctx, port)
}
