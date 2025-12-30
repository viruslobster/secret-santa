package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Server struct {
	Chat   *ChatClient
	Secret []byte
}

func (s *Server) Start(port string) error {
	http.HandleFunc("/api/chat/publish", s.chatHandler)
	http.HandleFunc("/api/thread", s.threadHandler)
	http.HandleFunc("/api/thread/subscribe", s.subscribeThreadHandler)
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("./dist"))))
	http.HandleFunc("/server", s.serverHandler)
	http.HandleFunc("/server/", s.serverHandler)
	http.HandleFunc("/", s.indexHandler)

	log.Printf("Starting server on port %s...", port)
	log.Printf("Serving files from current directory")
	return http.ListenAndServe(":"+port, nil)
}

// TODO: right now `login` always creates a JWT for the username you provide no matter what
// there is no actual authentication going on here
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": request.Username,
	})
	tokenString, err := token.SignedString(s.Secret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenString,
		HttpOnly: true,
		Secure:   false, // Only send over HTTPS, TODO: set true
		Path:     "/",
		MaxAge:   3600, // 1 hour
	})
	log.Printf("%s successful login\n", request.Username)
	http.Redirect(w, r, "/server", http.StatusFound)
}

// authenticatedUser returns true if a user is signed in. Otherwise it returns False and
// redirects to the login page
func (s *Server) authenticatedUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return "", false
	}
	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.Secret, nil
	})
	if err != nil || !token.Valid {
		log.Printf("WARN: recieved token cannot be validated err: %s, valid: %t\n", err, token.Valid)
		http.Redirect(w, r, "/", http.StatusFound)
		return "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Println("ERROR: token has unexpected claims map")
		return "", false
	}
	username, ok := claims["username"].(string)
	if !ok {
		log.Println("ERROR: token claims map missing string username")
		return "", false
	}
	log.Printf("Authenticated user: %s", username)
	return username, true
}

func (s *Server) serverHandler(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticatedUser(w, r)
	if !ok {
		return
	}
	tmpl := template.Must(template.ParseFiles("./html/server.html"))
	tmpl.Execute(w, map[string]string{
		"Username": username,
	})
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		s.login(w, r)
		return
	}
	http.ServeFile(w, r, "./html/index.html")
}

func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, ok := s.authenticatedUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Message string `json:"message"`
		user    string
		Thread  uint64 `json:"thread"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "While Deserialize PublishChatRequest", http.StatusBadRequest)
		return
	}
	req.user = user
	chat := ChatMessage{Message: req.Message, User: req.user, Timestamp: time.Now().Unix()}
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
	_, ok := s.authenticatedUser(w, r)
	if !ok {
		return
	}
	// TODO: also check that user has access to requested thread

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

func (s *Server) subscribeThreadHandler(w http.ResponseWriter, r *http.Request) {
	_, ok := s.authenticatedUser(w, r)
	if !ok {
		return
	}
	// TODO: also check that user has access to requested thread
	//
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
