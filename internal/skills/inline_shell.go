package skills

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// inlineShellRe matches !`cmd` expressions in skill content.
var inlineShellRe = regexp.MustCompile("!`([^`\n]+)`")

// ExpandInlineShell scans content for !`cmd` expressions, executes each
// via bash with a 5-second timeout, and substitutes the trimmed stdout
// in place. Errors and timeouts silently substitute an empty string.
func ExpandInlineShell(content string) string {
	return inlineShellRe.ReplaceAllStringFunc(content, func(match string) string {
		sub := inlineShellRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		cmd := sub[1]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var out bytes.Buffer
		c := exec.CommandContext(ctx, "bash", "-c", cmd)
		c.Stdout = &out
		if err := c.Run(); err != nil {
			return ""
		}
		return strings.TrimSpace(out.String())
	})
}
