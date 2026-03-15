package agent

import (
	"github.com/ruilisi/lsbot/internal/logger"
	"github.com/ruilisi/lsbot/internal/router"
)

// RouterCronNotifier implements cron.ChatNotifier by sending messages through the router
type RouterCronNotifier struct {
	router *router.Router
}

// NewRouterCronNotifier creates a new notifier that sends cron messages through the router
func NewRouterCronNotifier(r *router.Router) *RouterCronNotifier {
	return &RouterCronNotifier{router: r}
}

// NotifyChat logs a cron notification (no specific target)
func (n *RouterCronNotifier) NotifyChat(message string) error {
	logger.Info("[CRON] %s", message)
	return nil
}

// NotifyChatUser sends a cron notification to a specific user via the router
func (n *RouterCronNotifier) NotifyChatUser(platform, channelID, userID, message string) error {
	return n.router.SendToUser(platform, channelID, router.Response{Text: message})
}
