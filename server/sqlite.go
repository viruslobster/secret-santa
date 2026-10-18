package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteStore struct {
	Db      *sql.DB
	cacheMu sync.RWMutex
	cache   map[string][]byte
}

var _ Store = (*SQLiteStore)(nil)

func NewSQLiteStore() (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", "./db.sqlite")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	// Wait up to five seconds for a database lock instead of failing immediately.
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStore{Db: db, cache: make(map[string][]byte)}, nil
}

func (s *SQLiteStore) Close() error {
	return s.Db.Close()
}

func (s *SQLiteStore) PersistChat(chat ChatMessage, thread ThreadId) error {
	_, err := s.Db.Exec("INSERT INTO chats (thread, message, user_id, timestamp) VALUES (?, ?, ?, ?)",
		strconv.FormatUint(uint64(thread), 10), chat.Message, chat.User, chat.Timestamp)
	return err
}

func (s *SQLiteStore) LoadThread(thread ThreadId) ([]ChatMessage, error) {
	rows, err := s.Db.Query("SELECT message, user_id, timestamp FROM chats WHERE thread = ? ORDER BY id",
		strconv.FormatUint(uint64(thread), 10))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chats := make([]ChatMessage, 0, 256)
	for rows.Next() {
		var chat ChatMessage
		if err := rows.Scan(&chat.Message, &chat.User, &chat.Timestamp); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(chats) == 0 {
		return nil, fmt.Errorf("thread %d not found", thread)
	}
	return chats, nil
}

func (s *SQLiteStore) PutCache(key string, value []byte) error {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache[key] = value
	return nil
}

func (s *SQLiteStore) GetCache(key string) ([]byte, error) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	value, ok := s.cache[key]
	if !ok {
		return nil, fmt.Errorf("key '%s' not found in cache", key)
	}
	return value, nil
}

func (s *SQLiteStore) CreateUser(user *User) error {
	data, err := json.Marshal(user.Credentials)
	if err != nil {
		return err
	}
	result, err := s.Db.Exec("INSERT INTO users (credentials) VALUES (?)", string(data))
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	user.Id = UserId(id)
	return err
}

func (s *SQLiteStore) UpdateUser(user *User) error {
	data, err := json.Marshal(user.Credentials)
	if err != nil {
		return err
	}
	result, err := s.Db.Exec("UPDATE users SET credentials = ? WHERE id = ?", string(data), user.Id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("user %d not found", user.Id)
	}
	return nil
}

func (s *SQLiteStore) LoadUser(id UserId) (*User, error) {
	var data []byte
	if err := s.Db.QueryRow("SELECT credentials FROM users WHERE id = ?", id).Scan(&data); err != nil {
		return nil, err
	}
	user := User{Id: id}
	if err := json.Unmarshal(data, &user.Credentials); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *SQLiteStore) CreateServer(server *SantaServer) error {
	for _, name := range server.Names {
		if name == "" || strings.Contains(name, ",") {
			return fmt.Errorf("names must be nonempty and cannot contain commas")
		}
	}
	tx, err := s.Db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	pairings, err := json.Marshal(server.Pairing)
	if err != nil {
		return err
	}
	userThreads, err := json.Marshal(server.UserThreads)
	if err != nil {
		return err
	}
	result, err := tx.Exec("INSERT INTO servers (admin_id, group_thread, names, pairings, user_threads) VALUES (?, ?, ?, ?, ?)",
		server.Admin, strconv.FormatUint(uint64(server.GroupThread), 10), strings.Join(server.Names, ","), string(pairings), string(userThreads))
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	for name, user := range server.UserByName {
		if _, err := tx.Exec("INSERT INTO server_members (server_id, name, user_id) VALUES (?, ?, ?)", id, name, user); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	server.Id = ServerId(id)
	return nil
}

func (s *SQLiteStore) LoadServer(id ServerId) (*SantaServer, error) {
	tx, err := s.Db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	server := SantaServer{Id: id, UserByName: make(map[string]UserId)}
	var names string
	var pairings, userThreads []byte
	if err := tx.QueryRow("SELECT admin_id, group_thread, names, pairings, user_threads FROM servers WHERE id = ?", id).Scan(&server.Admin, &server.GroupThread, &names, &pairings, &userThreads); err != nil {
		return nil, err
	}
	if names != "" {
		server.Names = strings.Split(names, ",")
	}
	if err := loadServerRows(tx, "SELECT name, user_id FROM server_members WHERE server_id = ?", id, func(rows *sql.Rows) error {
		var name string
		var user UserId
		if err := rows.Scan(&name, &user); err != nil {
			return err
		}
		server.UserByName[name] = user
		return nil
	}); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(pairings, &server.Pairing); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(userThreads, &server.UserThreads); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &server, nil
}

func loadServerRows(tx *sql.Tx, query string, id ServerId, scan func(*sql.Rows) error) error {
	rows, err := tx.Query(query, id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}
