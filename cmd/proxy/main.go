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
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/adachng/go-tcp-tools/internal/logx"
	"github.com/adachng/go-tcp-tools/internal/proxy"
)

type proxyLoggerAdapter struct {
	l *logx.Logger
}

func (l proxyLoggerAdapter) Debug(v ...any)  { l.l.Debug(v...) }
func (l proxyLoggerAdapter) Info(v ...any)   { l.l.Info(v...) }
func (l proxyLoggerAdapter) Notice(v ...any) { l.l.Notice(v...) }
func (l proxyLoggerAdapter) Error(v ...any)  { l.l.Error(v...) }

// For the [proxy.Logger] interface since [logx.Logger] does not have this exact function.
func (l proxyLoggerAdapter) IncCallDepth() {
	c := l.l.GetConfig()
	c.CallDepth++
	l.l.Configure(c)
}

func main() {
	c := proxy.Config{
		ListenPort: uint(8090),

		SrcIP:   net.ParseIP("127.0.0.1"),
		DstAddr: "127.0.0.1:8091",
	}

	p := proxyLoggerAdapter{l: logx.Default()}

	app, err := proxy.New(c, p)
	if err != nil {
		panic(err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)

	err = app.Run(ctx)

	if err != nil {
		panic(err)
	}
}
