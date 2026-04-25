package skills

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExpandInlineShell_BasicSubstitution(t *testing.T) {
	out := ExpandInlineShell("hello !`echo world`")
	if out != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", out)
	}
}

func TestExpandInlineShell_MultipleExpressions(t *testing.T) {
	out := ExpandInlineShell("a=!`echo 1` b=!`echo 2`")
	if out != "a=1 b=2" {
		t.Errorf("expected %q, got %q", "a=1 b=2", out)
	}
}

func TestExpandInlineShell_Passthrough(t *testing.T) {
	cases := []string{
		"no shell here",
		"backtick without bang: `ls`",
		"",
		"## heading\n\nsome text\n",
	}
	for _, c := range cases {
		out := ExpandInlineShell(c)
		if out != c {
			t.Errorf("input %q: expected passthrough, got %q", c, out)
		}
	}
}

func TestExpandInlineShell_ErrorSilent(t *testing.T) {
	out := ExpandInlineShell("val: !`nonexistent_cmd_xyz_123`")
	if out != "val: " {
		t.Errorf("expected empty on error, got %q", out)
	}
}

func TestExpandInlineShell_NonZeroExitSilent(t *testing.T) {
	out := ExpandInlineShell("val: !`exit 1`")
	if out != "val: " {
		t.Errorf("expected empty on non-zero exit, got %q", out)
	}
}

func TestExpandInlineShell_OutputTrimmed(t *testing.T) {
	out := ExpandInlineShell("!`printf '  trimmed  '`")
	if out != "trimmed" {
		t.Errorf("expected trimmed output, got %q", out)
	}
}

func TestExpandInlineShell_Timeout(t *testing.T) {
	start := time.Now()
	out := ExpandInlineShell("!`sleep 10`")
	elapsed := time.Since(start)
	if out != "" {
		t.Errorf("expected empty on timeout, got %q", out)
	}
	if elapsed > 6*time.Second {
		t.Errorf("timeout not enforced: took %v", elapsed)
	}
}

func TestExpandInlineShell_PipelineCommand(t *testing.T) {
	out := ExpandInlineShell("!`echo hello | tr a-z A-Z`")
	if out != "HELLO" {
		t.Errorf("expected %q, got %q", "HELLO", out)
	}
}

func TestExpandInlineShell_EnvVarExpansion(t *testing.T) {
	out := ExpandInlineShell("!`echo ${HOME}`")
	if out == "" || out == "${HOME}" {
		t.Errorf("expected HOME to be expanded, got %q", out)
	}
}

func TestExpandInlineShell_CalendarSkillPattern(t *testing.T) {
	// Mirrors the exact expressions used in bundled-skills/calendar/SKILL.md
	input := "- 操作系统: !`uname -s`\n" +
		"- sqlite3: !`command -v sqlite3 >/dev/null 2>&1 && echo available || echo missing`\n" +
		"- 数据库路径: !`echo ${CALENDAR_DB:-$HOME/.lsbot/calendar/calendar.db}`"

	out := ExpandInlineShell(input)
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d:\n%s", len(lines), out)
	}

	// OS line: raw expression replaced with real OS name
	osLine := lines[0]
	if strings.Contains(osLine, "!`") {
		t.Errorf("OS line still contains raw expression: %q", osLine)
	}
	expectedOS := map[string]string{"darwin": "Darwin", "linux": "Linux"}[runtime.GOOS]
	if expectedOS != "" && !strings.Contains(osLine, expectedOS) {
		t.Errorf("OS line %q does not contain expected OS %q", osLine, expectedOS)
	}

	// sqlite3 line: contains "available" or "missing", not raw expression
	sqlite3Line := lines[1]
	if strings.Contains(sqlite3Line, "!`") {
		t.Errorf("sqlite3 line still contains raw expression: %q", sqlite3Line)
	}
	if !strings.Contains(sqlite3Line, "available") && !strings.Contains(sqlite3Line, "missing") {
		t.Errorf("sqlite3 line has unexpected value: %q", sqlite3Line)
	}

	// DB path line: $HOME expanded, no raw expression
	dbLine := lines[2]
	if strings.Contains(dbLine, "$HOME") {
		t.Errorf("DB path line still contains unexpanded $HOME: %q", dbLine)
	}
	if strings.Contains(dbLine, "!`") {
		t.Errorf("DB path line still contains raw expression: %q", dbLine)
	}
	if !strings.Contains(dbLine, ".lsbot/calendar/calendar.db") {
		t.Errorf("DB path line missing expected path: %q", dbLine)
	}
}

func TestExpandInlineShell_PreservesMarkdown(t *testing.T) {
	input := "# Heading\n\n```bash\nsqlite3 $DB \"SELECT ...\"\n```\n\n- item !`echo injected`"
	out := ExpandInlineShell(input)

	if !strings.Contains(out, "# Heading") {
		t.Error("heading was stripped")
	}
	if !strings.Contains(out, "```bash") {
		t.Error("code block was stripped")
	}
	if !strings.Contains(out, "injected") {
		t.Error("inline shell was not substituted")
	}
	if strings.Contains(out, "!`echo injected`") {
		t.Error("raw expression was not replaced")
	}
}

func TestExpandInlineShell_NewlineInExpressionNotMatched(t *testing.T) {
	// Newline inside !`...` must not match (regex uses [^`\n])
	input := "!`echo line1\necho line2`"
	out := ExpandInlineShell(input)
	if out != input {
		t.Errorf("multi-line expression should not match, got %q", out)
	}
}
