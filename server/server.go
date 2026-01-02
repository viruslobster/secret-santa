package server

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/golang-jwt/jwt/v5"
)

type Server struct {
	Chat     *ChatClient
	Secret   []byte
	WebAuthn *webauthn.WebAuthn
}

func (s *Server) Start(port string) error {
	http.HandleFunc("/api/chat/publish", s.chatHandler)
	http.HandleFunc("/api/thread", s.threadHandler)
	http.HandleFunc("/api/thread/subscribe", s.subscribeThreadHandler)
	http.HandleFunc("/api/register/begin", s.registerBeginHandler)
	http.HandleFunc("/api/register/finish", s.registerFinishHandler)
	http.HandleFunc("/api/login/begin", s.loginBeginHandler)
	http.HandleFunc("/api/login/finish", s.loginFinishHandler)
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("./dist"))))
	http.HandleFunc("/server", s.serverHandler)
	http.HandleFunc("/server/", s.serverHandler)
	http.HandleFunc("/", s.indexHandler)

	log.Printf("Starting server on port %s...", port)
	log.Printf("Serving files from current directory")
	return http.ListenAndServe(":"+port, nil)
}

// approveLogin is called after a user has already been verified and we want to generate a token
// that keeps the user logged in
func (s *Server) approveLogin(user *User, w http.ResponseWriter, r *http.Request) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": fmt.Sprintf("%x", user.ID),
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
	http.Redirect(w, r, "/server", http.StatusFound)
}

// authenticatedUser returns true if a user is signed in. Otherwise it returns False and
// redirects to the login page
func (s *Server) authenticatedUser(w http.ResponseWriter, r *http.Request) (*User, bool) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return nil, false
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
		return nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		log.Println("ERROR: token has unexpected claims map")
		http.Redirect(w, r, "/", http.StatusFound)
		return nil, false
	}
	userIdAny, ok := claims["userId"]
	if !ok {
		log.Println("ERROR: token claims map missing string userId")
		http.Redirect(w, r, "/", http.StatusFound)
		return nil, false
	}
	userIdStr, ok := userIdAny.(string)
	if !ok {
		log.Printf("ERROR: userId is not a string, type: %T", userIdAny)
		http.Redirect(w, r, "/", http.StatusFound)
		return nil, false
	}
	userId, err := parseUserId(userIdStr)
	if err != nil {
		log.Printf("ERROR: invalid userId: %s", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return nil, false
	}
	user, err := s.Chat.Store.LoadUser(userId)
	if err != nil {
		log.Printf("ERROR: load user: %s", err)
		http.Redirect(w, r, "/", http.StatusFound)
		return nil, false
	}
	log.Printf("Authenticated user: %s", user.WebAuthnName())
	return user, true
}

func (s *Server) serverHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := s.authenticatedUser(w, r)
	if !ok {
		return
	}
	tmpl := template.Must(template.ParseFiles("./html/server.html"))
	tmpl.Execute(w, map[string]string{
		"Username": user.WebAuthnName(),
	})
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Redirect(w, r, "/", http.StatusFound)
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
	req.user = user.WebAuthnName()
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

