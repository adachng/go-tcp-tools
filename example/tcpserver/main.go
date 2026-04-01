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
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const listenPort = ":1234" // "1234" does not work

// This function actually returns actual error upon closing [net.Listener].
//
// The rest are expected normal events when doing goroutines.
func closeListener(l net.Listener) error {
	log.Print("Closing listener at ", l.Addr().String())
	err := l.Close()

	if errors.Is(err, net.ErrClosed) {
		log.Print("Closed listener (repeated)")
		err = nil
	} else {
		log.Print("Closed listener with no errors")
		err = nil
	}

	return err
}

// This function actually returns actual error upon closing [net.Conn].
//
// The rest are expected normal events when doing goroutines.
func closeConn(c net.Conn) error {
	log.Print("Closing connection from ", c.RemoteAddr().String())
	err := c.Close()

	if errors.Is(err, net.ErrClosed) {
		log.Print("Closed connection (repeated)")
		err = nil
	} else {
		log.Print("Closed connection with no errors")
		err = nil
	}

	return err
}

func main() {
	// Configure default logger.
	log.SetFlags(log.Flags() | log.Lmicroseconds | log.Lshortfile)

	// Get listener.
	l, err := net.Listen("tcp", listenPort)
	if err != nil {
		log.Panic(err)
	}

	// Defer close of listener.
	defer func() {
		err := closeListener(l)
		if err != nil {
			log.Panic(err)
		}
	}()

	// Set up control flow.
	var wg sync.WaitGroup
	ctx, stopFunc := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer stopFunc()

	// Goroutine for accepting connections.
	wg.Go(func() {
		// Defer close of listener.
		defer func() {
			err := closeListener(l)
			if err != nil {
				log.Panic(err)
			}

			// Don't forget to unblock main goroutine, so main goroutine can further clean up.
			stopFunc()
		}()

		// Main listener loop.
		for {
			// Return if context is cancelled.
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Blocking accept.
			conn, err := l.Accept()

			// Handle accept error.
			if errors.Is(err, net.ErrClosed) {
				log.Printf("l.Accept() on closed listener = [%s]\n", err)
				return
			} else if err != nil {
				log.Panicf("l.Accept() error = [%s]\n", err)
			}

			// // The below may not be necessary, but it is fine to do so as well.
			// defer func() {
			// 	err := closeConn(conn)
			// 	if err != nil {
			// 		log.Panic(err)
			// 	}
			// }()

			// Handle connection.
			wg.Go(func() {
				// Defer close of the accepted connection. Due to control flow, placing it here is better.
				defer func() {
					err := closeConn(conn)
					if err != nil {
						log.Panic(err)
					}
				}()

				// Main loop of connection handling.
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Note that [io.Copy] reads until [io.EOF] and returns no error,
				// so it does not unblock upon [io.EOF], unlike in [net.Conn.Read].
				_, err := io.Copy(conn, conn)
				if errors.Is(err, io.EOF) {
					// Will never happen.
					log.Panic("This should never happen")
				} else if err != nil {
					log.Panicf("io.Copy() error = [%s]\n", err)
				}
				return
			})
		}
	})

	// Block until context is cancelled via signals.
	// Unblocked by signals or calling the stop function returned in the setup.
	<-ctx.Done()

	err = closeListener(l)
	if err != nil {
		log.Panic(err)
	}

	wg.Wait()
	log.Print("Program returned successfully")
}
