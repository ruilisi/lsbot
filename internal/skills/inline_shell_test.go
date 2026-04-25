package skills

import (
	"testing"
)

func TestExpandInlineShell(t *testing.T) {
	out := ExpandInlineShell("os: !`uname -s`")
	if out == "os: !`uname -s`" || out == "os: " {
		t.Errorf("expected OS name, got: %q", out)
	}
	t.Logf("result: %s", out)

	// Error case — should silently return empty
	out2 := ExpandInlineShell("bad: !`nonexistent_cmd_xyz`")
	if out2 != "bad: " {
		t.Errorf("expected empty on error, got: %q", out2)
	}

	// No expressions — passthrough
	out3 := ExpandInlineShell("no shell here")
	if out3 != "no shell here" {
		t.Errorf("passthrough failed: %q", out3)
	}
}
