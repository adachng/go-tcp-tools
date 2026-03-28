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
	var ret error

	if c.listenPort <= 0 {
		ret = errors.New("Invalid c.listenPort = [" + strconv.FormatUint(uint64(c.listenPort), 10) + "]")
	}

	if len(c.dstAddr) <= 0 { // || !isValidIPv4(c.dstAddr) {
		ret = errors.New("Invalid c.dstAddr = [" + c.dstAddr + "] with len [" + strconv.FormatInt(int64(len(c.dstAddr)), 10) + "]")
	}

	if len(c.srcAddr) <= 0 || !isValidIPv4(c.srcAddr) {
		ret = errors.New("Invalid c.srcAddr = [" + c.srcAddr + "] with len [" + strconv.FormatInt(int64(len(c.srcAddr)), 10) + "]")
	}

	return ret
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
	// App config:
	c := parseIntoConfig()

	// Validate config:
	{
		err := validateConfig(c)
		if err != nil {
			logger.Get().Fatal(err)
		}
	}

	// Log CLI input:
	{
		currentLogger := logger.Get()

		currentLogger.Printf("c.dstAddr = [%s]\n", c.dstAddr)
		currentLogger.Printf("c.srcAddr = [%s]\n", c.srcAddr)
		currentLogger.Printf("c.listenPort = [%v]\n", c.listenPort)
	}

	// Establish listener:
	listenAddr := ":" + strconv.FormatUint(uint64(c.listenPort), 10)

	logger.Get().Printf("Proxy listening at [%s]\n", listenAddr)

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Default().Fatal(err)
	}

	// Defer the listener.Close() to end of main():
	defer func() {
		err := listener.Close()
		if err != nil {
			log.Default().Panic(err)
		}
	}()

	// Main loop:
	for {
		conn, err := listener.Accept() // blocking function

		if err != nil {
			logger.Get().Printf("Accept() error: [%s]\n", err)
			continue
		}

		if true { // conn.RemoteAddr().String() == c.srcAddr {
			logger.Get().Printf("Accepted source connection from [%s]\n", conn.RemoteAddr().String())
			go handleConn(conn, c.dstAddr)
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
		"[%s]: Source connection [%s] tied to UUID\n",
		connId,
		srcConn.RemoteAddr().String(),
	)

	// Defer closing source connection:
	defer func() {
		err := srcConn.Close()
		logger.Get().Printf(
			"[%s]: Source connection [%s] closed\n",
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
		"[%s]: Dialed to [%s] successfully\n",
		connId,
		dstConn.RemoteAddr().String(),
	)

	// Defer closing destination connection:
	defer func() {
		err := dstConn.Close()
		logger.Get().Printf(
			"[%s]: Destination connection [%s] closed\n",
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
