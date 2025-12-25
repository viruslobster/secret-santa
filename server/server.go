package server

import (
	"fmt"
	"log"
	"net/http"
)

func Start(port string) error {
	fs := http.FileServer(http.Dir("."))
	http.Handle("/", fs)
	http.HandleFunc("/greeting", greetingHandler)

	log.Printf("Starting server on port %s...", port)
	log.Printf("Serving files from current directory")
	return http.ListenAndServe(":"+port, nil)
}

func greetingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "🎅 Ho ho ho! Merry Christmas from Santa! 🎄\n")
	fmt.Fprintf(w, "Welcome to Santa's HTTP server!\n")
}
