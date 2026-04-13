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

type evListener struct {
	l *logx.Logger
}

func (e *evListener) GotError(uuid string, err error) {
	if uuid == "" {
		e.l.Error("Error:\n\t", err)
	} else {
		e.l.Error("[", uuid, "] error:\n\t", err)
	}
}

func (e *evListener) AttemptedListen(lAddr net.Addr, err error) {
	lAddrStr := ""
	if lAddr != nil {
		lAddrStr = "at " + lAddr.String() + " "
	}

	if err != nil {
		e.l.Error(
			"Proxy listener ",
			lAddrStr,
			"failed:\n\t",
			err,
		)
	} else {
		e.l.Notice(
			"Proxy listener successfully established at ",
			lAddr.String(),
		)
	}
}

func (e *evListener) AttemptedAccept(lAddr net.Addr, rAddr net.Addr, err error) {
	rAddrStr := ""
	if rAddr != nil {
		rAddrStr = "from " + rAddr.String() + " "
	}

	if errors.Is(err, net.ErrClosed) {
		// Benign case of listener closed from somewhere else.
		e.l.Info(
			"Attempt to accept inbound connection ",
			rAddrStr,
			"is cancelled",
		)
	} else if err != nil {
		// Actual unexpected error.
		e.l.Error(
			"Attempt to accept inbound connection from ",
			rAddrStr,
			"failed:\n\t",
			err,
		)
	} else {
		suffix := ""
		if lAddr != nil {
			suffix = "to local address " + lAddr.String()
		}

		msg := "Successfully accepted inbound connection " + rAddrStr + suffix
		e.l.Info(strings.TrimSuffix(msg, " "))
	}
}

func (e *evListener) FailedInbConn(rAddr net.Addr, match string) {
	rAddrStr := ""
	if rAddr != nil {
		rAddrStr = "from " + rAddr.String() + " "
	}

	e.l.Notice(
		"Inbound connection ",
		rAddrStr,
		"rejected (does not match ",
		match,
		")",
	)
}

func (e *evListener) ValidatedInbConn(rAddr net.Addr, match string) {
	rAddrStr := ""
	if rAddr != nil {
		rAddrStr = "from " + rAddr.String() + " "
	}

	e.l.Info(
		"Inbound connection ",
		rAddrStr,
		"validated against ",
		match,
		" successfully",
	)
}

func (e *evListener) AttemptedDial(lAddr net.Addr, rAddr net.Addr, err error) {
	lAddrStr := ""
	if lAddr != nil {
		lAddrStr = "from local address " + lAddr.String() + " "
	}

	rAddrStr := ""
	if rAddr != nil {
		rAddrStr = "to remote address " + rAddr.String() + " "
	}

	if err != nil {
		e.l.Error(
			"Outbound connection attempt ",
			lAddrStr,
			rAddrStr,
			"failed:\n\t",
			err,
		)
	} else {
		e.l.Info(
			"Successfully connected outbound ",
			lAddrStr,
			rAddrStr,
		)
	}
}

func (e *evListener) GotConnPair(uuid string, inbLAddr net.Addr, inbRAddr net.Addr, outbLAddr net.Addr, outbRAddr net.Addr) {
	prefix := ""
	if uuid != "" {
		prefix = "[" + uuid + "] "
	}

	e.l.Notice(
		prefix,
		"Proxy connection established:\n\t",
		inbRAddr.String(),
		" > ",
		inbLAddr.String(),
		" (local) > ",
		outbLAddr.String(),
		" (local) > ",
		outbRAddr.String(),
	)
}

func (e *evListener) RelayedBytes(uuid string, b []byte, srcRAddr net.Addr, dstRAddr net.Addr) {
	prefix := ""
	if uuid != "" {
		prefix = "[" + uuid + "] "
	}

	hexStr := strings.ToUpper(hex.EncodeToString(b))

	e.l.Info(
		prefix,
		"Relayed ",
		len(b),
		" bytes from ",
		srcRAddr,
		" to ",
		dstRAddr,
		":\n\t",
		hexStr,
	)
}

