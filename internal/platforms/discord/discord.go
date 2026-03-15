package discord

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/ruilisi/lsbot/internal/router"
)

// Platform implements router.Platform for Discord
type Platform struct {
	session        *discordgo.Session
	botUserID      string
	messageHandler func(msg router.Message)
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Discord configuration
type Config struct {
	Token string // Bot token from Discord Developer Portal
}

// New creates a new Discord platform
func New(cfg Config) (*Platform, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("Discord bot token is required")
	}

	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Set intents
	session.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentMessageContent

	return &Platform{
		session: session,
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "discord"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for Discord events
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Add message handler
	p.session.AddHandler(p.handleMessage)

	// Open connection
	if err := p.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	// Get bot user ID
	user, err := p.session.User("@me")
	if err != nil {
		return fmt.Errorf("failed to get bot user: %w", err)
	}
	p.botUserID = user.ID

	log.Printf("[Discord] Connected as bot: %s#%s", user.Username, user.Discriminator)
	return nil
}

// Stop shuts down the Discord connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return p.session.Close()
}

// Send sends a message to a Discord channel
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	var reference *discordgo.MessageReference
	if resp.ThreadID != "" {
		reference = &discordgo.MessageReference{
			MessageID: resp.ThreadID,
			ChannelID: channelID,
		}
	}

	_, err := p.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:   resp.Text,
		Reference: reference,
	})
	return err
}

// handleMessage processes incoming Discord messages
func (p *Platform) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots
	if m.Author.Bot {
		return
	}

	// Check if we should respond
	if !p.shouldRespond(m) {
		return
	}

	text := p.cleanMention(m.Content)

	if p.messageHandler != nil {
		// Determine channel type
		channel, err := s.Channel(m.ChannelID)
		channelType := "unknown"
		if err == nil {
			switch channel.Type {
			case discordgo.ChannelTypeDM:
				channelType = "dm"
			case discordgo.ChannelTypeGuildText:
				channelType = "guild"
			case discordgo.ChannelTypeGroupDM:
				channelType = "group_dm"
			}
		}

		threadID := ""
		if m.ReferencedMessage != nil {
			threadID = m.ReferencedMessage.ID
		}

		p.messageHandler(router.Message{
			ID:        m.ID,
			Platform:  "discord",
			ChannelID: m.ChannelID,
			UserID:    m.Author.ID,
			Username:  m.Author.Username,
			Text:      text,
			ThreadID:  threadID,
			Metadata: map[string]string{
				"channel_type": channelType,
				"guild_id":     m.GuildID,
			},
		})
	}
}

// shouldRespond checks if the bot should respond to this message
func (p *Platform) shouldRespond(m *discordgo.MessageCreate) bool {
	// Get channel info to determine if DM
	channel, err := p.session.Channel(m.ChannelID)
	if err != nil {
		return false
	}

	// Always respond in DMs
	if channel.Type == discordgo.ChannelTypeDM {
		return true
	}

	// In servers, only respond to mentions or replies to bot
	for _, mention := range m.Mentions {
		if mention.ID == p.botUserID {
			return true
		}
	}

	// Check if replying to bot's message
	if m.ReferencedMessage != nil && m.ReferencedMessage.Author.ID == p.botUserID {
		return true
	}

	return false
}

// cleanMention removes the bot mention from the message
func (p *Platform) cleanMention(text string) string {
	// Discord mentions are in format <@USER_ID> or <@!USER_ID>
	mention1 := "<@" + p.botUserID + ">"
	mention2 := "<@!" + p.botUserID + ">"
	text = strings.ReplaceAll(text, mention1, "")
	text = strings.ReplaceAll(text, mention2, "")
	return strings.TrimSpace(text)
}
