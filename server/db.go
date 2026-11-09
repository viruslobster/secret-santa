package server

import (
	"fmt"
	"strconv"

	"github.com/go-webauthn/webauthn/webauthn"
)

type Store interface {
	PersistChat(chat ChatMessage, thread ThreadId) error
	LoadThread(thread ThreadId) ([]ChatMessage, error)

	PutCache(key string, value []byte) error
	GetCache(key string) ([]byte, error)

	CreateUser(user *User) error
	UpdateUser(user *User) error
	LoadUser(userId UserId) (*User, error)

	CreateServer(names []string, pairing []int) (ServerId, error)
	LoadServer(serverId ServerId) (*SantaServer, error)
}

type InMemoryStore struct {
	chats_by_thread map[ThreadId][]ChatMessage
	cache           map[string][]byte
	users           map[UserId]*User
	nextUserId      UserId
	servers         map[ServerId]*SantaServer
	nextServerId    ServerId
}

func NewInMemoryStore() InMemoryStore {
	return InMemoryStore{
		chats_by_thread: make(map[ThreadId][]ChatMessage, 1),
		cache:           make(map[string][]byte),
		users:           make(map[UserId]*User),
		servers:         make(map[ServerId]*SantaServer),
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
	s.nextUserId++
	user.Id = s.nextUserId
	s.users[user.Id] = user
	return nil
}

func (s *InMemoryStore) UpdateUser(user *User) error {
	if _, ok := s.users[user.Id]; !ok {
		return fmt.Errorf("user %d not found", user.Id)
	}
	s.users[user.Id] = user
	return nil
}

func (s *InMemoryStore) LoadUser(userID UserId) (*User, error) {
	user, ok := s.users[userID]
	if !ok {
		return nil, fmt.Errorf("load user: id '%d' does not exist", userID)
	}
	return user, nil
}

func (s *InMemoryStore) CreateServer(server *SantaServer) error {
	s.nextServerId++
	server.Id = s.nextServerId
	s.servers[server.Id] = server
	return nil
}

func (s *InMemoryStore) LoadServer(serverId ServerId) (*SantaServer, error) {
	server, ok := s.servers[serverId]
	if !ok {
		return nil, fmt.Errorf("load server: id '%d' does not exist", serverId)
	}
	return server, nil
}

type SantaServer struct {
	Id ServerId

	// UserByName["foo"] = 123 => UserId 123 has joined this server as "foo"
	// "foo" must be in Names but not all values in Names are guaranteed to be
	// in UserByName until all members have joined
	UserByName map[string]UserId

	// Names available for users to claim on this server
	Names []string

	// Pairing[i] = j => Users[i] is giving a gift to Users[j]
	Pairing []int
}

// User implements the webauthn.User interface
type User struct {
	Id          UserId
	Credentials []webauthn.Credential
}

func (u *User) WebAuthnID() []byte {
	return []byte(strconv.FormatInt(int64(u.Id), 10))
}

func (u *User) WebAuthnName() string {
	return fmt.Sprintf("user_%d", u.Id)
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
type UserId int64
type ServerId int64

func parseUserId(userId string) (UserId, error) {
	id, err := strconv.ParseInt(userId, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid user ID: %q", userId)
	}
	return UserId(id), nil
}
