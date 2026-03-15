package router

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ruilisi/lsbot/internal/logger"
)

// Message represents an incoming message from any platform
type Message struct {
	ID        string
	Platform  string            // "slack", "telegram", "discord", etc.
	ChannelID string            // Channel/Chat ID
	UserID    string            // User who sent the message
	Username  string            // Human-readable username
	Text      string            // Message content
	ThreadID  string            // For threaded replies
	MediaID   string            // Media file ID (for file/image/voice/video messages)
	FileName  string            // Original filename (for file messages)
	Metadata  map[string]string // Platform-specific metadata
}

// FileAttachment represents a file to upload and send
type FileAttachment struct {
	Path      string // Local file path to upload and send
	Name      string // Display name (defaults to filepath.Base)
	MediaType string // "file", "image", "voice", "video" (default: "file")
}

// Response represents a response to send back
type Response struct {
	Text     string
	Files    []FileAttachment  // File attachments to send
	ThreadID string            // Reply in thread if set
	Metadata map[string]string // Platform-specific options
}

// Platform interface for messaging platforms
type Platform interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	Send(ctx context.Context, channelID string, resp Response) error
	SetMessageHandler(handler func(msg Message))
}

// ProgressFunc sends an intermediate progress message to the user.
type ProgressFunc func(text string)

type progressKeyType struct{}

// ContextWithProgress attaches a ProgressFunc to the context.
func ContextWithProgress(ctx context.Context, fn ProgressFunc) context.Context {
	return context.WithValue(ctx, progressKeyType{}, fn)
}

// ProgressFromContext retrieves the ProgressFunc from the context, or nil.
func ProgressFromContext(ctx context.Context) ProgressFunc {
	fn, _ := ctx.Value(progressKeyType{}).(ProgressFunc)
	return fn
}

// MessageHandler processes incoming messages and returns responses
type MessageHandler func(ctx context.Context, msg Message) (Response, error)

// Router manages multiple messaging platforms
type Router struct {
	platforms map[string]Platform
	handler   MessageHandler
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new Router
func New(handler MessageHandler) *Router {
	return &Router{
		platforms: make(map[string]Platform),
		handler:   handler,
	}
}

// Register adds a platform to the router
func (r *Router) Register(platform Platform) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := platform.Name()
	r.platforms[name] = platform

	// Set up message handling for this platform
	platform.SetMessageHandler(func(msg Message) {
		go r.handleMessage(msg)
	})

	logger.Info("[Router] Registered platform: %s", name)
}

// handleMessage processes an incoming message
func (r *Router) handleMessage(msg Message) {
	// Use a generous timeout for the agent work — browser automation tasks can take
	// many rounds (each ~5s) so 2 minutes is far too short.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logger.Info("[Router] Message from %s/%s: %s", msg.Platform, msg.Username, msg.Text)

	// Attach a progress callback so the agent can send intermediate updates.
	r.mu.RLock()
	plat, platOK := r.platforms[msg.Platform]
	r.mu.RUnlock()
	if platOK {
		progressResp := Response{
			ThreadID: msg.ThreadID,
			Metadata: msg.Metadata,
		}
		ctx = ContextWithProgress(ctx, func(text string) {
			progressResp.Text = text
			sendCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := plat.Send(sendCtx, msg.ChannelID, progressResp); err != nil {
				logger.Warn("[Router] Failed to send progress: %v", err)
			}
		})
	}

	// Call the message handler
	resp, err := r.handler(ctx, msg)
	if err != nil {
		logger.Error("[Router] Error handling message: %v", err)
		resp = Response{Text: friendlyError(err)}
	}

	// Use a fresh context for sending the response so that an expired agent context
	// doesn't prevent the reply from being delivered.
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer sendCancel()

	// Send response back to the platform
	r.mu.RLock()
	platform, ok := r.platforms[msg.Platform]
	r.mu.RUnlock()

	if ok && (resp.Text != "" || len(resp.Files) > 0) {
		if msg.ThreadID != "" {
			resp.ThreadID = msg.ThreadID
		}
		// Propagate message metadata to response for platform-specific routing (e.g., kf messages)
		if resp.Metadata == nil {
			resp.Metadata = msg.Metadata
		} else if msg.Metadata != nil {
			for k, v := range msg.Metadata {
				if _, exists := resp.Metadata[k]; !exists {
					resp.Metadata[k] = v
				}
			}
		}
		if err := platform.Send(sendCtx, msg.ChannelID, resp); err != nil {
			logger.Error("[Router] Error sending response: %v", err)
			// Try to notify the user about the error in chat
			errResp := Response{
				Text:     fmt.Sprintf("[Error] %v", err),
				ThreadID: resp.ThreadID,
				Metadata: resp.Metadata, // Preserve routing metadata (e.g., kf)
			}
			if notifyErr := platform.Send(sendCtx, msg.ChannelID, errResp); notifyErr != nil {
				logger.Error("[Router] Failed to send error notification: %v", notifyErr)
			}
		}
	}
}

// Start begins listening on all registered platforms
func (r *Router) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, platform := range r.platforms {
		logger.Info("[Router] Starting platform: %s", name)
		if err := platform.Start(r.ctx); err != nil {
			return err
		}
	}

	logger.Info("[Router] All platforms started")
	return nil
}

// Stop shuts down all platforms
func (r *Router) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, platform := range r.platforms {
		logger.Info("[Router] Stopping platform: %s", name)
		if err := platform.Stop(); err != nil {
			logger.Error("[Router] Error stopping %s: %v", name, err)
		}
	}

	return nil
}

// SendToUser sends a proactive message to a user on a specific platform
func (r *Router) SendToUser(platformName, channelID string, resp Response) error {
	r.mu.RLock()
	platform, ok := r.platforms[platformName]
	r.mu.RUnlock()
	if !ok {
		return fmt.Errorf("platform %s not registered", platformName)
	}
	return platform.Send(context.Background(), channelID, resp)
}

// Wait blocks until the router is stopped
func (r *Router) Wait() {
	if r.ctx != nil {
		<-r.ctx.Done()
	}
}

// friendlyError converts AI provider errors into user-facing messages with actionable links.
func friendlyError(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "overdue-payment") || strings.Contains(msg, "account is in good standing"):
		return "AI 服务账户欠费，请前往充值: https://usercenter2.aliyun.com/finance/fund-management"
	case strings.Contains(msg, "invalid_api_key") || strings.Contains(msg, "Incorrect API key"):
		return "AI API Key 无效，请检查配置。"
	case strings.Contains(msg, "rate_limit") || strings.Contains(msg, "Rate limit"):
		return "AI 请求频率超限，请稍后再试。"
	case strings.Contains(msg, "only authorized for use with Claude Code"):
		return "Claude Setup Token 仅限 Claude Code 使用，请改用 API Key (console.anthropic.com)。"
	case strings.Contains(msg, "unexpected EOF") || msg == "EOF" || strings.Contains(msg, "connection reset"):
		return "AI 服务连接中断，请稍后重试。"
	default:
		return fmt.Sprintf("处理消息时出错: %v", err)
	}
}
