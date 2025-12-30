package main

import (
	"fmt"
	"log"
	"os"

	"github.com/viruslobster/secret-santa/server"
)

func main() {
	secret_str, exists := os.LookupEnv("SECRET")
	if !exists {
		log.Printf("No env variable SECRET")
		os.Exit(1)
	}
	secret := []byte(secret_str)
	store := server.NewInMemoryStore()
	client := server.ChatClient{
		Store: server.Store(&store),
	}
	my_server := server.Server{Chat: &client, Secret: secret}

	if err := my_server.Start("8000"); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
