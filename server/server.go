package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Server struct {
	Chat ChatClient
}

func (s *Server) Start(port string) error {
	http.HandleFunc("/api/chat/publish", s.chatHandler)
	http.HandleFunc("/api/thread", s.threadHandler)
	http.HandleFunc("/api/thread/subscribe", s.subscribeThreadHandler)
	http.HandleFunc("/api/user", s.userHandler)
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("./dist"))))
	http.HandleFunc("/server", s.serverHandler)
	http.HandleFunc("/server/", s.serverHandler)
	http.HandleFunc("/", s.indexHandler)

	log.Printf("Starting server on port %s...", port)
	log.Printf("Serving files from current directory")
	return http.ListenAndServe(":"+port, nil)
}

func (s *Server) serverHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./html/server.html")
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	log.Printf("Got spa request")
	http.ServeFile(w, r, "./html/index.html")
}

type PublishChatRequest struct {
	Message string `json:"message"`
	User    string `json:"user"`
	Thread  uint64 `json:"thread"`
}

func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Got chat request")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PublishChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "While Deserialize PublishChatRequest", http.StatusBadRequest)
		return
	}
	chat := ChatMessage{Message: req.Message, User: req.User, Timestamp: time.Now().Unix()}
	err := s.Chat.Publish(chat, ThreadId(req.Thread))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

type GetThreadResponse struct {
	Id    ThreadId      `json:"id"`
	Chats []ChatMessage `json:"chats"`
}

func (s *Server) threadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	threadParam := r.URL.Query().Get("id")
	if threadParam == "" {
		http.Error(w, "Missing thread parameter", http.StatusBadRequest)
		return
	}

	threadID, err := strconv.ParseUint(threadParam, 10, 64)
	if err != nil {
		http.Error(w, "Invalid thread parameter", http.StatusBadRequest)
		return
	}
	chats, err := s.Chat.Chats(ThreadId(threadID))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := GetThreadResponse{
		Chats: chats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) userHandler(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) subscribeThreadHandler(w http.ResponseWriter, r *http.Request) {
	threadParam := r.URL.Query().Get("id")
	if threadParam == "" {
		http.Error(w, "Missing thread parameter", http.StatusBadRequest)
		return
	}

	threadIDRaw, err := strconv.ParseUint(threadParam, 10, 64)
	if err != nil {
		http.Error(w, "Invalid thread parameter", http.StatusBadRequest)
		return
	}
	threadID := ThreadId(threadIDRaw)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	msgEncoder := json.NewEncoder(w)
	messages := s.Chat.SubscribeThread(threadID)
	defer s.Chat.UnsubscribeThread(threadID, messages)

	for {
		select {
		case message := <-messages:
			fmt.Fprintf(w, "data: ")
			msgEncoder.Encode(message)
			fmt.Fprintf(w, "\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			log.Printf("Client disconnect from thread %d\n", threadID)
			return

		case <-time.After(8 * time.Hour):
			log.Println("WARNING: canceling thread subscription after 8 hours")
			return
		}
	}
}
