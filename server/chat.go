package server

import (
	"fmt"
	"log"
)

type Thread uint64

type Store interface {
	Publish(chat Chat, thread Thread) error
	Chats(thread Thread) ([]Chat, error)
}

type InMemoryStore struct {
	chats_by_thread map[Thread][]Chat
}

func NewInMemoryStore() InMemoryStore {
	return InMemoryStore{
		chats_by_thread: make(map[Thread][]Chat, 1),
	}
}

func (s *InMemoryStore) Publish(chat Chat, thread Thread) error {
	chats, ok := s.chats_by_thread[thread]
	if !ok {
		chats = make([]Chat, 0)
	}
	chats = append(chats, chat)
	log.Printf("thread %d, chats: %v", thread, chats)
	s.chats_by_thread[thread] = chats
	return nil
}

func (s *InMemoryStore) Chats(thread Thread) ([]Chat, error) {
	chats, ok := s.chats_by_thread[thread]
	if !ok {
		return nil, fmt.Errorf("Thread %d not found", thread)
	}
	return chats, nil
}

type Chat struct {
	Message   string `json:"message"`
	User      string `json:"user"`
	Timestamp int64  `json:"timestamp"`
}

type Client struct {
	Store Store
}

func (c *Client) Publish(chat Chat, thread Thread) error {
	err := c.Store.Publish(chat, thread)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Chats(thread Thread) ([]Chat, error) {
	chats, err := c.Store.Chats(thread)
	return chats, err
}
