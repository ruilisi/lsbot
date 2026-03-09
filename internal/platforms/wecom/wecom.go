package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/pltanton/lingti-bot/internal/sentryutil"
)

const (
	tokenURL     = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	sendMsgURL   = "https://qyapi.weixin.qq.com/cgi-bin/message/send"
	kfSyncMsgURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/sync_msg"
	kfSendMsgURL = "https://qyapi.weixin.qq.com/cgi-bin/kf/send_msg"
)

// Platform implements router.Platform for WeChat Work (企业微信)
type Platform struct {
	corpID         string
	agentID        string
	secret         string
	token          string
	encodingAESKey string

	msgCrypt       *MsgCrypt
	accessToken    string
	tokenExpiry    time.Time
	tokenMu        sync.RWMutex
	messageHandler func(msg router.Message)
	server         *http.Server
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds WeChat Work configuration
type Config struct {
	CorpID         string // 企业ID
	AgentID        string // 应用AgentId
	Secret         string // 应用Secret
	Token          string // 回调Token
	EncodingAESKey string // 回调EncodingAESKey
	CallbackPort   int    // 回调服务端口 (default: 8080)
}

// New creates a new WeChat Work platform
func New(cfg Config) (*Platform, error) {
	if cfg.CorpID == "" || cfg.AgentID == "" || cfg.Secret == "" {
		return nil, fmt.Errorf("CorpID, AgentID, and Secret are required")
	}
	if cfg.Token == "" || cfg.EncodingAESKey == "" {
		return nil, fmt.Errorf("Token and EncodingAESKey are required for callback")
	}

	msgCrypt, err := NewMsgCrypt(cfg.Token, cfg.EncodingAESKey, cfg.CorpID)
	if err != nil {
		return nil, fmt.Errorf("failed to create message cryptographer: %w", err)
	}

	p := &Platform{
		corpID:         cfg.CorpID,
		agentID:        cfg.AgentID,
		secret:         cfg.Secret,
		token:          cfg.Token,
		encodingAESKey: cfg.EncodingAESKey,
		msgCrypt:       msgCrypt,
	}

	// Set up HTTP server for callbacks (skip if CallbackPort < 0, e.g. API-only mode)
	if cfg.CallbackPort >= 0 {
		port := cfg.CallbackPort
		if port == 0 {
			port = 8080
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/wecom/callback", p.handleCallback)
		p.server = &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		}
	}

	return p, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "wecom"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for WeChat Work events
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Get initial access token
	if err := p.refreshToken(); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Start token refresh goroutine
	go p.tokenRefreshLoop()

	// Start HTTP server (if configured)
	if p.server != nil {
		sentryutil.Go("wecom callback server", func() {
			logger.Info("[WeCom] Starting callback server on %s", p.server.Addr)
			if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("[WeCom] Server error: %v", err)
			}
		})
	}

	logger.Info("[WeCom] Connected with CorpID: %s, AgentID: %s", p.corpID, p.agentID)
	return nil
}

// Stop shuts down the WeChat Work connection
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

// Send sends a message to a WeChat Work user
func (p *Platform) Send(ctx context.Context, userID string, resp router.Response) error {
	// Handle KF (customer service) messages via kf/send_msg API
	if resp.Metadata != nil && resp.Metadata["kf"] == "true" {
		if resp.Text != "" {
			if err := p.SendKfMessage(resp.Metadata["external_userid"], resp.Metadata["open_kfid"], resp.Text); err != nil {
				return err
			}
		}
		return nil
	}

	// Send text message if present
	if resp.Text != "" {
		if err := p.sendTextMessage(userID, resp.Text); err != nil {
			return err
		}
	}

	// Send file attachments — notify user on per-file errors and continue
	var failCount int
	for _, file := range resp.Files {
		mediaType := file.MediaType
		if mediaType == "" {
			mediaType = "file"
		}

		name := file.Name
		if name == "" {
			name = filepath.Base(file.Path)
		}

		mediaID, err := p.UploadMedia(file.Path, mediaType)
		if err != nil {
			logger.Error("[WeCom] Failed to upload %s: %v", file.Path, err)
			_ = p.sendTextMessage(userID, fmt.Sprintf("[Error] Failed to send file \"%s\": %v", name, err))
			failCount++
			continue
		}

		if err := p.SendMediaMessage(userID, mediaID, mediaType); err != nil {
			logger.Error("[WeCom] Failed to send media %s: %v", file.Path, err)
			_ = p.sendTextMessage(userID, fmt.Sprintf("[Error] Failed to send file \"%s\": %v", name, err))
			failCount++
			continue
		}
	}

	if failCount > 0 {
		return fmt.Errorf("failed to send %d file(s)", failCount)
	}
	return nil
}

