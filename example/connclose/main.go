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

// Simple application to accept a connection,
// close the listener, sleep, then close the connection.
package main

import (
	"log"
	"net"
	"time"
)

func main() {
	log.SetFlags(log.Flags() | log.Lmicroseconds | log.LstdFlags)

	log.Print("Establishing listener...")
	l, err := net.Listen("tcp", ":8090")
	if err != nil {
		panic(err)
	}

	log.Print("Calling accept on listener...")
	c, err := l.Accept()
	if err != nil {
		panic(err)
	}

	log.Print("Accepted connection, closing listener...")
	err = l.Close()
	if err != nil {
		panic(err)
	}

	log.Print("Sleeping for 5 seconds and then closing connection...")
	time.Sleep(5 * time.Second)
	err = c.Close()
	if err != nil {
		panic(err)
	}

	log.Print("Connection closed successfully!")
}
