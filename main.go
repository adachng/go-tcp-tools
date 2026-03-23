package main

import (
	"fmt"
	"os"

	"github.com/adachng/go-tcp-tools/hello"
)

func main() {
	temp_str := hello.Hello("World")
	_, err := fmt.Fprintf(os.Stdout, "%s\n", temp_str) // https://pkg.go.dev/fmt#Fprintf

	// Error handling:
	if err != nil {
		panic(err) // https://pkg.go.dev/builtin#panic
	}
}

// https://go.dev/doc/modules/managing-dependencies#naming_module
