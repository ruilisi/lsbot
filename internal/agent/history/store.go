// Package history provides SQLite-backed persistent conversation storage.
// Conversations are saved across restarts, enabling cross-session recall and search.
package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ruilisi/lsbot/internal/config"
	_ "modernc.org/sqlite"
)

// Message is a single turn in a conversation.
type Message struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

// Store persists conversations to SQLite.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

var (
	globalStore *Store
	initOnce   sync.Once
	initErr    error
)

// Global returns the process-wide Store, initialising it on first call.
func Global() (*Store, error) {
	initOnce.Do(func() {
		globalStore, initErr = open(dbPath())
	})
	return globalStore, initErr
}

func dbPath() string {
	return filepath.Join(config.HubDir(), "sessions.db")
}

func open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("history: mkdir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("history: open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS sessions (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		conv_key   TEXT    NOT NULL,
		platform   TEXT    NOT NULL DEFAULT '',
		username   TEXT    NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_conv_key ON sessions(conv_key);

	CREATE TABLE IF NOT EXISTS messages (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
		role       TEXT    NOT NULL,
		content    TEXT    NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);

	PRAGMA journal_mode=WAL;
	`)
	return err
}

// sessionID returns the row ID for convKey, creating it if needed.
func (s *Store) sessionID(convKey, platform, username string) (int64, error) {
	now := time.Now().Unix()
	res, err := s.db.Exec(`
		INSERT INTO sessions(conv_key, platform, username, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(conv_key) DO UPDATE SET updated_at=excluded.updated_at`,
		convKey, platform, username, now, now)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		// Conflict path — fetch existing
		err = s.db.QueryRow(`SELECT id FROM sessions WHERE conv_key=?`, convKey).Scan(&id)
	}
	return id, err
}

// Save appends messages to the persistent store for convKey.
func (s *Store) Save(convKey, platform, username string, msgs []Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sid, err := s.sessionID(convKey, platform, username)
	if err != nil {
		return fmt.Errorf("history: session: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.Prepare(`INSERT INTO messages(session_id, role, content, created_at) VALUES(?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, m := range msgs {
		ts := now
		if !m.CreatedAt.IsZero() {
			ts = m.CreatedAt.Unix()
		}
		if _, err := stmt.Exec(sid, m.Role, m.Content, ts); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Load returns up to limit recent messages for convKey.
func (s *Store) Load(convKey string, limit int) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.db.Query(`
		SELECT m.role, m.content, m.created_at
		FROM messages m
		JOIN sessions ses ON ses.id = m.session_id
		WHERE ses.conv_key = ?
		ORDER BY m.id DESC
		LIMIT ?`, convKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		var ts int64
		if err := rows.Scan(&m.Role, &m.Content, &ts); err != nil {
			return nil, err
		}
		m.CreatedAt = time.Unix(ts, 0)
		out = append(out, m)
	}
	// Reverse so oldest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, rows.Err()
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}
