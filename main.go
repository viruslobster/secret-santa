package main

import (
	"fmt"

	"github.com/viruslobster/secret-santa/server"
)

func main() {
	if err := server.Start("8080"); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
