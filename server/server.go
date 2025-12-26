package server

import (
	"fmt"
	"log"
	"net/http"
)

func Start(port string) error {
	http.HandleFunc("/greeting", greetingHandler)
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("./dist"))))
	http.HandleFunc("/", spaHandler)

	log.Printf("Starting server on port %s...", port)
	log.Printf("Serving files from current directory")
	return http.ListenAndServe(":"+port, nil)
}

func spaHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./index.html")
}

func greetingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "🎅 Ho ho ho! Merry Christmas from Santa! 🎄\n")
	fmt.Fprintf(w, "Welcome to Santa's HTTP server!\n")
}