func (s *Server) registerBeginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate a random user ID for username-less registration
	var userID UserId
	if _, err := rand.Read(userID[:]); err != nil {
		http.Error(w, "Failed to generate user ID", http.StatusInternalServerError)
		return
	}
	user := &User{ID: userID}

	options, session, err := s.WebAuthn.BeginRegistration(
		user,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			ResidentKey:        protocol.ResidentKeyRequirementRequired,
			UserVerification:   protocol.VerificationDiscouraged,
		}),
	)
	if err != nil {
		log.Printf("Error beginning registration: %v", err)
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		return
	}
	sessionData, err := json.Marshal(session)
	if err != nil {
		http.Error(w, "Failed to marshal session", http.StatusInternalServerError)
		return
	}

	// Cached so it can be retrieved in `registerFinishHandler`
	cacheKey := fmt.Sprintf("registration_session_%x", userID)
	if err := s.Chat.Store.PutCache(cacheKey, sessionData); err != nil {
		http.Error(w, "Failed to store session", http.StatusInternalServerError)
		return
	}

	var response struct {
		Options *protocol.CredentialCreation `json:"options"`
		UserID  string                       `json:"userId"`
	}
	response.Options = options
	response.UserID = fmt.Sprintf("%x", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) registerFinishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request struct {
		UserId   string          `json:"userId"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	key := "registration_session_" + request.UserId
	sessionData, err := s.Chat.Store.GetCache(key)
	if err != nil {
		log.Printf("Failed to retrieve session: %v", err)
		http.Error(w, "Session not found", http.StatusBadRequest)
		return
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		http.Error(w, "Invalid session data", http.StatusInternalServerError)
		return
	}
	userId, err := parseUserId(request.UserId)
	if err != nil {
		log.Printf("ERROR: Invalid user ID: %s", err)
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	reader := io.NopCloser(bytes.NewReader(request.Response))
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(reader)
	if err != nil {
		log.Printf("Failed to parse credential: %v", err)
		http.Error(w, "Invalid credential response", http.StatusBadRequest)
		return
	}

	// Actual verification
	user := &User{ID: userId}
	credential, err := s.WebAuthn.CreateCredential(user, session, parsedResponse)
	if err != nil {
		log.Printf("Failed to create credential: %v", err)
		http.Error(w, "Failed to verify credential", http.StatusBadRequest)
		return
	}
	user.Credentials = append(user.Credentials, *credential)
	s.Chat.Store.RegisterUser(user)
	log.Printf("Successfully registered user with credential ID: '%x', user ID: '%x'", credential.ID, user.ID)
	s.approveLogin(user, w, r)
}

func (s *Server) loginBeginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	credential, session, err := s.WebAuthn.BeginDiscoverableLogin()
	if err != nil {
		log.Printf("ERROR: begin login: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sessionData, err := json.Marshal(session)
	if err != nil {
		log.Printf("ERROR: marshal session: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// At this point we don't know the id of the user trying to login,
	// but we need some id to use to store session data
	var sessionId [32]byte
	if _, err := rand.Read(sessionId[:]); err != nil {
		http.Error(w, "Failed to generate sesion ID", http.StatusInternalServerError)
		return
	}

	key := fmt.Sprintf("login_session_%x", sessionId)
	err = s.Chat.Store.PutCache(key, sessionData)
	log.Printf("storing key: %s", key)
	if err != nil {
		log.Printf("ERROR: put cache: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response struct {
		SessionId  string                        `json:"sessionId"`
		Credential *protocol.CredentialAssertion `json:"credential"`
	}
	response.SessionId = fmt.Sprintf("%x", sessionId)
	response.Credential = credential
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) loginFinishHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request struct {
		SessionId string          `json:"sessionId"`
		Response  json.RawMessage `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("ERRER: decode request: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reader := io.NopCloser(bytes.NewReader(request.Response))
	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(reader)
	if err != nil {
		log.Printf("Failed to parse credential: %v", err)
		http.Error(w, "Invalid credential response", http.StatusBadRequest)
		return
	}
	if request.SessionId == "" {
		log.Printf("/api/login/finish recieved no SessionId")
		http.Error(w, "no SessionID", http.StatusBadRequest)
		return
	}
	key := fmt.Sprintf("login_session_%s", request.SessionId)
	sessionData, err := s.Chat.Store.GetCache(key)
	if err != nil {
		log.Printf("ERROR: get login session: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionData, &session); err != nil {
		log.Printf("ERROR: unmarshal login session: %s", err)
		http.Error(w, "Invalid session data", http.StatusInternalServerError)
		return
	}
	discoverUser := func(_credentialId, userId []byte) (webauthn.User, error) {
		n := len(userId)
		if n != 32 {
			return nil, fmt.Errorf("Unexpected userId len: %d", n)
		}
		sized := (*UserId)(userId)
		user, err := s.Chat.Store.LoadUser(*sized)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	handle := webauthn.DiscoverableUserHandler(discoverUser)
	userInterface, credential, err := s.WebAuthn.ValidatePasskeyLogin(handle, session, parsedResponse)
	if err != nil {
		log.Printf("ERROR: validate login: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = credential
	user := userInterface.(*User)
	s.approveLogin(user, w, r)
	log.Printf("Successful login")
}
