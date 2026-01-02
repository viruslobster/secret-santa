package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-webauthn/webauthn/webauthn"
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
	authConfig := webauthn.Config{
		RPDisplayName: "Secret Santa",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8000"},
	}
	authClient, err := webauthn.New(&authConfig)
	if err != nil {
		fmt.Printf("Failed to create WebAuthn: %v\n", err)
		os.Exit(1)
	}
	my_server := server.Server{Chat: &client, Secret: secret, WebAuthn: authClient}

	if err := my_server.Start("8000"); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
