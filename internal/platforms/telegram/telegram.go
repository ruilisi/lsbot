package telegram

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"
	"github.com/ruilisi/lsbot/internal/transcription"
)

// Platform implements router.Platform for Telegram
type Platform struct {
	bot            *tgbotapi.BotAPI
	messageHandler func(msg router.Message)
	ctx            context.Context
	cancel         context.CancelFunc
	transcriber    *transcription.Transcriber // optional; nil = no voice transcription
}

// Config holds Telegram configuration
type Config struct {
	Token           string // Bot token from @BotFather
	Debug           bool   // Enable debug logging
	WhisperAPIKey   string // OpenAI API key for voice transcription (optional)
	WhisperBaseURL  string // Custom Whisper-compatible endpoint (optional)
}

// New creates a new Telegram platform
func New(cfg Config) (*Platform, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("Telegram bot token is required")
	}

	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Telegram bot: %w", err)
	}

	bot.Debug = cfg.Debug

	p := &Platform{bot: bot}
	if cfg.WhisperAPIKey != "" {
		p.transcriber = transcription.New(cfg.WhisperAPIKey, cfg.WhisperBaseURL)
	}
	return p, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "telegram"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for Telegram updates
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := p.bot.GetUpdatesChan(u)

	sentryutil.Go("telegram handleUpdates", func() { p.handleUpdates(updates) })

	log.Printf("[Telegram] Connected as bot: @%s", p.bot.Self.UserName)
	return nil
}

// Stop shuts down the Telegram connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.bot.StopReceivingUpdates()
	return nil
}

// Send sends a message to a Telegram chat
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	chatID, err := parseChatID(channelID)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewMessage(chatID, resp.Text)

	// Enable Markdown formatting
	msg.ParseMode = "Markdown"

	// Reply to specific message if ThreadID is set
	if resp.ThreadID != "" {
		if msgID, err := parseMessageID(resp.ThreadID); err == nil {
			msg.ReplyToMessageID = msgID
		}
	}

	_, err = p.bot.Send(msg)
	return err
}

// handleUpdates processes incoming Telegram updates
func (p *Platform) handleUpdates(updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case <-p.ctx.Done():
			return
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			// Skip messages from bots
			if update.Message.From.IsBot {
				continue
			}

			// Check if we should respond
			if !p.shouldRespond(update.Message) {
				continue
			}

			text := p.cleanMention(update.Message.Text)

			// Handle voice messages via Whisper transcription
			if text == "" && update.Message.Voice != nil && p.transcriber != nil {
				if t, err := p.transcribeVoice(update.Message.Voice.FileID); err == nil {
					text = "[Voice] " + t
				}
			}

			// Handle audio files (music, audio attachments)
			if text == "" && update.Message.Audio != nil && p.transcriber != nil {
				if t, err := p.transcribeVoice(update.Message.Audio.FileID); err == nil {
					text = "[Audio] " + t
				}
			}

			// Handle caption for media messages (photos, videos, documents with caption)
			if text == "" && update.Message.Caption != "" {
				text = p.cleanMention(update.Message.Caption)
			}

			if text == "" {
				continue
			}

			if p.messageHandler != nil {
				threadID := ""
				if update.Message.ReplyToMessage != nil {
					threadID = fmt.Sprintf("%d", update.Message.ReplyToMessage.MessageID)
				}

				p.messageHandler(router.Message{
					ID:        fmt.Sprintf("%d", update.Message.MessageID),
					Platform:  "telegram",
					ChannelID: fmt.Sprintf("%d", update.Message.Chat.ID),
					UserID:    fmt.Sprintf("%d", update.Message.From.ID),
					Username:  getUsername(update.Message.From),
					Text:      text,
					ThreadID:  threadID,
					Metadata: map[string]string{
						"chat_type": update.Message.Chat.Type,
					},
				})
			}
		}
	}
}

// transcribeVoice downloads a Telegram file by fileID and transcribes it.
func (p *Platform) transcribeVoice(fileID string) (string, error) {
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	tgFile, err := p.bot.GetFile(fileConfig)
	if err != nil {
		return "", fmt.Errorf("telegram: get file: %w", err)
	}

	url := tgFile.Link(p.bot.Token)

	// Download to a temp file
	tmpDir := os.TempDir()
	ext := filepath.Ext(tgFile.FilePath)
	if ext == "" {
		ext = ".ogg"
	}
	tmpFile, err := os.CreateTemp(tmpDir, "tg-voice-*"+ext)
	if err != nil {
		return "", fmt.Errorf("telegram: temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("telegram: download voice: %w", err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return "", fmt.Errorf("telegram: write voice: %w", err)
	}
	tmpFile.Close()

	return p.transcriber.TranscribeFile(tmpFile.Name())
}

// shouldRespond checks if the bot should respond to this message
func (p *Platform) shouldRespond(msg *tgbotapi.Message) bool {
	// Always respond in private chats
	if msg.Chat.IsPrivate() {
		return true
	}

	// In groups, only respond to mentions or replies to bot
	if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
		if strings.Contains(msg.Text, "@"+p.bot.Self.UserName) {
			return true
		}
		if msg.Caption != "" && strings.Contains(msg.Caption, "@"+p.bot.Self.UserName) {
			return true
		}
		if msg.ReplyToMessage != nil && msg.ReplyToMessage.From.ID == p.bot.Self.ID {
			return true
		}
		if msg.IsCommand() {
			return true
		}
		// Voice messages in groups only if they reply to bot
		if msg.Voice != nil && msg.ReplyToMessage != nil && msg.ReplyToMessage.From.ID == p.bot.Self.ID {
			return true
		}
		return false
	}

	return true
}

// cleanMention removes the bot mention from the message
func (p *Platform) cleanMention(text string) string {
	mention := "@" + p.bot.Self.UserName
	text = strings.ReplaceAll(text, mention, "")
	return strings.TrimSpace(text)
}

// getUsername returns a human-readable username
func getUsername(user *tgbotapi.User) string {
	if user.UserName != "" {
		return user.UserName
	}
	if user.FirstName != "" {
		name := user.FirstName
		if user.LastName != "" {
			name += " " + user.LastName
		}
		return name
	}
	return fmt.Sprintf("%d", user.ID)
}

// parseChatID parses a string chat ID to int64
func parseChatID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}

// parseMessageID parses a string message ID to int
func parseMessageID(s string) (int, error) {
	var id int
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}
