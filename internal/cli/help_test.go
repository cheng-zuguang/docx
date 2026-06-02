package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpListsMVPCommands(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"--help"}, t.TempDir(), &stdout, &stderr); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	help := stdout.String()
	for _, command := range []string{"init", "scan", "update", "index", "doctor", "proposals"} {
		if !strings.Contains(help, command) {
			t.Fatalf("help should list %q, got:\n%s", command, help)
		}
	}
}
