// Package termui provides helpers for styled terminal output.
// Use these functions consistently so the visual language is coherent:
//
//   - Warn  – bold bright-red (visible on Solarized Dark), for config conflicts / misuse
//   - Error – bold bright-red + "ERROR" prefix, for fatal or serious failures
//   - Info  – cyan, for neutral informational messages
//   - OK    – green, for success confirmations
//   - Bold  – plain bold, for emphasis without colour
//
// All output goes to os.Stderr so it never pollutes piped stdout.
package termui

import (
	"fmt"
	"os"
	"strings"
)

// ANSI codes
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Cyan       = "\033[36m"
	Gray       = "\033[90m"
	BrightRed  = "\033[1;91m" // bold bright-red — readable on Solarized Dark
	BrightCyan = "\033[1;96m"
)

// Warn prints a bold bright-red warning to stderr.
// Use for config conflicts, deprecated settings, or anything the user should act on.
func Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")
	for i, l := range lines {
		if i == 0 {
			fmt.Fprintf(os.Stderr, "%s⚠  %s%s\n", BrightRed, l, Reset)
		} else {
			fmt.Fprintf(os.Stderr, "%s   %s%s\n", BrightRed, l, Reset)
		}
	}
}

// Error prints a bold bright-red error to stderr.
// Use for serious failures the user must fix before the program can work.
func Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")
	for i, l := range lines {
		if i == 0 {
			fmt.Fprintf(os.Stderr, "%sERROR  %s%s\n", BrightRed, l, Reset)
		} else {
			fmt.Fprintf(os.Stderr, "%s       %s%s\n", BrightRed, l, Reset)
		}
	}
}

// Info prints a cyan informational message to stderr.
func Info(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s%s%s\n", Cyan, fmt.Sprintf(format, args...), Reset)
}

// OK prints a green success message to stderr.
func OK(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s✓  %s%s\n", Green, fmt.Sprintf(format, args...), Reset)
}

// Colorize wraps text in the given ANSI code and resets after.
func Colorize(code, text string) string {
	return code + text + Reset
}
