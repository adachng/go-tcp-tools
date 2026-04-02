package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/adachng/go-tcp-tools/internal/logx"
)

func sigNot(goroutine string) {
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt, syscall.SIGTERM) // os.Interrupt is syscall.SIGINT in Linux
	for {
		s := <-sigch
		fmt.Println("Got signal from ", goroutine, " goroutine: ", s)
	}
}

type Logger interface {
	Info(v ...any)
	Fatal(v ...any)
}

type A struct {
	l Logger
}

func stringFunc(s string) {
	s = strings.ReplaceAll(s, " ", "")
}

func main() {
	var ptr []any
	{
		str := ""
		log.Panic("ptr = [", len(ptr), "]")

		if str == "" {
			panic("String is nil!")
		}
		m := make(map[string]*int)
		if m["ABC"] == nil {

			panic(m["ABC"])
		}
	}

	log.Default().SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	logx.Default().SetUnderlying(nil)

	str := "A B C"
	logx.Default().Info(str)
	stringFunc(str)
	logx.Default().Fatal(str)

	return
	{
		a := A{l: logx.Default()}
		a.l.Info("ABCDDDDDD")
		a.l.Fatal("SUCECESSS")
	}

	logx.Default().LogTime()

	var b *logx.Logger = nil
	b.Info("AA")

	m := make(map[string]int)

	_, prs := m["k1"]
	logx.Default().Fatal(prs)

	var a *int = nil
	if a == nil {

	}

	// All get notified, but not in any order.
	// Also, this does not mean other goroutines have default behaviour.
	// This prevents other non-relevant goroutines to catch this and end.
	go sigNot("second")
	go sigNot("third")
	for {
		fmt.Println("A")
		time.Sleep(5 * time.Second)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	for {
		s := <-sigch
		fmt.Println("Got signal from main goroutine: ", s)
	}
	u := logx.Default().GetUnderlying()

	u.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// logx.Default().Panic("Hello, World!", []byte{0xDE, 0xAD, 0xBE, 0xEF})

	var now time.Time = time.Now()
	var offset string = now.Format("-0700")

	logx.Default().Notice(offset)
	logx.Default().Fatal(time.Now().Format("200601_2-15405"))
}
