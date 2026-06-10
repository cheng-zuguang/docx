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
	for _, command := range []string{"init", "sync", "finish", "scan", "update", "apply", "index", "doctor", "proposals", "install-agent-hook"} {
		if !strings.Contains(help, command) {
			t.Fatalf("help should list %q, got:\n%s", command, help)
		}
	}
	for _, expected := range []string{"Usage:", "Commands:", "Options:", "docx <command> --help"} {
		if !strings.Contains(help, expected) {
			t.Fatalf("top-level help should include %q, got:\n%s", expected, help)
		}
	}
}

func TestSubcommandHelpDescribesOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args     []string
		expected []string
	}{
		{
			args: []string{"init", "--help"},
			expected: []string{
				"Usage:",
				"docx init",
				"--accept-candidates",
				"--summarize",
				"--interactive",
			},
		},
		{
			args: []string{"update", "--help"},
			expected: []string{
				"Usage:",
				"docx update",
				"--staged",
				"--changed",
				"--since <ref>",
				"--propose",
				"docx finish",
			},
		},
		{
			args: []string{"sync", "--help"},
			expected: []string{
				"Usage:",
				"docx sync",
				"active-agent",
				"deterministic module facts",
				"semantic follow-up",
			},
		},
		{
			args: []string{"finish", "--help"},
			expected: []string{
				"Usage:",
				"docx finish",
				"end-of-turn",
				"docx sync",
			},
		},
		{
			args: []string{"proposals", "--help"},
			expected: []string{
				"Usage:",
				"docx proposals",
				"list",
				"show <id>",
				"accept <id>",
				"--target <path>",
			},
		},
		{
			args: []string{"apply", "--help"},
			expected: []string{
				"Usage:",
				"docx apply init",
				"docx apply proposals",
				"--stdin",
			},
		},
		{
			args: []string{"install-hook", "--help"},
			expected: []string{
				"Usage:",
				"docx install-hook",
				"pre-commit",
				"--propose",
			},
		},
		{
			args: []string{"install-agent-hook", "--help"},
			expected: []string{
				"Usage:",
				"docx install-agent-hook",
				"codex",
				"claude",
				"Stop",
				"docx finish",
				"--propose",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			if err := Run(tt.args, t.TempDir(), &stdout, &stderr); err != nil {
				t.Fatalf("help failed: %v", err)
			}
			help := stdout.String()
			for _, expected := range tt.expected {
				if !strings.Contains(help, expected) {
					t.Fatalf("help should include %q, got:\n%s", expected, help)
				}
			}
		})
	}
}

func TestUpdateHelpMentionsProposalTaskButNotRemovedAIOptions(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"update", "--help"}, t.TempDir(), &stdout, &stderr); err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "--propose") {
		t.Fatalf("update help should include active-agent --propose, got:\n%s", stdout.String())
	}
	for _, unexpected := range []string{"--ai", "--ai-command", "--auto", "DOCX_AI_UPDATE_CMD"} {
		if strings.Contains(stdout.String(), unexpected) {
			t.Fatalf("update help should not make %s part of the default path, got:\n%s", unexpected, stdout.String())
		}
	}
}

func TestAIRootCommandIsRemoved(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := Run([]string{"ai"}, t.TempDir(), &stdout, &stderr)
	if err == nil {
		t.Fatalf("docx ai should fail after the command was removed")
	}
	if !strings.Contains(err.Error(), `unknown command "ai"`) {
		t.Fatalf("docx ai should report an unknown command, got %v", err)
	}
}