// sendTextMessage sends a markdown message to a user.
func (p *Platform) sendTextMessage(userID string, text string) error {
	token, err := p.getToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	agentID, _ := strconv.Atoi(p.agentID)
	msg := map[string]any{
		"touser":  userID,
		"msgtype": "markdown",
		"agentid": agentID,
		"markdown": map[string]string{
			"content": text,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("%s?access_token=%s", sendMsgURL, token)
	httpResp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// handleCallback handles incoming callback requests from WeChat Work
func (p *Platform) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	msgSignature := query.Get("msg_signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")

	switch r.Method {
	case http.MethodGet:
		// URL verification
		echostr := query.Get("echostr")
		plaintext, err := p.msgCrypt.VerifyURL(msgSignature, timestamp, nonce, echostr)
		if err != nil {
			logger.Error("[WeCom] URL verification failed: %v", err)
			http.Error(w, "verification failed", http.StatusForbidden)
			return
		}
		w.Write([]byte(plaintext))

	case http.MethodPost:
		// Message handling
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("[WeCom] Failed to read body: %v", err)
			http.Error(w, "read failed", http.StatusBadRequest)
			return
		}

		var encryptedMsg EncryptedMsg
		if err := xml.Unmarshal(body, &encryptedMsg); err != nil {
			logger.Error("[WeCom] Failed to parse XML: %v", err)
			http.Error(w, "parse failed", http.StatusBadRequest)
			return
		}

		plaintext, err := p.msgCrypt.DecryptMsg(msgSignature, timestamp, nonce, &encryptedMsg)
		if err != nil {
			logger.Error("[WeCom] Failed to decrypt message: %v", err)
			http.Error(w, "decrypt failed", http.StatusBadRequest)
			return
		}

		// Parse the decrypted message
		p.processMessage(plaintext)

		// Return success (empty response)
		w.WriteHeader(http.StatusOK)
	}
}

// ReceivedMsg represents a received message from WeChat Work
type ReceivedMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        string   `xml:"MsgId"`
	AgentID      string   `xml:"AgentID"`
	// Media fields
	MediaId  string `xml:"MediaId"`
	Format   string `xml:"Format"`   // Voice format (amr)
	PicUrl   string `xml:"PicUrl"`   // Image thumbnail URL
	FileName string `xml:"FileName"` // File name
	FileSize string `xml:"FileSize"` // File size
	// Event fields
	Event    string `xml:"Event"`
	EventKey string `xml:"EventKey"`
	// Customer service (kf) fields
	Token string `xml:"Token"` // Token from kf_msg_or_event callback, used for sync_msg API
}

