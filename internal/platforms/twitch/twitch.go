package twitch

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"
)

// Platform implements router.Platform for Twitch IRC chat
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	conn           net.Conn
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Twitch configuration
type Config struct {
	Token   string // OAuth token (oauth:xxx)
	Channel string // Channel name to join
	BotName string // Bot username
}

// New creates a new Twitch platform
func New(cfg Config) (*Platform, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("Twitch token is required")
	}
	if cfg.Channel == "" {
		return nil, fmt.Errorf("Twitch channel is required")
	}
	if cfg.BotName == "" {
		return nil, fmt.Errorf("Twitch bot name is required")
	}

	return &Platform{
		config: cfg,
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "twitch"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start connects to Twitch IRC and begins listening
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	if err := p.connect(); err != nil {
		return fmt.Errorf("failed to connect to Twitch: %w", err)
	}

	sentryutil.Go("twitch readLoop", p.readLoop)

	log.Printf("[Twitch] Connected to channel #%s as %s", p.config.Channel, p.config.BotName)
	return nil
}

// Stop shuts down the Twitch connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// Send sends a message to the Twitch channel
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" || p.conn == nil {
		return nil
	}

	// Twitch IRC has a 500 char limit per message
	text := resp.Text
	if len(text) > 490 {
		text = text[:490] + "..."
	}

	msg := fmt.Sprintf("PRIVMSG #%s :%s\r\n", channelID, text)
	_, err := p.conn.Write([]byte(msg))
	return err
}

// connect establishes the IRC connection
func (p *Platform) connect() error {
	conn, err := net.DialTimeout("tcp", "irc.chat.twitch.tv:6667", 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	p.conn = conn

	// Authenticate
	token := p.config.Token
	if !strings.HasPrefix(token, "oauth:") {
		token = "oauth:" + token
	}

	fmt.Fprintf(conn, "PASS %s\r\n", token)
	fmt.Fprintf(conn, "NICK %s\r\n", p.config.BotName)
	fmt.Fprintf(conn, "JOIN #%s\r\n", p.config.Channel)

	return nil
}

// readLoop processes incoming IRC messages
func (p *Platform) readLoop() {
	scanner := bufio.NewScanner(p.conn)

	for scanner.Scan() {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		// Respond to PING to keep connection alive
		if strings.HasPrefix(line, "PING") {
			fmt.Fprintf(p.conn, "PONG %s\r\n", strings.TrimPrefix(line, "PING "))
			continue
		}

		// Parse PRIVMSG
		if strings.Contains(line, "PRIVMSG") {
			p.handlePrivMsg(line)
		}
	}

	if err := scanner.Err(); err != nil {
		if p.ctx.Err() != nil {
			return
		}
		log.Printf("[Twitch] Read error: %v, reconnecting...", err)
		time.Sleep(5 * time.Second)
		if err := p.connect(); err != nil {
			log.Printf("[Twitch] Reconnect failed: %v", err)
			return
		}
		sentryutil.Go("twitch readLoop", p.readLoop)
	}
}

// handlePrivMsg parses and processes a PRIVMSG IRC line
func (p *Platform) handlePrivMsg(line string) {
	// Format: :user!user@user.tmi.twitch.tv PRIVMSG #channel :message
	parts := strings.SplitN(line, " PRIVMSG ", 2)
	if len(parts) != 2 {
		return
	}

	// Extract username
	username := ""
	if strings.HasPrefix(parts[0], ":") {
		userPart := parts[0][1:]
		if idx := strings.Index(userPart, "!"); idx > 0 {
			username = userPart[:idx]
		}
	}

	// Ignore own messages
	if strings.EqualFold(username, p.config.BotName) {
		return
	}

	// Extract channel and message
	msgParts := strings.SplitN(parts[1], " :", 2)
	if len(msgParts) != 2 {
		return
	}

	channel := strings.TrimPrefix(msgParts[0], "#")
	text := msgParts[1]

	if p.messageHandler != nil {
		p.messageHandler(router.Message{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			Platform:  "twitch",
			ChannelID: channel,
			UserID:    username,
			Username:  username,
			Text:      text,
			Metadata: map[string]string{
				"channel": channel,
			},
		})
	}
}
