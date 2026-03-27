package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/adachng/go-tcp-tools/internal/logger"

	"github.com/google/uuid"
)

type config struct {
	listenPort uint
	srcAddr    string // e.g. "192.168.0.0"
	dstAddr    string // e.g. "192.168.0.0:1234"
}

type app struct {
	c        config
	listener net.Listener
}

func parseIntoConfig() config {
	var ret config

	// Parse CLI flags:
	flag.StringVar(&ret.dstAddr,
		"dst",
		"",
		"Destination IP address in form of \"192.168.0.0:1234\"")

	flag.StringVar(&ret.srcAddr,
		"src",
		"",
		"Source IP address in form of \"192.168.0.0\" (exclude port)")

	flag.UintVar(&ret.listenPort, "p", uint(0), "Proxy listen port")

	flag.Parse()

	return ret
}

func validateConfig(c config) error {
	if c.listenPort <= 0 {
		return errors.New("Invalid c.listenPort = [" + strconv.FormatUint(uint64(c.listenPort), 10) + "]")
	}

	if len(c.dstAddr) <= 0 { // || !isValidIPv4(c.dstAddr) {
		return errors.New("Invalid c.dstAddr = [" + c.dstAddr + "] with len [" + strconv.FormatInt(int64(len(c.dstAddr)), 10) + "]")
	}

	if len(c.srcAddr) <= 0 || !isValidIPv4(c.srcAddr) {
		return errors.New("Invalid c.srcAddr = [" + c.srcAddr + "] with len [" + strconv.FormatInt(int64(len(c.srcAddr)), 10) + "]")
	}

	return nil
}

func isValidIPv4(s string) bool {
	if strings.Contains(s, ":") {
		return false
	}

	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}

	if ip.To4() == nil {
		return false
	}

	return true
}

func main() {
	var a app

	// App config:
	a.c = parseIntoConfig()

	// Log CLI input:
	{
		currentLogger := logger.Get()

		currentLogger.Printf("c.dstAddr = [%s]\n", a.c.dstAddr)
		currentLogger.Printf("c.srcAddr = [%s]\n", a.c.srcAddr)
		currentLogger.Printf("c.listenPort = [%v]\n", a.c.listenPort)
	}

	// Validate config:
	{
		err := validateConfig(a.c)
		if err != nil {
			logger.Get().Fatal(err)
		}
	}

	// Establish listener:
	{
		listenAddr := ":" + strconv.FormatUint(uint64(a.c.listenPort), 10)

		logger.Get().Printf("Proxy listening at [%s]\n", listenAddr)

		var err error
		a.listener, err = net.Listen("tcp", listenAddr)
		if err != nil {
			log.Default().Fatal(err)
		}
	}

	// Defer the listener.Close() to end of main():
	defer func() {
		err := a.listener.Close()
		if err != nil {
			log.Default().Panic(err)
		}
	}()

	// Main loop:
	for {
		conn, err := a.listener.Accept() // blocking function

		if err != nil {
			logger.Get().Printf("Accept() error: [%s]\n", err)
			continue
		}

		if true { // conn.RemoteAddr().String() == a.c.srcAddr {
			logger.Get().Printf("Accepted source connection from [%s]\n", conn.RemoteAddr().String())
			go handleConn(conn, a.c.dstAddr)
		} else {
			logger.Get().Printf("Invalid source connection accepted from [%s], closing connection\n", conn.RemoteAddr().String())
			err := conn.Close()
			if err != nil {
				logger.Get().Printf("Close() error: [%s]\n", err)
				continue
			} else {
				logger.Get().Printf("Connection closed successfully\n")
			}
		}
	}
}

func handleConn(srcConn net.Conn, dstAddr string) {
	// UUID to represent source connection:
	connId := uuid.New().String()
	logger.Get().Printf(
		"[%s]: source connection [%s] tied to UUID\n",
		connId,
		srcConn.RemoteAddr().String(),
	)

	// Defer closing source connection:
	defer func() {
		err := srcConn.Close()
		logger.Get().Printf(
			"[%s]: source connection [%s] closed\n",
			connId,
			srcConn.RemoteAddr().String(),
		)
		if err != nil {
			logger.Get().Print("[", connId, "]: ", err)
		}
	}()

	// Attempt to dial to destination:
	dstConn, err := net.Dial("tcp", dstAddr)
	if err != nil {
		logger.Get().Print("[", connId, "]: ", err)
		return
	}

	logger.Get().Printf(
		"[%s]: dialed to [%s] successfully\n",
		connId,
		dstConn.RemoteAddr().String(),
	)

	// Defer closing destination connection:
	defer func() {
		err := dstConn.Close()
		logger.Get().Printf(
			"[%s]: destination connection [%s] closed\n",
			connId,
			dstConn.RemoteAddr().String(),
		)
		if err != nil {
			logger.Get().Print("[", connId, "]: ", err)
		}
	}()

	// Concurrent stream copying:
	var s sync.WaitGroup

	// Reader from source to writer from destination:
	s.Go(func() {
		_, err := io.Copy(dstConn, srcConn)
		if err != nil {
			logger.Get().Print("[", connId, "]: ", err)
			return
		}
	})

	// Reader from destination to writer from source:
	s.Go(func() {
		_, err := io.Copy(dstConn, srcConn)
		if err != nil {
			logger.Get().Print("[", connId, "]: ", err)
			return
		}
	})

	s.Wait()
}

// func handleSrcConn(srcConn net.Conn, dstAddr string) {
// 	defer func() {
// 		err := srcConn.Close()
// 		if err != nil {
// 			logger.Get().Panic(err)
// 		}
// 	}()

// 	dstConn, err := net.Dial("tcp", dstAddr)
// 	if err != nil {
// 		logger.Get().Panic(err)
// 	}

// 	go handleDestConn(dstConn, srcConn)

// 	buf := make([]byte, 1024)

// 	for {
// 		nread, err := srcConn.Read(buf)
// 		// _ = srcConn.Read()
// 	}
// }

// func handleDestConn(dstConn net.Conn, srcConn net.Conn) {
// 	defer func() {
// 		err := dstConn.Close()
// 		if err != nil {
// 			logger.Get().Panic(err)
// 		}
// 	}()

// 	for {
// 		dstConn.Read()
// 	}
// }
