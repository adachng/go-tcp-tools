package main

import (
	"flag" // https://pkg.go.dev/flag
	"fmt"
	"log"
)

var config struct {
	ipv4 string
	port uint16
}

func main() {
	var string_p *string = flag.String("a", "127.0.0.1", "Server IPv4 address")

	flag.Parse()

	// Print the specified address:
	{
		_, err := fmt.Printf("IPv4 address = [%s]\n", *string_p) // fmt.Printf("IPv4 address = [", *string_p, "]") adds space

		if err != nil {
			log.Default().Fatal(err)
		}
	}
}