func (e *evListener) AttemptedIOCopy(uuid string, bytesWritten int64, err error, srcLAddr net.Addr, srcRAddr net.Addr, dstLAddr net.Addr, dstRAddr net.Addr) {
	prefix := ""
	if uuid != "" {
		prefix = "[" + uuid + "] "
	}

	prefix += "IO copy from " + srcRAddr.String() + " to " + dstRAddr.String() + " with " + strconv.FormatInt(bytesWritten, 10) + " bytes written:\n\t"

	if errors.Is(err, net.ErrClosed) {
		e.l.Info(
			prefix,
			"IO copy interrupted by connection closing",
		)
	} else if errors.Is(err, io.EOF) {
		e.l.Panic(
			prefix,
			"io.Copy returned EOF error",
		) // never happens according to official docs
	} else if err != nil {
		e.l.Notice(
			prefix,
			"Success (EOF encountered)",
		)
	}
}

func (e *evListener) ClosedConn(uuid string, lAddr net.Addr, rAddr net.Addr, err error) {
	prefix := ""
	if uuid != "" {
		prefix = "[" + uuid + "] "
	}

	if errors.Is(err, net.ErrClosed) {
		e.l.Panic(
			prefix,
			"Connection with ",
			rAddr.String(),
			" is repeatedly closed",
		)
	} else if errors.Is(err, io.EOF) {
		e.l.Info(
			prefix,
			"Connection with ",
			rAddr.String(),
			" is closed somewhere else",
		)
	} else if err != nil {
		e.l.Error(
			prefix,
			"Connection with ",
			rAddr.String(),
			" failed to close:\n\t", err,
		)
	} else {
		e.l.Notice(
			prefix,
			"Connection with ",
			rAddr.String(),
			" is closed successfully",
		)
	}
}

func (e *evListener) ClosedListener(lAddr net.Addr, err error) {
	if errors.Is(err, net.ErrClosed) {
		e.l.Panic(
			"Listener at ",
			lAddr.String(),
			" is repeatedly closed",
		)
	} else if err != nil {
		e.l.Error(
			"Listener at ",
			lAddr.String(),
			" failed to close:\n\t",
			err,
		)
	} else {
		e.l.Notice(
			"Listener at ",
			lAddr.String(),
			" is closed successfully",
		)
	}
}

func printUsage(err error) {
	if err != nil {
		fmt.Println("Error:\n\t", err)
	}

	fmt.Println("Usage:\n\t",
		os.Args[0], " <PORT> <INBOUND_IPV4> <OUTBOUND_IPV4_WITH_PORT> [LOG_FILE_PREFIX]\n",
		"Example:\n\t",
		"proxy 8080 0.0.0.0 127.0.0.1:8081 ./proxy.log",
	)
}

func main() {
	if len(os.Args) < 4 {
		printUsage(errors.New("main: not enough arguments"))
		return
	}
	if len(os.Args) > 5 {
		printUsage(errors.New("main: too many arguments"))
		return
	}

	var logger *logx.Logger
	if len(os.Args) < 5 {
		// Log to terminal.
		logger = logx.Default()
	} else {
		printUsage(errors.New("main: WIP feature"))
		return
	}

	// Log timestamp.
	logger.LogTime()

	// Log usage and PID.
	{
		pid := os.Getpid()
		str := strings.Join(os.Args, " ")
		logger.Info(
			"PID of ",
			pid,
			" with arguments:\n\t",
			str,
		)
	}

	// Parse port number to Uint.
	port, err := strconv.ParseUint(os.Args[1], 10, 16)

	if err != nil {
		printUsage(err)
		return
	}

	ip := net.ParseIP(os.Args[2])

	c := proxy.Config{
		ListenPort: uint(port),

		SrcIP:   ip,
		DstAddr: os.Args[3],
	}

	l := evListener{l: logger}

	app, err := proxy.New(c, &l)
	if err != nil {
		printUsage(err)
		return
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)

	// Configure the call depth to be inside the proxy package.
	{
		c := logger.GetConfig()
		c.CallDepth += 1
		logger.Configure(c)
	}

	app.Run(ctx)
}
