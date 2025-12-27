package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func Start(port string) error {
	http.HandleFunc("/greeting", greetingHandler)
	http.HandleFunc("/api/sendChat", chatHandler)
	http.HandleFunc("/api/user", userHandler)
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("./dist"))))
	http.HandleFunc("/", spaHandler)

	log.Printf("Starting server on port %s...", port)
	log.Printf("Serving files from current directory")
	return http.ListenAndServe(":"+port, nil)
}

func spaHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got spa request")
	http.ServeFile(w, r, "./index.html")
}

func greetingHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "🎅 Ho ho ho! Merry Christmas from Santa! 🎄\n")
	fmt.Fprintf(w, "Welcome to Santa's HTTP server!\n")
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got chat request")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user := map[string]string{
		"name": "Mr. Foo",
		"id":   "123",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
