package server

import (
	"encoding/json"
	"log"
	"net/http"
)

func Start(port string) error {
	http.HandleFunc("/api/sendChat", chatHandler)
	http.HandleFunc("/api/user", userHandler)
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("./dist"))))
	http.HandleFunc("/server", serverHandler)
	http.HandleFunc("/server/", serverHandler)
	http.HandleFunc("/", indexHandler)

	log.Printf("Starting server on port %s...", port)
	log.Printf("Serving files from current directory")
	return http.ListenAndServe(":"+port, nil)
}

func serverHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./html/server.html")
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	log.Printf("Got spa request")
	http.ServeFile(w, r, "./html/index.html")
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
