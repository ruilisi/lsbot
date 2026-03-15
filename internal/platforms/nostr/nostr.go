package nostr

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ruilisi/lsbot/internal/router"
)

// Platform implements router.Platform for NOSTR protocol
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	conns          []*websocket.Conn
	connsMu        sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds NOSTR configuration
type Config struct {
	PrivateKey string // NOSTR private key (hex or nsec)
	Relays     string // Comma-separated relay URLs
}

// New creates a new NOSTR platform
func New(cfg Config) (*Platform, error) {
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("NOSTR private key is required")
	}
	if cfg.Relays == "" {
		return nil, fmt.Errorf("at least one NOSTR relay is required")
	}

	return &Platform{
		config: cfg,
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "nostr"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start connects to NOSTR relays and subscribes to DM events
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	relays := strings.Split(p.config.Relays, ",")
	for _, relay := range relays {
		relay = strings.TrimSpace(relay)
		if relay == "" {
			continue
		}
		go p.connectRelay(relay)
	}

	log.Printf("[NOSTR] Connecting to relays: %s", p.config.Relays)
	return nil
}

// Stop shuts down all NOSTR connections
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.connsMu.Lock()
	defer p.connsMu.Unlock()
	for _, conn := range p.conns {
		conn.Close()
	}
	p.conns = nil
	return nil
}

// Send publishes a DM event to all connected relays
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	// Create a kind 4 (DM) event
	event := nostrEvent{
		Kind:      4,
		Content:   resp.Text,
		Tags:      [][]string{{"p", channelID}},
		CreatedAt: time.Now().Unix(),
	}

	eventJSON, err := json.Marshal([]any{"EVENT", event})
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	p.connsMu.Lock()
	defer p.connsMu.Unlock()

	var lastErr error
	for _, conn := range p.conns {
		if err := conn.WriteMessage(websocket.TextMessage, eventJSON); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// connectRelay connects to a single NOSTR relay
func (p *Platform) connectRelay(relayURL string) {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		conn, _, err := websocket.DefaultDialer.Dial(relayURL, nil)
		if err != nil {
			log.Printf("[NOSTR] Failed to connect to %s: %v", relayURL, err)
			time.Sleep(5 * time.Second)
			continue
		}

		p.connsMu.Lock()
		p.conns = append(p.conns, conn)
		p.connsMu.Unlock()

		log.Printf("[NOSTR] Connected to relay: %s", relayURL)

		// Subscribe to DM events (kind 4)
		sub := []any{"REQ", "lingti-dm", map[string]any{
			"kinds": []int{4},
			"since": time.Now().Unix(),
		}}
		subJSON, _ := json.Marshal(sub)
		conn.WriteMessage(websocket.TextMessage, subJSON)

		// Read loop
		p.readRelay(conn, relayURL)

		// Remove connection from list
		p.connsMu.Lock()
		for i, c := range p.conns {
			if c == conn {
				p.conns = append(p.conns[:i], p.conns[i+1:]...)
				break
			}
		}
		p.connsMu.Unlock()

		if p.ctx.Err() != nil {
			return
		}

		log.Printf("[NOSTR] Disconnected from %s, reconnecting...", relayURL)
		time.Sleep(5 * time.Second)
	}
}

// readRelay reads events from a relay connection
func (p *Platform) readRelay(conn *websocket.Conn, relayURL string) {
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}
			log.Printf("[NOSTR] Read error from %s: %v", relayURL, err)
			return
		}

		// Parse NOSTR message: ["EVENT", "sub_id", {...event...}]
		var envelope []json.RawMessage
		if err := json.Unmarshal(msg, &envelope); err != nil {
			continue
		}

		if len(envelope) < 3 {
			continue
		}

		var msgType string
		if err := json.Unmarshal(envelope[0], &msgType); err != nil {
			continue
		}

		if msgType != "EVENT" {
			continue
		}

		var event nostrEvent
		if err := json.Unmarshal(envelope[2], &event); err != nil {
			continue
		}

		// Process DM events (kind 4)
		if event.Kind == 4 && event.Content != "" {
			if p.messageHandler != nil {
				p.messageHandler(router.Message{
					ID:        event.ID,
					Platform:  "nostr",
					ChannelID: event.Pubkey,
					UserID:    event.Pubkey,
					Username:  event.Pubkey[:16],
					Text:      event.Content,
					Metadata: map[string]string{
						"relay":  relayURL,
						"pubkey": event.Pubkey,
					},
				})
			}
		}
	}
}

// nostrEvent represents a NOSTR event
type nostrEvent struct {
	ID        string     `json:"id,omitempty"`
	Pubkey    string     `json:"pubkey,omitempty"`
	Kind      int        `json:"kind"`
	Content   string     `json:"content"`
	Tags      [][]string `json:"tags"`
	CreatedAt int64      `json:"created_at"`
	Sig       string     `json:"sig,omitempty"`
}
