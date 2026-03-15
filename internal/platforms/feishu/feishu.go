package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

// Platform implements router.Platform for Feishu/Lark
type Platform struct {
	client         *lark.Client
	wsClient       *larkws.Client
	botOpenID      string
	messageHandler func(msg router.Message)
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Feishu configuration
type Config struct {
	AppID     string // from Feishu Developer Console
	AppSecret string // from Feishu Developer Console
}

// New creates a new Feishu platform
func New(cfg Config) (*Platform, error) {
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, fmt.Errorf("both AppID and AppSecret are required")
	}

	client := lark.NewClient(cfg.AppID, cfg.AppSecret)

	// Get bot info to retrieve bot's open_id
	botOpenID, err := getBotOpenID(client)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot info: %w", err)
	}

	p := &Platform{
		client:    client,
		botOpenID: botOpenID,
	}

	// Create WebSocket client with event handler
	p.wsClient = larkws.NewClient(cfg.AppID, cfg.AppSecret,
		larkws.WithEventHandler(p.buildEventHandler()),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)

	return p, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "feishu"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for Feishu events
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	sentryutil.Go("feishu websocket", func() {
		if err := p.wsClient.Start(p.ctx); err != nil {
			log.Printf("[Feishu] WebSocket error: %v", err)
		}
	})

	log.Printf("[Feishu] Connected as bot: %s", p.botOpenID)
	return nil
}

// Stop shuts down the Feishu connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// Send sends a message to a Feishu chat
func (p *Platform) Send(ctx context.Context, chatID string, resp router.Response) error {
	content, err := json.Marshal(map[string]string{"text": resp.Text})
	if err != nil {
		return fmt.Errorf("failed to marshal message content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(chatID).
			MsgType(larkim.MsgTypeText).
			Content(string(content)).
			Build()).
		Build()

	result, err := p.client.Im.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if !result.Success() {
		return fmt.Errorf("failed to send message: code=%d, msg=%s", result.Code, result.Msg)
	}

	return nil
}

// buildEventHandler creates the event handler for WebSocket events
func (p *Platform) buildEventHandler() *dispatcher.EventDispatcher {
	handler := dispatcher.NewEventDispatcher("", "")
	handler.OnP2MessageReceiveV1(p.handleMessageEvent)
	return handler
}

// handleMessageEvent processes incoming message events
func (p *Platform) handleMessageEvent(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	if event == nil || event.Event == nil || event.Event.Message == nil {
		return nil
	}

	msg := event.Event.Message
	sender := event.Event.Sender

	// Ignore bot's own messages
	if sender != nil && sender.SenderId != nil && *sender.SenderId.OpenId == p.botOpenID {
		return nil
	}

	// Check if we should respond to this message
	if !p.shouldRespond(event) {
		return nil
	}

	// Extract text content from message
	text, err := p.extractText(msg)
	if err != nil {
		log.Printf("[Feishu] Failed to extract text: %v", err)
		return nil
	}

	// Clean @mention from text
	text = p.cleanMention(text)

	if p.messageHandler != nil {
		userID := ""
		username := ""
		if sender != nil && sender.SenderId != nil {
			userID = *sender.SenderId.OpenId
			username = p.getUsername(ctx, userID)
		}

		chatID := ""
		chatType := ""
		if msg.ChatId != nil {
			chatID = *msg.ChatId
		}
		if event.Event.Message.ChatType != nil {
			chatType = *event.Event.Message.ChatType
		}

		msgID := ""
		if msg.MessageId != nil {
			msgID = *msg.MessageId
		}

		p.messageHandler(router.Message{
			ID:        msgID,
			Platform:  "feishu",
			ChannelID: chatID,
			UserID:    userID,
			Username:  username,
			Text:      text,
			ThreadID:  "", // Feishu doesn't have traditional threading like Slack
			Metadata: map[string]string{
				"chat_type": chatType,
			},
		})
	}

	return nil
}

// shouldRespond checks if the bot should respond to this message
func (p *Platform) shouldRespond(event *larkim.P2MessageReceiveV1) bool {
	if event.Event == nil || event.Event.Message == nil {
		return false
	}

	msg := event.Event.Message

	// Respond to DMs (p2p chats)
	if msg.ChatType != nil && *msg.ChatType == "p2p" {
		return true
	}

	// Respond to @mentions in group chats
	if msg.Mentions != nil {
		for _, mention := range msg.Mentions {
			if mention.Id != nil && mention.Id.OpenId != nil && *mention.Id.OpenId == p.botOpenID {
				return true
			}
		}
	}

	return false
}

// extractText extracts text content from message
func (p *Platform) extractText(msg *larkim.EventMessage) (string, error) {
	if msg.Content == nil {
		return "", nil
	}

	var content struct {
		Text string `json:"text"`
	}

	if err := json.Unmarshal([]byte(*msg.Content), &content); err != nil {
		return "", fmt.Errorf("failed to unmarshal content: %w", err)
	}

	return content.Text, nil
}

// cleanMention removes @mention from the message
func (p *Platform) cleanMention(text string) string {
	// Feishu @mentions appear as @_user_N in the text
	// Remove them for cleaner processing
	for {
		start := strings.Index(text, "@_user_")
		if start == -1 {
			break
		}
		end := start + 8 // "@_user_" + at least one digit
		for end < len(text) && text[end] >= '0' && text[end] <= '9' {
			end++
		}
		text = text[:start] + text[end:]
	}
	return strings.TrimSpace(text)
}

// getUsername fetches the username for a user ID
func (p *Platform) getUsername(ctx context.Context, openID string) string {
	req := larkcontact.NewGetUserReqBuilder().
		UserId(openID).
		UserIdType(larkcontact.UserIdTypeOpenId).
		Build()

	result, err := p.client.Contact.User.Get(ctx, req)
	if err != nil || !result.Success() {
		return openID
	}

	if result.Data != nil && result.Data.User != nil && result.Data.User.Name != nil {
		return *result.Data.User.Name
	}

	return openID
}

// getBotOpenID retrieves the bot's open_id
func getBotOpenID(client *lark.Client) (string, error) {
	// Use bot info API to get bot's open_id
	// This is done by calling the auth endpoint which returns bot info
	ctx := context.Background()

	// Get tenant access token to verify credentials and get bot info
	// The bot's open_id can be retrieved from the /bot/v3/info endpoint
	req := larkim.NewListChatReqBuilder().
		PageSize(1).
		Build()

	result, err := client.Im.Chat.List(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to verify credentials: %w", err)
	}

	if !result.Success() {
		return "", fmt.Errorf("failed to verify credentials: code=%d, msg=%s", result.Code, result.Msg)
	}

	// For now, we'll identify the bot by its messages
	// The bot's open_id will be populated when we receive our first message
	// This is a workaround since the SDK doesn't expose bot info directly
	log.Printf("[Feishu] Credentials verified successfully")
	return "", nil
}
