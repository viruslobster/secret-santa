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

	RegisterUser(user *User) error
	LoadUser(userId UserId) (*User, error)
}

type InMemoryStore struct {
	chats_by_thread map[ThreadId][]ChatMessage
	cache           map[string][]byte
	users           map[string]*User // key is credential ID (hex encoded)
}

func NewInMemoryStore() InMemoryStore {
	return InMemoryStore{
		chats_by_thread: make(map[ThreadId][]ChatMessage, 1),
		cache:           make(map[string][]byte),
		users:           make(map[string]*User),
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

func (s *InMemoryStore) RegisterUser(user *User) error {
	if len(user.Credentials) == 0 {
		return fmt.Errorf("user must have at least one credential")
	}
	// Store user by credential ID
	key := fmt.Sprintf("%x", user.ID)
	_, contains := s.users[key]
	if contains {
		return fmt.Errorf("register user: id '%s' taken", key)
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

// User implements the webauthn.User interface
type User struct {
	ID          UserId
	Credentials []webauthn.Credential
}

func (u *User) WebAuthnID() []byte {
	return u.ID[:]
}

func (u *User) WebAuthnName() string {
	return fmt.Sprintf("user_%x", u.ID[:4])
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
