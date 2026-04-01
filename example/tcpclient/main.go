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

// Example program to gracefully handle all expected "errors" in TCP connection.
//
// The 2 expected errors are:
// - Operation done on already-closed socket (on local end) checked via errors.Is(err, net.ErrClosed).
// - Operation detected remote peer closing the socket checked via errors.Is(err, io.EOF).
//
// Other benign "errors" may be timeouts, resets, and cancellations.
//
// Any other errors are unusual and need attention.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const destAddr = "127.0.0.1:1234"

func handleConn(ctx context.Context, c net.Conn) {
	// Best practice to defer close of connection here, even though it may already be closed
	// as control flow.
	defer func() {
		log.Print("handleConn() exited")
		err := c.Close()
		// Handle it properly by filtering with errors.Is(err, net.ErrClosed).
		if err != nil && !errors.Is(err, net.ErrClosed) {
			log.Panic(err)
		}
	}()

	req := []byte("hello\n")

	// Blocking write "hello\n" and if already closed, just return.
	nwrite, err := c.Write(req)
	if errors.Is(err, net.ErrClosed) {
		log.Print("net.Conn.Write() on already closed")
		return
	} else if err != nil {
		// Some unknown error. Needs attention.
		log.Panic("net.Conn.Write() error = [", err, "]")
		return
	}

	log.Print("nwrite = ", nwrite)

	buf := make([]byte, 1024)

	// Main loop to print response.
	for {
		select {
		case <-ctx.Done():
			return
			// Default case makes the select non-blocking.
		default:
		}

		nread, err := c.Read(buf)
		if errors.Is(err, net.ErrClosed) {
			log.Print("net.Conn.Read() on already closed")
			return
		} else if errors.Is(err, io.EOF) {
			log.Print("net.Conn.Read() detected remote peer close")
			return
		} else if err != nil {
			log.Panic("net.Conn.Read() error = [", err, "]")
			return
		}

		log.Print("nread = [", nread, "]")
		byteSubSeg := buf[0:nread]
		hexStr := strings.ToUpper(hex.EncodeToString(byteSubSeg))
		log.Print("buf (HEX) = [", hexStr, "]")
		// NOTE: do this if desired in ASCII.
		// str := string(buf[0:nread])
		// log.Print("buf (ASCII) = [", str, "]")
	}
}

// Establishes TCP client to write and then keep reading.
//
// Terminates gracefully upon receiving SIGINT or SIGTERM.
//
// To terminate gracefully, signal needs to trigger the TCP endpoint to close.
func main() {
	log.SetFlags(log.Flags() | log.Lmicroseconds | log.Lshortfile)

	log.Print("Starting net.Dial()")
	conn, err := net.Dial("tcp", destAddr)
	if err != nil {
		log.Panic(err)
	}

	// It is also okay to defer this close, as long as it is filtered with errors.Is(err, net.ErrClosed).
	defer func() {
		log.Print("handleConn() exited")
		err := conn.Close()

		if err != nil && !errors.Is(err, net.ErrClosed) {
			log.Panic(err)
		}
	}()

	log.Print("Connected with net.Dial()")

	var wg sync.WaitGroup

	// Use [signal.NotifyContext] to get a context to pass into the worker function.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM)
	defer stop()

	// Use [sync.WaitGroup.Go] which eliminates need for both [sync.WaitGroup.Add] and [sync.WaitGroup.Done].
	wg.Go(func() {
		// Stop the signal and cleans up (e.g. signal.Stop() and close()).
		defer stop()
		handleConn(ctx, conn)
	})

	// Blocking wait for the context cancellation.
	<-ctx.Done()
	log.Print("ctx.Done() received with cause = [", context.Cause(ctx), "]")

	// Can control the flow by closing it.
	err = conn.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		log.Panic(err)
	}

	log.Print("Connection closed successfully")

	// Wait for the goroutine in case which the program is
	// ended via SIGINT or SIGTERM instead of remote peer closing.
	log.Print("Waiting for goroutine")
	wg.Wait()
	log.Print("Wait complete")
}
