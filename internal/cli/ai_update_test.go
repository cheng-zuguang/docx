package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateProposalTaskCreatesProposalTaskWithoutChangingSemanticMemory(t *testing.T) {
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
	modulePath := filepath.Join(dir, ".doc", "modules", "chat.json")
	beforeModule, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 2\n")

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --propose failed: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Agent proposal task") {
		t.Fatalf("update --propose should explain the proposal task, got:\n%s", stdout.String())
	}
	for _, path := range []string{
		".doc/tmp/proposals-input.json",
		".doc/tmp/proposals-prompt.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected proposal task file %s: %v", path, err)
		}
	}

	inputBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "tmp", "proposals-input.json"))
	if err != nil {
		t.Fatal(err)
	}
	var input struct {
		SchemaVersion string `json:"schemaVersion"`
		ChangeID      string `json:"changeId"`
		Source        string `json:"source"`
		Modules       []struct {
			Name      string   `json:"name"`
			Paths     []string `json:"paths"`
			RiskRules []string `json:"riskRules"`
		} `json:"modules"`
		Files []changeFile `json:"files"`
	}
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		t.Fatalf("agent update input should be JSON, got %v:\n%s", err, string(inputBytes))
	}
	if input.SchemaVersion != schemaVersion || input.ChangeID == "" || input.Source != "git:changed" {
		t.Fatalf("unexpected AI update input metadata: %#v", input)
	}
	if len(input.Modules) != 1 || input.Modules[0].Name != "chat" || len(input.Modules[0].Paths) == 0 {
		t.Fatalf("agent update input should include module context, got %#v", input.Modules)
	}
	if len(input.Files) != 1 || input.Files[0].Path != "src/modules/chat/index.ts" {
		t.Fatalf("agent update input should include changed files, got %#v", input.Files)
	}

	change := readOnlyChangeRecord(t, dir)
	if len(change.Proposals) != 0 {
		t.Fatalf("proposal task should not link proposals until apply proposals, got %#v", change.Proposals)
	}
	afterModule, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeModule) != string(afterModule) {
		t.Fatalf("update --propose should not directly mutate semantic module memory\nbefore:\n%s\nafter:\n%s", string(beforeModule), string(afterModule))
	}
}

func TestUpdateRejectsRemovedAIOption(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, option := range []string{"--ai", "--handoff"} {
		option := option
		t.Run(option, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run([]string{"update", "--changed", option}, dir, &stdout, &stderr)
			if err == nil {
				t.Fatalf("update %s should fail after the option was removed", option)
			}
			if !strings.Contains(err.Error(), `unknown option "`+option+`"`) {
				t.Fatalf("update %s should report an unknown option, got %v", option, err)
			}
		})
	}
}