// processMessage processes the decrypted message
func (p *Platform) processMessage(plaintext []byte) {
	var msg ReceivedMsg
	if err := xml.Unmarshal(plaintext, &msg); err != nil {
		logger.Error("[WeCom] Failed to parse message: %v", err)
		return
	}

	// Handle event messages
	if msg.MsgType == "event" {
		if msg.Event == "kf_msg_or_event" {
			logger.Info("[WeCom] Received kf_msg_or_event, token=%s", msg.Token)
			p.handleKfEvent(msg.Token)
		} else {
			logger.Trace("[WeCom] Ignoring event: %s", msg.Event)
		}
		return
	}

	routerMsg := router.Message{
		ID:        msg.MsgId,
		Platform:  "wecom",
		ChannelID: msg.FromUserName,
		UserID:    msg.FromUserName,
		Username:  msg.FromUserName,
		Text:      msg.Content,
		Metadata: map[string]string{
			"agent_id": msg.AgentID,
			"msg_type": msg.MsgType,
		},
	}

	switch msg.MsgType {
	case "text":
		// Text is already set via msg.Content
	case "image":
		routerMsg.MediaID = msg.MediaId
		routerMsg.Text = "[图片]"
		routerMsg.Metadata["pic_url"] = msg.PicUrl
	case "voice":
		routerMsg.MediaID = msg.MediaId
		routerMsg.Text = "[语音]"
		routerMsg.Metadata["format"] = msg.Format
	case "video":
		routerMsg.MediaID = msg.MediaId
		routerMsg.Text = "[视频]"
	case "file":
		routerMsg.MediaID = msg.MediaId
		routerMsg.FileName = msg.FileName
		routerMsg.Text = "[文件] " + msg.FileName
		routerMsg.Metadata["file_size"] = msg.FileSize
	default:
		logger.Trace("[WeCom] Ignoring message type: %s", msg.MsgType)
		return
	}

	if p.messageHandler != nil {
		p.messageHandler(routerMsg)
	}
}

// handleKfEvent processes a kf_msg_or_event by calling sync_msg to fetch actual messages
func (p *Platform) handleKfEvent(token string) {
	accounts, err := p.ListKfAccounts()
	if err != nil {
		logger.Error("[WeCom] Failed to list kf accounts: %v", err)
		return
	}

	var allMessages []KfMessage
	for _, acc := range accounts {
		result, err := p.SyncKfMessages(token, "", acc.OpenKfID, 1000)
		if err != nil {
			logger.Error("[WeCom] Failed to sync kf messages for %s (%s): %v", acc.Name, acc.OpenKfID, err)
			continue
		}
		logger.Info("[WeCom] KF sync_msg for %s: %d messages, has_more=%d", acc.Name, len(result.MsgList), result.HasMore)
		allMessages = append(allMessages, result.MsgList...)
	}

	for _, msg := range allMessages {
		if msg.Origin != 3 {
			logger.Trace("[WeCom] Skipping kf message origin=%d, type=%s", msg.Origin, msg.MsgType)
			continue
		}

		if msg.MsgType != "text" || msg.Text == nil || msg.Text.Content == "" {
			logger.Info("[WeCom] Skipping non-text kf message: type=%s, msgid=%s", msg.MsgType, msg.MsgID)
			continue
		}

		logger.Info("[WeCom] KF message from %s via %s: %s", msg.ExternalUserID, msg.OpenKfID, msg.Text.Content)

		routerMsg := router.Message{
			ID:        msg.MsgID,
			Platform:  "wecom",
			ChannelID: msg.ExternalUserID,
			UserID:    msg.ExternalUserID,
			Username:  msg.ExternalUserID,
			Text:      msg.Text.Content,
			Metadata: map[string]string{
				"message_id":      msg.MsgID,
				"msg_type":        msg.MsgType,
				"kf":              "true",
				"open_kfid":       msg.OpenKfID,
				"external_userid": msg.ExternalUserID,
			},
		}

		if p.messageHandler != nil {
			p.messageHandler(routerMsg)
		}
	}
}

// Token management

type tokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (p *Platform) refreshToken() error {
	url := fmt.Sprintf("%s?corpid=%s&corpsecret=%s", tokenURL, p.corpID, p.secret)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("token API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	p.tokenMu.Lock()
	p.accessToken = result.AccessToken
	// Refresh 5 minutes before expiry
	p.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)
	p.tokenMu.Unlock()

	logger.Trace("[WeCom] Access token refreshed, expires in %d seconds", result.ExpiresIn)
	return nil
}

