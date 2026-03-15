package slack

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

// Platform implements router.Platform for Slack
type Platform struct {
	client         *slack.Client
	socketClient   *socketmode.Client
	botUserID      string
	messageHandler func(msg router.Message)
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Slack configuration
type Config struct {
	BotToken string // xoxb-...
	AppToken string // xapp-...
}

// New creates a new Slack platform
func New(cfg Config) (*Platform, error) {
	if cfg.BotToken == "" || cfg.AppToken == "" {
		return nil, fmt.Errorf("both BotToken and AppToken are required")
	}

	client := slack.New(
		cfg.BotToken,
		slack.OptionAppLevelToken(cfg.AppToken),
	)

	socketClient := socketmode.New(
		client,
		socketmode.OptionDebug(false),
	)

	// Get bot user ID
	authTest, err := client.AuthTest()
	if err != nil {
		return nil, fmt.Errorf("failed to auth: %w", err)
	}

	return &Platform{
		client:       client,
		socketClient: socketClient,
		botUserID:    authTest.UserID,
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "slack"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for Slack events
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	sentryutil.Go("slack handle events", p.handleEvents)
	sentryutil.Go("slack socket mode", func() {
		if err := p.socketClient.RunContext(p.ctx); err != nil {
			log.Printf("[Slack] Socket mode error: %v", err)
		}
	})

	log.Printf("[Slack] Connected as bot user: %s", p.botUserID)
	return nil
}

// Stop shuts down the Slack connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

// Send sends a message to a Slack channel
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	options := []slack.MsgOption{
		slack.MsgOptionText(resp.Text, false),
	}

	if resp.ThreadID != "" {
		options = append(options, slack.MsgOptionTS(resp.ThreadID))
	}

	_, _, err := p.client.PostMessageContext(ctx, channelID, options...)
	return err
}

// handleEvents processes incoming Slack events
func (p *Platform) handleEvents() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case evt := <-p.socketClient.Events:
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}
				p.socketClient.Ack(*evt.Request)
				p.handleEventsAPI(eventsAPIEvent)

			case socketmode.EventTypeSlashCommand:
				cmd, ok := evt.Data.(slack.SlashCommand)
				if !ok {
					continue
				}
				p.socketClient.Ack(*evt.Request)
				p.handleSlashCommand(cmd)
			}
		}
	}
}

// handleEventsAPI processes Events API payloads
func (p *Platform) handleEventsAPI(event slackevents.EventsAPIEvent) {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.MessageEvent:
			// Ignore bot's own messages
			if ev.User == p.botUserID || ev.BotID != "" {
				return
			}

			// Only respond to direct mentions or DMs
			if !p.shouldRespond(ev) {
				return
			}

			text := p.cleanMention(ev.Text)

			if p.messageHandler != nil {
				p.messageHandler(router.Message{
					ID:        ev.TimeStamp,
					Platform:  "slack",
					ChannelID: ev.Channel,
					UserID:    ev.User,
					Username:  p.getUsername(ev.User),
					Text:      text,
					ThreadID:  ev.ThreadTimeStamp,
					Metadata: map[string]string{
						"channel_type": ev.ChannelType,
					},
				})
			}

		case *slackevents.AppMentionEvent:
			text := p.cleanMention(ev.Text)

			if p.messageHandler != nil {
				p.messageHandler(router.Message{
					ID:        ev.TimeStamp,
					Platform:  "slack",
					ChannelID: ev.Channel,
					UserID:    ev.User,
					Username:  p.getUsername(ev.User),
					Text:      text,
					ThreadID:  ev.ThreadTimeStamp,
					Metadata:  map[string]string{},
				})
			}
		}
	}
}

// handleSlashCommand processes slash commands
func (p *Platform) handleSlashCommand(cmd slack.SlashCommand) {
	if p.messageHandler != nil {
		p.messageHandler(router.Message{
			ID:        cmd.TriggerID,
			Platform:  "slack",
			ChannelID: cmd.ChannelID,
			UserID:    cmd.UserID,
			Username:  cmd.UserName,
			Text:      cmd.Command + " " + cmd.Text,
			Metadata: map[string]string{
				"command": cmd.Command,
			},
		})
	}
}

// shouldRespond checks if the bot should respond to this message
func (p *Platform) shouldRespond(ev *slackevents.MessageEvent) bool {
	// Respond to DMs
	if ev.ChannelType == "im" {
		return true
	}

	// Respond to mentions
	if strings.Contains(ev.Text, "<@"+p.botUserID+">") {
		return true
	}

	return false
}

// cleanMention removes the bot mention from the message
func (p *Platform) cleanMention(text string) string {
	mention := "<@" + p.botUserID + ">"
	text = strings.ReplaceAll(text, mention, "")
	return strings.TrimSpace(text)
}

// getUsername fetches the username for a user ID
func (p *Platform) getUsername(userID string) string {
	user, err := p.client.GetUserInfo(userID)
	if err != nil {
		return userID
	}
	return user.Name
}
