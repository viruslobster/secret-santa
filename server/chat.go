package server

import (
	"fmt"
	"log"
	"slices"
	"sync"
	"time"
)

type ThreadId uint64

type Store interface {
	Persist(chat ChatMessage, thread ThreadId) error
	Load(thread ThreadId) ([]ChatMessage, error)
}

type InMemoryStore struct {
	chats_by_thread map[ThreadId][]ChatMessage
}

func NewInMemoryStore() InMemoryStore {
	return InMemoryStore{
		chats_by_thread: make(map[ThreadId][]ChatMessage, 1),
	}
}

func (s *InMemoryStore) Persist(chat ChatMessage, thread ThreadId) error {
	chats, ok := s.chats_by_thread[thread]
	if !ok {
		chats = make([]ChatMessage, 0)
	}
	chats = append(chats, chat)
	log.Printf("thread %d, chats: %v", thread, chats)
	s.chats_by_thread[thread] = chats
	return nil
}

func (s *InMemoryStore) Load(thread ThreadId) ([]ChatMessage, error) {
	chats, ok := s.chats_by_thread[thread]
	if !ok {
		return nil, fmt.Errorf("Thread %d not found", thread)
	}
	return chats, nil
}

type ChatMessage struct {
	Message   string `json:"message"`
	User      string `json:"user"`
	Timestamp int64  `json:"timestamp"`
}

type ChatClient struct {
	Store Store

	mutex                 sync.RWMutex
	subscriptionsByThread map[ThreadId][]chan ChatMessage
}

func (c *ChatClient) Publish(chat ChatMessage, thread ThreadId) error {
	err := c.Store.Persist(chat, thread)
	if err != nil {
		return err
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	channels, ok := c.subscriptionsByThread[thread]
	if !ok {
		return nil
	}
	// Must write to channel while lock is held. Otherwise channel may be closed
	for i, channel := range channels {
		select {
		case channel <- chat:
		case <-time.After(100 * time.Millisecond):
			log.Printf("ERROR: failed to send message to subscriber %d\n", i)
		}
	}
	return nil
}

func (c *ChatClient) Chats(thread ThreadId) ([]ChatMessage, error) {
	chats, err := c.Store.Load(thread)
	return chats, err
}

// SubscribeThread returns a new channel that you can recieve chat threads.
// When you stop recieving from the returned channel, you must call UnsubscribeThread.
func (c *ChatClient) SubscribeThread(thread ThreadId) chan ChatMessage {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.subscriptionsByThread == nil {
		c.subscriptionsByThread = make(map[ThreadId][]chan ChatMessage)
	}
	channels, ok := c.subscriptionsByThread[thread]
	if !ok {
		channels = make([]chan ChatMessage, 0, 1)
	}
	channel := make(chan ChatMessage, 1)
	channels = append(channels, channel)
	c.subscriptionsByThread[thread] = channels
	return channel
}

// UnsubscribeThread safely closes `channel`
func (c *ChatClient) UnsubscribeThread(thread ThreadId, channel chan ChatMessage) {
	// Throw away any remaining data so writers don't block
	go func() {
		for range channel {
		}
	}()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	close(channel)

	channels, ok := c.subscriptionsByThread[thread]
	if !ok {
		return
	}

	idx := slices.Index(channels, channel)
	if idx == -1 {
		return
	}

	// Swap remove
	last_idx := len(channels) - 1
	channels[idx] = channels[last_idx]
	channels = channels[:last_idx]
	c.subscriptionsByThread[thread] = channels
}
