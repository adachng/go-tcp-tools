// https://go.dev/blog/pipelines
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
	defer log.Default().Print("handleConn() exited")

	req := []byte("hello\n")

	nwrite, err := c.Write(req)
	if err != nil {
		log.Default().Print("net.Conn.Write() error = [", err, "]")
		return
	}

	log.Default().Print("nwrite = ", nwrite)

	buf := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			return
			// Default case makes the select non-blocking.
		default:
		}

		nread, err := c.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) { // expected
				return
			}
			log.Default().Print("net.Conn.Read() error = [", err, "]")
			return
		}

		log.Default().Print("nread = [", nread, "]")
		byteSubSeg := buf[0:nread]
		hexStr := strings.ToUpper(hex.EncodeToString(byteSubSeg))
		str := string(buf[0:nread])
		log.Default().Print("buf (HEX) = [", hexStr, "]")
		log.Default().Print("buf (ASCII) = [", str, "]")
	}
}

// Establishes TCP client to write and then keep reading.
//
// Terminates gracefully upon receiving SIGINT or SIGTERM.
//
// To terminate gracefully, signal needs to trigger the TCP endpoint to close.
func main() {
	log.Default().SetFlags(log.Default().Flags() | log.Lmicroseconds | log.Lshortfile)

	log.Default().Print("Starting net.Dial()")
	conn, err := net.Dial("tcp", destAddr)
	if err != nil {
		panic(err)
	}
	log.Default().Print("Connected with net.Dial()")

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
	log.Default().Print("ctx.Done() received with cause = [", context.Cause(ctx), "]")

	err = conn.Close()
	if err != nil {
		panic(err)
	} else {
		log.Default().Print("Connection closed successfully")
	}

	// Wait for the goroutine in case which the program is
	// ended via SIGINT or SIGTERM instead of remote peer closing.
	log.Default().Print("Waiting for goroutine")
	wg.Wait()
	log.Default().Print("Wait complete")
}