func TestAIUpdateApplyWritesProposalsAndLinksChange(t *testing.T) {
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
	if err := Run([]string{"update", "--changed", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --propose failed: %v\nstderr: %s", err, stderr.String())
	}
	change := readOnlyChangeRecord(t, dir)

	writeFixtureFile(t, dir, ".doc/tmp/proposals-output.json", `{
  "schemaVersion": "1.0",
  "changeId": "`+change.ID+`",
  "proposals": [
    {
      "schemaVersion": "1.0",
      "id": "active-ai-chat-summary",
      "type": "module-summary",
      "status": "pending",
      "source": "ai:active-agent",
      "evidence": [
        {"path": "src/modules/chat/index.ts", "reason": "Active AI reviewed the changed chat module."}
      ],
      "suggestedTarget": ".doc/modules/chat.json",
      "suggestedPatch": {
        "purpose": "Owns chat conversations after active AI review.",
        "ownedConcepts": ["conversation"],
        "nonGoals": []
      }
    }
  ]
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"apply", "proposals", ".doc/tmp/proposals-output.json"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("apply proposals failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Applied proposal output: proposals=1") {
		t.Fatalf("apply should report proposal count, got:\n%s", stdout.String())
	}

	change = readOnlyChangeRecord(t, dir)
	assertContains(t, change.Proposals, ".doc/proposals/active-ai-chat-summary.json")
	proposalBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "proposals", "active-ai-chat-summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal := string(proposalBytes)
	for _, expected := range []string{`"source": "ai:active-agent"`, `"status": "pending"`, `"suggestedTarget": ".doc/modules/chat.json"`} {
		if !strings.Contains(proposal, expected) {
			t.Fatalf("AI proposal should contain %q, got:\n%s", expected, proposal)
		}
	}
}

func TestApplyProposalsWritesProposalsAndLinksChange(t *testing.T) {
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
	if err := Run([]string{"sync", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("sync --propose failed: %v\nstderr: %s", err, stderr.String())
	}
	change := readOnlyChangeRecord(t, dir)

	writeFixtureFile(t, dir, ".doc/tmp/proposals-output.json", `{
  "schemaVersion": "1.0",
  "changeId": "`+change.ID+`",
  "proposals": [
    {
      "schemaVersion": "1.0",
      "id": "active-agent-chat-summary",
      "type": "module-summary",
      "status": "pending",
      "source": "agent:active",
      "evidence": [
        {"path": "src/modules/chat/index.ts", "reason": "The active agent reviewed the changed chat module."}
      ],
      "suggestedTarget": ".doc/modules/chat.json",
      "suggestedPatch": {
        "purpose": "Owns chat conversations after active agent review.",
        "ownedConcepts": ["conversation"],
        "nonGoals": []
      }
    }
  ]
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"apply", "proposals", ".doc/tmp/proposals-output.json"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("apply proposals failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Applied proposal output: proposals=1") {
		t.Fatalf("apply should report proposal count, got:\n%s", stdout.String())
	}

	change = readOnlyChangeRecord(t, dir)
	assertContains(t, change.Proposals, ".doc/proposals/active-agent-chat-summary.json")
}

func TestAIUpdateApplyRejectsInvalidOutputWithoutWritingProposal(t *testing.T) {
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
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 3\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --propose failed: %v\nstderr: %s", err, stderr.String())
	}
	change := readOnlyChangeRecord(t, dir)
	writeFixtureFile(t, dir, ".doc/tmp/proposals-output.json", `{
  "schemaVersion": "1.0",
  "changeId": "`+change.ID+`",
  "proposals": [
    {
      "schemaVersion": "1.0",
      "id": "bad-active-ai",
      "type": "module-summary",
      "status": "accepted",
      "source": "ai:active-agent",
      "evidence": [{"path": "src/modules/chat/index.ts", "reason": "invalid"}],
      "suggestedTarget": ".doc/modules/chat.json",
      "suggestedPatch": {"purpose": "invalid"}
    }
  ]
}`)

	stdout.Reset()
	stderr.Reset()
	err := Run([]string{"apply", "proposals", ".doc/tmp/proposals-output.json"}, dir, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "missing required fields or is not pending") {
		t.Fatalf("apply proposals should reject invalid proposals, got: %v", err)
	}

	proposals, err := filepath.Glob(filepath.Join(dir, ".doc", "proposals", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposals = withoutIndexJSON(proposals)
	if len(proposals) != 0 {
		t.Fatalf("invalid agent output should not write proposals, got %#v", proposals)
	}
	change = readOnlyChangeRecord(t, dir)
	if len(change.Proposals) != 0 {
		t.Fatalf("invalid agent output should not link proposals, got %#v", change.Proposals)
	}
}

func TestAIUpdateApplyCanReadStdin(t *testing.T) {
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
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 4\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --propose failed: %v\nstderr: %s", err, stderr.String())
	}
	change := readOnlyChangeRecord(t, dir)

	stdin := strings.NewReader(`{
  "schemaVersion": "1.0",
  "changeId": "` + change.ID + `",
  "proposals": {
    "schemaVersion": "1.0",
    "id": "stdin-active-ai-risk",
    "type": "risk-rule",
    "status": "pending",
    "source": "ai:active-agent",
    "evidence": [{"path": "src/modules/chat/index.ts", "reason": "Active AI found an ordering risk."}],
    "suggestedTarget": ".doc/modules/chat.json",
    "suggestedPatch": {"rule": "Preserve chat message ordering."}
  }
}`)

	stdout.Reset()
	stderr.Reset()
	if err := RunWithInput([]string{"apply", "proposals", "--stdin"}, dir, stdin, &stdout, &stderr); err != nil {
		t.Fatalf("apply proposals --stdin failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	change = readOnlyChangeRecord(t, dir)
	assertContains(t, change.Proposals, ".doc/proposals/stdin-active-ai-risk.json")
}

func TestUpdateAIIgnoresConfiguredCommandAndCreatesProposalTask(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	script := filepath.Join(dir, "mock-proposals.sh")
	writeFixtureFile(t, dir, "mock-proposals.sh", "#!/bin/sh\ninput=$(cat)\nchange_id=$(printf '%s' \"$input\" | sed -n 's/.*\"changeId\":\"\\([^\"]*\\)\".*/\\1/p')\ncat <<EOF\n{\"schemaVersion\":\"1.0\",\"changeId\":\"$change_id\",\"proposals\":[{\"schemaVersion\":\"1.0\",\"id\":\"auto-chat-summary\",\"type\":\"module-summary\",\"status\":\"pending\",\"source\":\"ai:auto\",\"evidence\":[{\"path\":\"src/modules/chat/index.ts\",\"reason\":\"auto runner\"}],\"suggestedTarget\":\".doc/modules/chat.json\",\"suggestedPatch\":{\"purpose\":\"Auto chat summary.\",\"ownedConcepts\":[\"conversation\"],\"nonGoals\":[]}}]}\nEOF\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCX_AI_UPDATE_CMD", script)

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 5\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --propose failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Agent proposal task") {
		t.Fatalf("expected proposal task output, got:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".doc", "tmp", "proposals-prompt.md")); err != nil {
		t.Fatalf("agent proposal task should create prompt even when AI command is configured: %v", err)
	}
	change := readOnlyChangeRecord(t, dir)
	if len(change.Proposals) != 0 {
		t.Fatalf("configured AI command should not write proposals, got %#v", change.Proposals)
	}
}

func TestUpdateAIProposalTaskDoesNotReportConfiguredCommandFailure(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")
	t.Setenv("DOCX_AI_UPDATE_CMD", filepath.Join(dir, "missing-ai-command.sh"))

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 6\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --propose failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	if strings.Contains(stdout.String(), "AI auto-update unavailable") {
		t.Fatalf("agent proposal task should not try or report configured commands, got:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".doc", "tmp", "proposals-prompt.md")); err != nil {
		t.Fatalf("proposal task should create prompt: %v", err)
	}
	change := readOnlyChangeRecord(t, dir)
	if len(change.Proposals) != 0 {
		t.Fatalf("fallback proposal task should not auto-link proposals, got %#v", change.Proposals)
	}
}
