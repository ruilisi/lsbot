package history

import (
	"fmt"
	"time"
)

// SearchResult is one matching session returned by FTS search.
type SearchResult struct {
	ConvKey   string
	Platform  string
	Username  string
	UpdatedAt time.Time
	Snippet   string // highlighted snippet from FTS
	Messages  []Message
}

// EnsureFTS creates the FTS5 virtual table and its triggers if they do not exist.
// Safe to call on every startup – uses IF NOT EXISTS.
func (s *Store) EnsureFTS() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
	CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		role,
		content,
		content='messages',
		content_rowid='id'
	);

	-- Keep FTS in sync with the messages table.
	CREATE TRIGGER IF NOT EXISTS msg_ai AFTER INSERT ON messages BEGIN
		INSERT INTO messages_fts(rowid, role, content) VALUES (new.id, new.role, new.content);
	END;
	CREATE TRIGGER IF NOT EXISTS msg_ad AFTER DELETE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, role, content) VALUES('delete', old.id, old.role, old.content);
	END;
	CREATE TRIGGER IF NOT EXISTS msg_au AFTER UPDATE ON messages BEGIN
		INSERT INTO messages_fts(messages_fts, rowid, role, content) VALUES('delete', old.id, old.role, old.content);
		INSERT INTO messages_fts(rowid, role, content) VALUES (new.id, new.role, new.content);
	END;
	`)
	return err
}

// Search performs a full-text search across all saved conversations.
// Returns up to maxSessions sessions (default 3), each with up to msgLimit messages.
func (s *Store) Search(query string, maxSessions, msgLimit int) ([]SearchResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxSessions <= 0 {
		maxSessions = 3
	}
	if msgLimit <= 0 {
		msgLimit = 20
	}

	// Find top matching session IDs via FTS5
	rows, err := s.db.Query(`
		SELECT DISTINCT m.session_id, snippet(messages_fts, 1, '[', ']', '...', 20)
		FROM messages_fts
		JOIN messages m ON messages_fts.rowid = m.id
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, query, maxSessions)
	if err != nil {
		return nil, fmt.Errorf("history: fts query: %w", err)
	}
	defer rows.Close()

	type sessionSnippet struct {
		id      int64
		snippet string
	}
	var hits []sessionSnippet
	for rows.Next() {
		var h sessionSnippet
		if err := rows.Scan(&h.id, &h.snippet); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, h := range hits {
		// Fetch session metadata
		var r SearchResult
		var updatedAt int64
		err := s.db.QueryRow(`SELECT conv_key, platform, username, updated_at FROM sessions WHERE id=?`, h.id).
			Scan(&r.ConvKey, &r.Platform, &r.Username, &updatedAt)
		if err != nil {
			continue
		}
		r.UpdatedAt = time.Unix(updatedAt, 0)
		r.Snippet = h.snippet

		// Fetch recent messages for context
		msgRows, err := s.db.Query(`
			SELECT role, content, created_at FROM messages
			WHERE session_id=? ORDER BY id DESC LIMIT ?`, h.id, msgLimit)
		if err != nil {
			continue
		}
		var msgs []Message
		for msgRows.Next() {
			var m Message
			var ts int64
			if err := msgRows.Scan(&m.Role, &m.Content, &ts); err != nil {
				break
			}
			m.CreatedAt = time.Unix(ts, 0)
			msgs = append(msgs, m)
		}
		msgRows.Close()
		// Reverse to chronological order
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}
		r.Messages = msgs
		results = append(results, r)
	}
	return results, nil
}
