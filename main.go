package main

import (
	"fmt"

	"github.com/viruslobster/secret-santa/server"
)

func main() {
	store := server.NewInMemoryStore()
	client := server.ChatClient{
		Store: server.Store(&store),
	}
	my_server := server.Server{Chat: client}

	if err := my_server.Start("8000"); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
