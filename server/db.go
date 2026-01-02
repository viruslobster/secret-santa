package server

import (
	"encoding/hex"
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
)

type Store interface {
	PersistChat(chat ChatMessage, thread ThreadId) error
	LoadThread(thread ThreadId) ([]ChatMessage, error)

	PutCache(key string, value []byte) error
	GetCache(key string) ([]byte, error)

	CreateUser(user *User) error
	LoadUser(userId UserId) (*User, error)

	CreateServer(server *SantaServer) error
	LoadServer(serverId ServerId) (*SantaServer, error)
}

type InMemoryStore struct {
	chats_by_thread map[ThreadId][]ChatMessage
	cache           map[string][]byte
	users           map[string]*User        // key is user ID (hex encoded)
	servers         map[string]*SantaServer // key is server ID (hex encoded)
}

func NewInMemoryStore() InMemoryStore {
	return InMemoryStore{
		chats_by_thread: make(map[ThreadId][]ChatMessage, 1),
		cache:           make(map[string][]byte),
		users:           make(map[string]*User),
		servers:         make(map[string]*SantaServer),
	}
}

func (s *InMemoryStore) PersistChat(chat ChatMessage, thread ThreadId) error {
	chats, ok := s.chats_by_thread[thread]
	if !ok {
		chats = make([]ChatMessage, 0)
	}
	chats = append(chats, chat)
	s.chats_by_thread[thread] = chats
	return nil
}

func (s *InMemoryStore) LoadThread(thread ThreadId) ([]ChatMessage, error) {
	chats, ok := s.chats_by_thread[thread]
	if !ok {
		return nil, fmt.Errorf("Thread %d not found", thread)
	}
	return chats, nil
}

func (s *InMemoryStore) PutCache(key string, value []byte) error {
	s.cache[key] = value
	return nil
}

func (s *InMemoryStore) GetCache(key string) ([]byte, error) {
	value, ok := s.cache[key]
	if !ok {
		return nil, fmt.Errorf("key '%s' not found in cache", key)
	}
	return value, nil
}

func (s *InMemoryStore) CreateUser(user *User) error {
	if len(user.Credentials) == 0 {
		return fmt.Errorf("user must have at least one credential")
	}
	// Store user by user ID
	key := fmt.Sprintf("%x", user.Id)
	_, contains := s.users[key]
	if contains {
		return fmt.Errorf("create user: id '%s' taken", key)
	}
	s.users[key] = user
	return nil
}

func (s *InMemoryStore) LoadUser(userID UserId) (*User, error) {
	key := fmt.Sprintf("%x", userID)
	user, ok := s.users[key]
	if !ok {
		return nil, fmt.Errorf("load user: id '%s' does not exist", key)
	}
	return user, nil
}

func (s *InMemoryStore) CreateServer(server *SantaServer) error {
	key := fmt.Sprintf("%x", server.Id)
	_, contains := s.servers[key]
	if contains {
		return fmt.Errorf("create server: id '%s' taken", key)
	}
	s.servers[key] = server
	return nil
}

func (s *InMemoryStore) LoadServer(serverId ServerId) (*SantaServer, error) {
	key := fmt.Sprintf("%x", serverId)
	server, ok := s.servers[key]
	if !ok {
		return nil, fmt.Errorf("load server: id '%s' does not exist", key)
	}
	return server, nil
}

type SantaServer struct {
	Id    ServerId
	Admin UserId

	// UserByName["foo"] = 123 => UserId 123 has joined this server as "foo"
	// "foo" must be in Names but not all values in Names are guaranteed to be
	// in UserByName until all members have joined
	UserByName map[string]UserId

	// Names available for users to claim on this server
	Names       []string
	GroupThread ThreadId

	// Pairing[i] = j => Users[i] is giving a gift to Users[j]
	Pairing []int

	// UserThreads[i] is the thread were Users[i] is giving a gift to Pairing[i]
	UserThreads []int
}

// User implements the webauthn.User interface
type User struct {
	Id          UserId
	Credentials []webauthn.Credential
}

func (u *User) WebAuthnID() []byte {
	return u.Id[:]
}

func (u *User) WebAuthnName() string {
	return fmt.Sprintf("user_%x", u.Id[:4])
}

func (u *User) WebAuthnDisplayName() string {
	return "Anonymous User"
}

func (u *User) WebAuthnIcon() string {
	return ""
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

type ThreadId uint64
type UserId [32]byte
type ServerId [32]byte

func parseUserId(userId string) (UserId, error) {
	userIdSlice, err := hex.DecodeString(userId)
	if err != nil {
		return UserId{}, fmt.Errorf("failed to decode hex: %w", err)
	}
	if len(userIdSlice) != 32 {
		return UserId{}, fmt.Errorf("invalid length: %d, expected 32", len(userIdSlice))
	}
	return *(*UserId)(userIdSlice), nil
}
