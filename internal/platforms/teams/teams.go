package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/sentryutil"
)

// Platform implements router.Platform for Microsoft Teams via Bot Framework
type Platform struct {
	config         Config
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	server         *http.Server
	accessToken    string
	tokenExpiry    time.Time
	tokenMu        sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds Microsoft Teams configuration
type Config struct {
	AppID       string // Bot Framework App ID
	AppPassword string // Bot Framework App Password
	TenantID    string // Azure AD Tenant ID
	WebhookPort int    // Port for incoming webhooks (default: 8086)
}

// New creates a new Teams platform
func New(cfg Config) (*Platform, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("Teams app ID is required")
	}
	if cfg.AppPassword == "" {
		return nil, fmt.Errorf("Teams app password is required")
	}
	if cfg.WebhookPort == 0 {
		cfg.WebhookPort = 8086
	}

	return &Platform{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "teams"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for Teams webhook messages
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/messages", p.handleMessage)

	p.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", p.config.WebhookPort),
		Handler: mux,
	}

	sentryutil.Go("teams webhook server", func() {
		log.Printf("[Teams] Webhook server listening on :%d", p.config.WebhookPort)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Teams] Server error: %v", err)
		}
	})

	log.Printf("[Teams] Platform started, app_id: %s", p.config.AppID)
	return nil
}

// Stop shuts down the Teams connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	}
	return nil
}

// Send sends a message via Bot Framework REST API
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	if resp.Text == "" {
		return nil
	}

	token, err := p.getAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// channelID is expected to be in format "serviceURL|conversationID"
	parts := strings.SplitN(channelID, "|", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid channel ID format, expected serviceURL|conversationID")
	}
	serviceURL, conversationID := parts[0], parts[1]

	activityURL := fmt.Sprintf("%s/v3/conversations/%s/activities", strings.TrimRight(serviceURL, "/"), conversationID)

	payload := map[string]any{
		"type": "message",
		"text": resp.Text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, activityURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("Teams API error %d: %s", httpResp.StatusCode, string(respBody))
	}

	return nil
}

// getAccessToken retrieves or refreshes the OAuth2 access token
func (p *Platform) getAccessToken(ctx context.Context) (string, error) {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return p.accessToken, nil
	}

	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {p.config.AppID},
		"client_secret": {p.config.AppPassword},
		"scope":         {"https://api.botframework.com/.default"},
	}

	tokenURL := "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token"
	if p.config.TenantID != "" {
		tokenURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", p.config.TenantID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	p.accessToken = tokenResp.AccessToken
	p.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return p.accessToken, nil
}

// handleMessage processes incoming Bot Framework activity messages
func (p *Platform) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var activity botActivity
	if err := json.NewDecoder(r.Body).Decode(&activity); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)

	// Only process message activities
	if activity.Type != "message" || activity.Text == "" {
		return
	}

	if p.messageHandler != nil {
		// Compose channelID as serviceURL|conversationID for Send()
		channelID := activity.ServiceURL + "|" + activity.Conversation.ID

		p.messageHandler(router.Message{
			ID:        activity.ID,
			Platform:  "teams",
			ChannelID: channelID,
			UserID:    activity.From.ID,
			Username:  activity.From.Name,
			Text:      activity.Text,
			Metadata: map[string]string{
				"service_url":     activity.ServiceURL,
				"conversation_id": activity.Conversation.ID,
				"tenant_id":       activity.Conversation.TenantID,
			},
		})
	}
}

// Bot Framework activity types
type botActivity struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	Text         string `json:"text"`
	ServiceURL   string `json:"serviceUrl"`
	From         botAccount `json:"from"`
	Conversation struct {
		ID       string `json:"id"`
		TenantID string `json:"tenantId"`
	} `json:"conversation"`
}

type botAccount struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