func (p *Platform) getToken() (string, error) {
	p.tokenMu.RLock()
	token := p.accessToken
	expiry := p.tokenExpiry
	p.tokenMu.RUnlock()

	if time.Now().After(expiry) {
		if err := p.refreshToken(); err != nil {
			return "", err
		}
		p.tokenMu.RLock()
		token = p.accessToken
		p.tokenMu.RUnlock()
	}

	return token, nil
}

func (p *Platform) tokenRefreshLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if err := p.refreshToken(); err != nil {
				logger.Error("[WeCom] Failed to refresh token: %v", err)
			}
		}
	}
}

// KF (Customer Service) API types and methods

// KfSyncResult represents the response from kf/sync_msg API
type KfSyncResult struct {
	ErrCode    int         `json:"errcode"`
	ErrMsg     string      `json:"errmsg"`
	NextCursor string      `json:"next_cursor"`
	HasMore    int         `json:"has_more"`
	MsgList    []KfMessage `json:"msg_list"`
}

// KfMessage represents a single message from kf/sync_msg
type KfMessage struct {
	MsgID          string         `json:"msgid"`
	OpenKfID       string         `json:"open_kfid"`
	ExternalUserID string         `json:"external_userid"`
	SendTime       int64          `json:"send_time"`
	Origin         int            `json:"origin"` // 3=customer, 4=system, 5=kf agent
	MsgType        string         `json:"msgtype"`
	Text           *KfTextContent `json:"text,omitempty"`
}

// KfTextContent represents text content in a kf message
type KfTextContent struct {
	Content string `json:"content"`
}

// ListKfAccounts calls the kf/account/list API to list all customer service accounts
func (p *Platform) ListKfAccounts() ([]KfAccount, error) {
	accessToken, err := p.getToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/kf/account/list?access_token=%s", accessToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("failed to call kf/account/list: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode     int         `json:"errcode"`
		ErrMsg      string      `json:"errmsg"`
		AccountList []KfAccount `json:"account_list"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("kf/account/list API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return result.AccountList, nil
}

// CheckKfAvailable attempts to list KF accounts and returns true if KF is available
func (p *Platform) CheckKfAvailable() bool {
	_, err := p.ListKfAccounts()
	return err == nil
}

// KfAccount represents a customer service account
type KfAccount struct {
	OpenKfID string `json:"open_kfid"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
}

