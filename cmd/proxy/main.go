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
	"strings"
	"syscall"

	"github.com/adachng/go-tcp-tools/internal/logx"
	"github.com/adachng/go-tcp-tools/internal/proxy"
)

type listener struct {
}

func (l *listener) GotError(uuid string, err error) {
	if uuid == "" {
		logx.Default().Error("Error:\n\t", err)
	} else {
		logx.Default().Error("[", uuid, "] error:\n\t", err)
	}
}

func (l *listener) AttemptedListen(lAddr net.Addr, err error) {
	lAddrStr := ""
	if lAddr != nil {
		lAddrStr = "at " + lAddr.String() + " "
	}

	if err != nil {
		logx.Default().Error("Proxy listener ", lAddrStr, "failed:\n\t", err)
	} else {
		logx.Default().Notice("Proxy listener successfully established at ", lAddr.String())
	}
}

func (l *listener) AttemptedAccept(lAddr net.Addr, rAddr net.Addr, err error) {
	rAddrStr := ""
	if rAddr != nil {
		rAddrStr = "from " + rAddr.String() + " "
	}

	if errors.Is(err, net.ErrClosed) {
		// Benign case of listener closed from somewhere else.
		logx.Default().Info("Attempt to accept inbound connection ", rAddrStr, "is cancelled")
	} else if err != nil {
		// Actual unexpected error.
		logx.Default().Error("Attempt to accept inbound connection from ", rAddrStr, "failed:\n\t", err)
	} else {
		suffix := ""
		if lAddr != nil {
			suffix = "to local address " + lAddr.String()
		}

		msg := "Successfully accepted inbound connection " + rAddrStr + suffix
		logx.Default().Info(strings.TrimSuffix(msg, " "))
	}
}

func (l *listener) FailedSrcConn(rAddr net.Addr, match string) {
	rAddrStr := ""
	if rAddr != nil {
		rAddrStr = "from " + rAddr.String() + " "
	}

	logx.Default().Notice("Inbound connection ", rAddrStr, "rejected (does not match ", match, ")")
}

func (l *listener) ValidatedSrcConn(rAddr net.Addr, match string) {
	rAddrStr := ""
	if rAddr != nil {
		rAddrStr = "from " + rAddr.String() + " "
	}

	logx.Default().Info("Inbound connection ", rAddrStr, "validated against ", match, " successfully")
}

func (l *listener) AttemptedDial(lAddr net.Addr, rAddr net.Addr, err error) {
	lAddrStr := ""
	if lAddr != nil {
		lAddrStr = "from local address " + lAddr.String() + " "
	}

	rAddrStr := ""
	if rAddr != nil {
		rAddrStr = "to remote address " + rAddr.String() + " "
	}

	if err != nil {
		logx.Default().Error("Outbound connection attempt ", lAddrStr, rAddrStr, "failed:\n\t", err)
	} else {
		logx.Default().Info("Successfully connected outbound ", lAddrStr, rAddrStr)
	}
}

func (l *listener) GotConnPair(uuid string, srcLAddr net.Addr, srcRAddr net.Addr, dstLAddr net.Addr, dstRAddr net.Addr) {
	prefix := ""
	if uuid != "" {
		prefix = "[" + uuid + "] "
	}

	logx.Default().Notice(prefix, "Proxy connection established:\n\t", srcRAddr.String(), " > ", srcLAddr.String(), " > ", dstLAddr.String(), " > ", dstRAddr.String())
}

func (l *listener) RelayedBytes(uuid string, b []byte, srcRAddr net.Addr, dstRAddr net.Addr) {
	prefix := ""
	if uuid != "" {
		prefix = "[" + uuid + "] "
	}

	hexStr := strings.ToUpper(hex.EncodeToString(b))

	logx.Default().Info(prefix, "Relayed ", len(b), " bytes from ", srcRAddr, " to ", dstRAddr, ":\n\t", hexStr)
}

func (l *listener) AttemptedIOCopy(uuid string, bytesWritten int64, err error, srcLAddr net.Addr, srcRAddr net.Addr, dstLAddr net.Addr, dstRAddr net.Addr) {
	prefix := ""
	if uuid != "" {
		prefix = "[" + uuid + "] "
	}

	prefix += "IO copy from " + srcRAddr.String() + " to " + dstRAddr.String() + " with " + strconv.FormatInt(bytesWritten, 10) + " bytes written:\n\t"

	if errors.Is(err, net.ErrClosed) {
		logx.Default().Info(prefix, "IO copy interrupted by connection closing")
	} else if errors.Is(err, io.EOF) {
		logx.Default().Panic(prefix, "io.Copy returned EOF error") // never happens according to official docs
	} else if err != nil {
		logx.Default().Notice(prefix, "Success (EOF encountered)")
	}
}

func (l *listener) ClosedConn(uuid string, lAddr net.Addr, rAddr net.Addr, err error) {
	prefix := ""
	if uuid != "" {
		prefix = "[" + uuid + "] "
	}

	if errors.Is(err, net.ErrClosed) {
		logx.Default().Panic(prefix, "Connection with ", rAddr.String(), " is repeatedly closed")
	} else if errors.Is(err, io.EOF) {
		logx.Default().Info(prefix, "Connection with ", rAddr.String(), " is closed somewhere else")
	} else if err != nil {
		logx.Default().Error(prefix, "Connection with ", rAddr.String(), " failed to close:\n\t", err)
	} else {
		logx.Default().Notice(prefix, "Connection with ", rAddr.String(), " is closed successfully")
	}
}

func (l *listener) ClosedListener(lAddr net.Addr, err error) {
	if errors.Is(err, net.ErrClosed) {
		logx.Default().Panic("Listener at ", lAddr.String(), " is repeatedly closed")
	} else if err != nil {
		logx.Default().Error("Listener at ", lAddr.String(), " failed to close:\n\t", err)
	} else {
		logx.Default().Notice("Listener at ", lAddr.String(), " is closed successfully")
	}
}

func printUsage(err error) {
	if err != nil {
		fmt.Println("Error:\n\t", err)
	}

	fmt.Println("Usage:\n\t", os.Args[0], " <PORT> <INBOUND_IPV4> <OUTBOUND_IPV4_WITH_PORT>")
}

func main() {
	if len(os.Args) < 4 {
		printUsage(errors.New("main: not enough arguments"))
		return
	}

	logx.Default().LogTime()

	args := os.Args
	port, err := strconv.ParseUint(args[1], 10, 16)

	if err != nil {
		printUsage(err)
		return
	}

	ip := net.ParseIP(args[2])

	c := proxy.Config{
		ListenPort: uint(port),

		SrcIP:   ip,
		DstAddr: args[3],
	}

	l := listener{}

	app, err := proxy.New(c, &l)
	if err != nil {
		printUsage(err)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)

	// Configure the call depth to be inside the proxy package.
	{
		c := logx.Default().GetConfig()
		c.CallDepth += 2
		logx.Default().Configure(c)
	}

	app.Run(ctx)
}
