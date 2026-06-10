package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFinishRecordsChangedContextAndCreatesAgentTask(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 2\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"finish"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("finish failed: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Recorded change") || !strings.Contains(stdout.String(), "Agent sync task") {
		t.Fatalf("finish should record the change and create an agent task, got:\n%s", stdout.String())
	}
	change := readOnlyChangeRecord(t, dir)
	assertContains(t, change.Modules, "chat")
	if len(change.Files) != 1 || change.Files[0].Path != "src/modules/chat/index.ts" {
		t.Fatalf("finish should record changed module files, got %#v", change.Files)
	}
	if _, err := os.Stat(filepath.Join(dir, ".doc", "tmp", "agent-sync.md")); err != nil {
		t.Fatalf("expected active agent sync task: %v", err)
	}
}

func TestFinishRecordsStagedModuleChanges(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initialize docx")

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 2\n")
	runGit(t, dir, "add", "src/modules/chat/index.ts")

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"finish"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("finish failed: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Recorded change") || !strings.Contains(stdout.String(), "Agent sync task") {
		t.Fatalf("finish should record staged module changes, got:\n%s", stdout.String())
	}
	change := readOnlyChangeRecord(t, dir)
	if len(change.Files) != 1 || change.Files[0].Path != "src/modules/chat/index.ts" {
		t.Fatalf("finish should record the staged module file, got %#v", change.Files)
	}
}

func TestFinishProposeCreatesProposalTaskWhenModuleChangesExist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 2\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"finish", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("finish --propose failed: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Recorded change") || !strings.Contains(stdout.String(), "Agent proposal task") {
		t.Fatalf("finish --propose should record changes and create a proposal task, got:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".doc", "tmp", "proposals-prompt.md")); err != nil {
		t.Fatalf("expected proposal prompt: %v", err)
	}
}

func TestFinishDoesNothingWhenNoChangedModuleFilesExist(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initialize docx")

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"finish"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("finish with no changes should succeed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	if strings.Contains(stdout.String(), "Recorded change") || strings.Contains(stdout.String(), "Agent sync task") {
		t.Fatalf("finish should not create sync artifacts when there are no changed module files, got:\n%s", stdout.String())
	}
	changeFiles, err := filepath.Glob(filepath.Join(dir, ".doc", "changes", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := withoutIndexJSON(changeFiles); len(got) != 0 {
		t.Fatalf("finish should not create change records without changed module files, got %#v", got)
	}
}

func TestFinishDoesNothingOutsideGitRepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"finish"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("finish outside a git repository should succeed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "No git repository found") {
		t.Fatalf("finish should explain why no sync ran, got:\n%s", stdout.String())
	}
}

func TestInstallAgentHookWritesCodexStopHook(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"install-agent-hook", "codex"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("install-agent-hook codex failed: %v\nstderr: %s", err, stderr.String())
	}

	hookBytes, err := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
	if err != nil {
		t.Fatalf("expected Codex hooks config: %v", err)
	}
	var config agentHookConfig
	if err := json.Unmarshal(hookBytes, &config); err != nil {
		t.Fatalf("Codex hooks config should be JSON: %v\n%s", err, string(hookBytes))
	}
	if !agentHookConfigHasCommand(config, "Stop", "docx finish") {
		t.Fatalf("Codex Stop hook should run docx finish, got:\n%s", string(hookBytes))
	}
	if !strings.Contains(stdout.String(), "Installed docx codex agent hook") {
		t.Fatalf("install should report the installed hook, got:\n%s", stdout.String())
	}
}

func TestInstallAgentHookCanRunFinishWithPropose(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"install-agent-hook", "codex", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("install-agent-hook codex --propose failed: %v\nstderr: %s", err, stderr.String())
	}

	hookBytes, err := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
	if err != nil {
		t.Fatalf("expected Codex hooks config: %v", err)
	}
	var config agentHookConfig
	if err := json.Unmarshal(hookBytes, &config); err != nil {
		t.Fatalf("Codex hooks config should be JSON: %v\n%s", err, string(hookBytes))
	}
	if !agentHookConfigHasCommand(config, "Stop", "docx finish --propose") {
		t.Fatalf("Codex Stop hook should run docx finish --propose, got:\n%s", string(hookBytes))
	}
}