// TransKfServiceState transitions a kf session's service state.
// service_state: 0=未处理, 1=人工接入, 2=待接入, 3=机器人, 4=已结束
// servicerUserID is required when transitioning to state 1 (human agent).
func (p *Platform) TransKfServiceState(openKfID, externalUserID string, serviceState int, servicerUserID string) error {
	accessToken, err := p.getToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody := map[string]any{
		"open_kfid":       openKfID,
		"external_userid": externalUserID,
		"service_state":   serviceState,
	}
	if servicerUserID != "" {
		reqBody["servicer_userid"] = servicerUserID
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/kf/service_state/trans?access_token=%s", accessToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to call service_state/trans: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("service_state/trans API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	logger.Info("[WeCom] KF service state transitioned to %d for %s via %s", serviceState, externalUserID, openKfID)
	return nil
}

// GetKfServiceState returns the current service state for a kf session
func (p *Platform) GetKfServiceState(openKfID, externalUserID string) (int, error) {
	accessToken, err := p.getToken()
	if err != nil {
		return -1, fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody := map[string]any{
		"open_kfid":       openKfID,
		"external_userid": externalUserID,
	}
	body, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/kf/service_state/get?access_token=%s", accessToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return -1, fmt.Errorf("failed to call service_state/get: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode      int    `json:"errcode"`
		ErrMsg       string `json:"errmsg"`
		ServiceState int    `json:"service_state"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return -1, fmt.Errorf("failed to decode response: %w", err)
	}
	if result.ErrCode != 0 {
		return -1, fmt.Errorf("service_state/get error: %d - %s", result.ErrCode, result.ErrMsg)
	}
	return result.ServiceState, nil
}

// SyncKfMessages calls the kf/sync_msg API to fetch customer service messages
func (p *Platform) SyncKfMessages(token, cursor, openKfID string, limit int) (*KfSyncResult, error) {
	accessToken, err := p.getToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	if limit <= 0 {
		limit = 1000
	}

	reqBody := map[string]any{
		"cursor": cursor,
		"token":  token,
		"limit":  limit,
	}
	if openKfID != "" {
		reqBody["open_kfid"] = openKfID
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s?access_token=%s", kfSyncMsgURL, accessToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call sync_msg: %w", err)
	}
	defer resp.Body.Close()

	var result KfSyncResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode sync_msg response: %w", err)
	}

	if result.ErrCode != 0 {
		return &result, fmt.Errorf("sync_msg API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return &result, nil
}

// SendKfMessage sends a text message via the kf/send_msg API
func (p *Platform) SendKfMessage(toUser, openKfID, text string) error {
	accessToken, err := p.getToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	reqBody := map[string]any{
		"touser":    toUser,
		"open_kfid": openKfID,
		"msgtype":   "text",
		"text": map[string]string{
			"content": text,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s?access_token=%s", kfSendMsgURL, accessToken)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to call kf/send_msg: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		MsgID   string `json:"msgid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode kf/send_msg response: %w", err)
	}

	if result.ErrCode == 95018 {
		// Session state invalid — check current state and transition to AI bot
		currentState, getErr := p.GetKfServiceState(openKfID, toUser)
		if getErr != nil {
			return fmt.Errorf("failed to get kf session state: %w (original: 95018 - %s)", getErr, result.ErrMsg)
		}
		logger.Info("[WeCom] KF session state for %s: %d", toUser, currentState)

		// State machine transitions to reach state 1 (AI bot):
		// 0 (unprocessed) → 1: direct
		// 2 (waiting) → 1: direct
		// 3 (human agent) → 1: not allowed; must go 3→4→ user resends → 0→1
		// 4 (ended) → cannot transition; user must send new message
		switch currentState {
		case 0, 2:
			logger.Info("[WeCom] Transitioning kf session %s: %d → 1 (AI bot)", toUser, currentState)
			if err := p.TransKfServiceState(openKfID, toUser, 1, ""); err != nil {
				return fmt.Errorf("failed to transition kf session %d→1: %w", currentState, err)
			}
		case 3:
			// End the human session; user's next message will start fresh at state 0
			logger.Info("[WeCom] Ending human agent session for %s: 3 → 4", toUser)
			if err := p.TransKfServiceState(openKfID, toUser, 4, ""); err != nil {
				return fmt.Errorf("failed to end kf session 3→4: %w", err)
			}
			// Now transition from the new session state
			logger.Info("[WeCom] Transitioning kf session %s: → 1 (AI bot)", toUser)
			if err := p.TransKfServiceState(openKfID, toUser, 1, ""); err != nil {
				// Session might need user to re-send; log and return
				return fmt.Errorf("failed to transition after ending session: %w (user may need to resend)", err)
			}
		case 4:
			return fmt.Errorf("kf session ended (state=4), user needs to send a new message")
		case 1:
			// Already in AI bot state, just retry
		default:
			return fmt.Errorf("unexpected kf session state %d", currentState)
		}
		// Retry send
		resp2, err := http.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to retry kf/send_msg: %w", err)
		}
		defer resp2.Body.Close()
		var result2 struct {
			ErrCode int    `json:"errcode"`
			ErrMsg  string `json:"errmsg"`
			MsgID   string `json:"msgid"`
		}
		if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
			return fmt.Errorf("failed to decode retry response: %w", err)
		}
		if result2.ErrCode != 0 {
			return fmt.Errorf("kf/send_msg retry error: %d - %s", result2.ErrCode, result2.ErrMsg)
		}
		logger.Info("[WeCom] KF message sent (after state transition) to %s via %s, msgid=%s", toUser, openKfID, result2.MsgID)
		return nil
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("kf/send_msg API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	logger.Info("[WeCom] KF message sent to %s via %s, msgid=%s", toUser, openKfID, result.MsgID)
	return nil
}
