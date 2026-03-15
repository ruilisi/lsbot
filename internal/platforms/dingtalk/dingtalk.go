package dingtalk

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
	"github.com/ruilisi/lsbot/internal/router"
)

// Platform implements router.Platform for DingTalk
type Platform struct {
	cli            *client.StreamClient
	messageHandler func(msg router.Message)
	webhooks       map[string]string // conversationID -> sessionWebhook
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds DingTalk configuration
type Config struct {
	ClientID     string // AppKey from DingTalk Developer Console
	ClientSecret string // AppSecret from DingTalk Developer Console
}

// New creates a new DingTalk platform
func New(cfg Config) (*Platform, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, fmt.Errorf("both ClientID (AppKey) and ClientSecret (AppSecret) are required")
	}

	p := &Platform{
		webhooks: make(map[string]string),
	}

	// Create stream client
	cli := client.NewStreamClient(
		client.WithAppCredential(client.NewAppCredentialConfig(cfg.ClientID, cfg.ClientSecret)),
	)

	// Register chatbot callback
	cli.RegisterChatBotCallbackRouter(p.onChatBotMessageReceived)

	p.cli = cli
	return p, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "dingtalk"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for DingTalk events
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	if err := p.cli.Start(p.ctx); err != nil {
		return fmt.Errorf("failed to start DingTalk stream client: %w", err)
	}

	log.Printf("[DingTalk] Stream client connected")
	return nil
}

// Stop shuts down the DingTalk connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.cli != nil {
		p.cli.Close()
	}
	return nil
}

// Send sends a message to a DingTalk conversation
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	// DingTalk uses sessionWebhook for replies
	sessionWebhook := ""
	if resp.Metadata != nil {
		sessionWebhook = resp.Metadata["session_webhook"]
	}

	if sessionWebhook == "" {
		// Try to get from stored webhooks (for async replies)
		p.mu.RLock()
		sessionWebhook = p.webhooks[channelID]
		p.mu.RUnlock()
	}

	if sessionWebhook == "" {
		return fmt.Errorf("no session webhook available for conversation %s", channelID)
	}

	replier := chatbot.NewChatbotReplier()
	return replier.SimpleReplyText(ctx, sessionWebhook, []byte(resp.Text))
}

// onChatBotMessageReceived handles incoming chatbot messages
func (p *Platform) onChatBotMessageReceived(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
	if data == nil {
		return []byte(""), nil
	}

	// Extract text content
	text := strings.TrimSpace(data.Text.Content)
	if text == "" {
		return []byte(""), nil
	}

	// Check if we should respond
	if !p.shouldRespond(data) {
		return []byte(""), nil
	}

	// Clean @mention from text
	text = p.cleanMention(text)

	// Store session webhook for later use in Send()
	p.mu.Lock()
	p.webhooks[data.ConversationId] = data.SessionWebhook
	p.mu.Unlock()

	if p.messageHandler != nil {
		// Create message
		msg := router.Message{
			ID:        data.MsgId,
			Platform:  "dingtalk",
			ChannelID: data.ConversationId,
			UserID:    data.SenderId,
			Username:  data.SenderNick,
			Text:      text,
			ThreadID:  "",
			Metadata: map[string]string{
				"conversation_type":  data.ConversationType, // "1" = private, "2" = group
				"session_webhook":    data.SessionWebhook,
				"conversation_title": data.ConversationTitle,
				"sender_corp_id":     data.SenderCorpId,
				"chatbot_user_id":    data.ChatbotUserId,
			},
		}

		// Call message handler (this is synchronous)
		p.messageHandler(msg)
	}

	return []byte(""), nil
}

// shouldRespond checks if the bot should respond to this message
func (p *Platform) shouldRespond(data *chatbot.BotCallbackDataModel) bool {
	// Always respond to private chats
	if data.ConversationType == "1" {
		return true
	}

	// In group chats, respond if bot is mentioned
	if data.IsInAtList {
		return true
	}

	return false
}

// cleanMention removes @mention from the message
func (p *Platform) cleanMention(text string) string {
	// DingTalk @mentions are typically at the beginning of the message
	// They appear as @nickname in the text
	// The actual mention handling is done via AtUsers field
	return strings.TrimSpace(text)
}

// ReplyText sends a text reply using the session webhook
func ReplyText(ctx context.Context, sessionWebhook string, text string) error {
	replier := chatbot.NewChatbotReplier()
	return replier.SimpleReplyText(ctx, sessionWebhook, []byte(text))
}

// ReplyMarkdown sends a markdown reply using the session webhook
func ReplyMarkdown(ctx context.Context, sessionWebhook string, title, text string) error {
	replier := chatbot.NewChatbotReplier()
	return replier.SimpleReplyMarkdown(ctx, sessionWebhook, []byte(title), []byte(text))
}