func TestInstallAgentHookWritesClaudeStopHook(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"install-agent-hook", "claude"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("install-agent-hook claude failed: %v\nstderr: %s", err, stderr.String())
	}

	hookBytes, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("expected Claude settings config: %v", err)
	}
	var config agentHookConfig
	if err := json.Unmarshal(hookBytes, &config); err != nil {
		t.Fatalf("Claude settings config should be JSON: %v\n%s", err, string(hookBytes))
	}
	if !agentHookConfigHasCommand(config, "Stop", "docx finish") {
		t.Fatalf("Claude Stop hook should run docx finish, got:\n%s", string(hookBytes))
	}
	if !strings.Contains(stdout.String(), "Installed docx claude agent hook") {
		t.Fatalf("install should report the installed hook, got:\n%s", stdout.String())
	}
}

func TestInstallAgentHookPreservesExistingCodexHooksAndIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".codex/hooks.json", `{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "echo keep-user-hook"
          }
        ]
      }
    ]
  }
}
`)

	for i := 0; i < 2; i++ {
		stdout.Reset()
		stderr.Reset()
		if err := Run([]string{"install-agent-hook", "codex"}, dir, &stdout, &stderr); err != nil {
			t.Fatalf("install-agent-hook codex run %d failed: %v\nstderr: %s", i+1, err, stderr.String())
		}
	}

	hookBytes, err := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
	if err != nil {
		t.Fatalf("expected Codex hooks config: %v", err)
	}
	var config agentHookConfig
	if err := json.Unmarshal(hookBytes, &config); err != nil {
		t.Fatalf("Codex hooks config should be JSON: %v\n%s", err, string(hookBytes))
	}
	if !agentHookConfigHasCommand(config, "UserPromptSubmit", "echo keep-user-hook") {
		t.Fatalf("install should preserve existing user hooks, got:\n%s", string(hookBytes))
	}
	if count := agentHookConfigCommandCount(config, "Stop", "docx finish"); count != 1 {
		t.Fatalf("install should add one docx finish hook, got %d in:\n%s", count, string(hookBytes))
	}
}

func TestInstallAgentHookPreservesExistingClaudeSettings(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".claude/settings.json", `{
  "permissions": {
    "allow": ["Bash(echo *)"]
  },
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "echo keep-claude-hook",
            "extra": "keep-extra-field"
          }
        ]
      }
    ]
  }
}
`)

	if err := Run([]string{"install-agent-hook", "claude"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("install-agent-hook claude failed: %v\nstderr: %s", err, stderr.String())
	}

	hookBytes, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("expected Claude settings config: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(hookBytes, &raw); err != nil {
		t.Fatalf("Claude settings config should be JSON: %v\n%s", err, string(hookBytes))
	}
	if _, ok := raw["permissions"]; !ok {
		t.Fatalf("install should preserve top-level Claude settings, got:\n%s", string(hookBytes))
	}
	var config agentHookConfig
	if err := json.Unmarshal(hookBytes, &config); err != nil {
		t.Fatalf("Claude settings config should match hook shape: %v\n%s", err, string(hookBytes))
	}
	if !agentHookConfigHasCommand(config, "UserPromptSubmit", "echo keep-claude-hook") {
		t.Fatalf("install should preserve existing Claude hooks, got:\n%s", string(hookBytes))
	}
	if !strings.Contains(string(hookBytes), "keep-extra-field") {
		t.Fatalf("install should preserve existing hook fields, got:\n%s", string(hookBytes))
	}
	if !agentHookConfigHasCommand(config, "Stop", "docx finish") {
		t.Fatalf("Claude Stop hook should run docx finish, got:\n%s", string(hookBytes))
	}
}

type agentHookConfig struct {
	Hooks map[string][]agentHookMatcher `json:"hooks"`
}

type agentHookMatcher struct {
	Matcher string             `json:"matcher,omitempty"`
	Hooks   []agentHookCommand `json:"hooks"`
}

type agentHookCommand struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

func agentHookConfigHasCommand(config agentHookConfig, event string, command string) bool {
	return agentHookConfigCommandCount(config, event, command) > 0
}

func agentHookConfigCommandCount(config agentHookConfig, event string, command string) int {
	count := 0
	for _, matcher := range config.Hooks[event] {
		for _, hook := range matcher.Hooks {
			if hook.Type != "command" {
				continue
			}
			if hook.Command == command {
				count++
				continue
			}
			if hook.Command == "docx" && strings.Join(hook.Args, " ") == strings.TrimPrefix(command, "docx ") {
				count++
			}
		}
	}
	return count
}
