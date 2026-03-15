package sentryutil

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/ruilisi/lsbot/internal/logger"
)

// RecoverAndReport recovers from a panic, logs it, and reports it to Sentry.
// Usage: defer sentryutil.RecoverAndReport("context description")
func RecoverAndReport(tag string) {
	r := recover()
	if r == nil {
		return
	}
	var err error
	switch v := r.(type) {
	case error:
		err = v
	default:
		err = fmt.Errorf("%v", v)
	}
	logger.Error("[panic] %s: %v", tag, err)
	sentry.CurrentHub().Clone().RecoverWithContext(context.Background(), r)
	sentry.Flush(0)
}

// Go runs fn in a new goroutine with panic recovery and Sentry reporting.
// Usage: sentryutil.Go("context description", func() { ... })
func Go(tag string, fn func()) {
	go func() {
		defer RecoverAndReport(tag)
		fn()
	}()
}
