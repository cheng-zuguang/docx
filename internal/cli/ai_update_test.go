package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpdateAIGeneratesProposalWithoutChangingSemanticMemory(t *testing.T) {
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
	if err := Run([]string{"update", "--changed", "--ai"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --ai failed: %v\nstderr: %s", err, stderr.String())
	}

	change := readOnlyChangeRecord(t, dir)
	if len(change.Proposals) != 1 {
		t.Fatalf("change should reference generated proposal, got %#v", change.Proposals)
	}
	proposalID := strings.TrimSuffix(filepath.Base(change.Proposals[0]), ".json")
	proposalBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "proposals", proposalID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal := string(proposalBytes)
	for _, expected := range []string{`"status": "pending"`, `"type": "module-summary"`, `"source": "ai:provider-agnostic"`, `.doc/modules/chat.json`} {
		if !strings.Contains(proposal, expected) {
			t.Fatalf("proposal should contain %q, got:\n%s", expected, proposal)
		}
	}
	afterModule, err := os.ReadFile(modulePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeModule) != string(afterModule) {
		t.Fatalf("update --ai should not directly mutate semantic module memory\nbefore:\n%s\nafter:\n%s", string(beforeModule), string(afterModule))
	}
}

func TestUpdateAICommandUsesLocalToolOutputAsProposal(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell local AI fixture is POSIX-only")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	aiCommand := filepath.Join(dir, "local-ai.sh")
	inputPath := filepath.Join(dir, "ai-input.json")
	if err := os.WriteFile(aiCommand, []byte(`#!/bin/sh
cat > ai-input.json
cat <<'JSON'
[
  {
    "schemaVersion": "1.0",
    "id": "local-ai-chat-summary",
    "type": "module-summary",
    "status": "pending",
    "source": "ai:local-command",
    "evidence": [
      {"path": "src/modules/chat/index.ts", "reason": "Local AI reviewed the changed chat module."}
    ],
    "suggestedTarget": ".doc/modules/chat.json",
    "suggestedPatch": {
      "purpose": "Owns chat conversations after local AI review.",
      "ownedConcepts": ["conversation"],
      "nonGoals": []
    }
  }
]
JSON
`), 0o755); err != nil {
		t.Fatal(err)
	}

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 2\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--ai", "--ai-command", aiCommand}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --ai --ai-command failed: %v\nstderr: %s", err, stderr.String())
	}

	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	var input struct {
		SchemaVersion string `json:"schemaVersion"`
		Source        string `json:"source"`
		Modules       []struct {
			Name      string   `json:"name"`
			Paths     []string `json:"paths"`
			RiskRules []string `json:"riskRules"`
		} `json:"modules"`
		Files []changeFile `json:"files"`
	}
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		t.Fatalf("local AI input should be JSON, got %v:\n%s", err, string(inputBytes))
	}
	if input.SchemaVersion != schemaVersion || input.Source != "git:changed" {
		t.Fatalf("unexpected local AI input metadata: %#v", input)
	}
	if len(input.Modules) != 1 || input.Modules[0].Name != "chat" || len(input.Modules[0].Paths) == 0 {
		t.Fatalf("local AI input should include module context, got %#v", input.Modules)
	}
	if len(input.Files) != 1 || input.Files[0].Path != "src/modules/chat/index.ts" {
		t.Fatalf("local AI input should include changed files, got %#v", input.Files)
	}

	change := readOnlyChangeRecord(t, dir)
	assertContains(t, change.Proposals, ".doc/proposals/local-ai-chat-summary.json")
	proposalBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "proposals", "local-ai-chat-summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposal := string(proposalBytes)
	for _, expected := range []string{`"source": "ai:local-command"`, `"status": "pending"`, `"suggestedTarget": ".doc/modules/chat.json"`} {
		if !strings.Contains(proposal, expected) {
			t.Fatalf("local AI proposal should contain %q, got:\n%s", expected, proposal)
		}
	}
}

func TestUpdateAICommandRejectsInvalidOutputWithoutWritingProposal(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell local AI fixture is POSIX-only")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	beforeModule, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "chat.json"))
	if err != nil {
		t.Fatal(err)
	}
	aiCommand := filepath.Join(dir, "bad-ai.sh")
	if err := os.WriteFile(aiCommand, []byte("#!/bin/sh\necho 'not json'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 3\n")
	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"update", "--changed", "--ai", "--ai-command", aiCommand}, dir, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "expected proposal JSON") {
		t.Fatalf("update should reject invalid local AI output, got: %v", err)
	}

	proposals, err := filepath.Glob(filepath.Join(dir, ".doc", "proposals", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	proposals = withoutIndexJSON(proposals)
	if len(proposals) != 0 {
		t.Fatalf("invalid local AI output should not write proposals, got %#v", proposals)
	}
	afterModule, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "chat.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeModule) != string(afterModule) {
		t.Fatalf("invalid local AI output should not mutate module memory\nbefore:\n%s\nafter:\n%s", string(beforeModule), string(afterModule))
	}
}

func TestUpdateAICommandAcceptsSingleProposalObject(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell local AI fixture is POSIX-only")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	aiCommand := filepath.Join(dir, "single-ai.sh")
	if err := os.WriteFile(aiCommand, []byte(`#!/bin/sh
cat >/dev/null
cat <<'JSON'
{
  "schemaVersion": "1.0",
  "id": "single-local-ai-risk",
  "type": "risk-rule",
  "status": "pending",
  "source": "ai:local-command",
  "evidence": [
    {"path": "src/modules/chat/index.ts", "reason": "Local AI found ordering risk."}
  ],
  "suggestedTarget": ".doc/modules/chat.json",
  "suggestedPatch": {
    "rule": "Preserve chat message ordering."
  }
}
JSON
`), 0o755); err != nil {
		t.Fatal(err)
	}

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 4\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--ai", "--ai-command", aiCommand}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --ai --ai-command failed: %v\nstderr: %s", err, stderr.String())
	}

	change := readOnlyChangeRecord(t, dir)
	assertContains(t, change.Proposals, ".doc/proposals/single-local-ai-risk.json")
	if _, err := os.Stat(filepath.Join(dir, ".doc", "proposals", "single-local-ai-risk.json")); err != nil {
		t.Fatalf("single proposal object should be written: %v", err)
	}
}

func TestUpdateAIReadsLocalCommandFromDocxConfig(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell local AI fixture is POSIX-only")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	aiCommand := filepath.Join(dir, "configured-ai.sh")
	if err := os.WriteFile(aiCommand, []byte(`#!/bin/sh
cat >/dev/null
cat <<'JSON'
{
  "schemaVersion": "1.0",
  "id": "configured-local-ai-summary",
  "type": "module-summary",
  "status": "pending",
  "source": "ai:local-command",
  "evidence": [
    {"path": "src/modules/chat/index.ts", "reason": "Configured local AI reviewed the change."}
  ],
  "suggestedTarget": ".doc/modules/chat.json",
  "suggestedPatch": {
    "purpose": "Configured local AI summary."
  }
}
JSON
`), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, dir, ".docx.json", `{
  "schemaVersion": "1.0",
  "contextDir": ".doc",
  "contextSchemaVersion": "1.0",
  "entryFiles": ["AGENTS.md"],
  "ai": {
    "provider": "local-command",
    "command": "`+aiCommand+`",
    "timeoutSeconds": 120,
    "contextSources": ["docx", "codegraph"],
    "output": "proposal-json"
  }
}`)

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 5\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--ai"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --ai should use configured local command: %v\nstderr: %s", err, stderr.String())
	}

	change := readOnlyChangeRecord(t, dir)
	assertContains(t, change.Proposals, ".doc/proposals/configured-local-ai-summary.json")
}

func TestUpdateAICommandFlagOverridesDocxConfig(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell local AI fixture is POSIX-only")
	}

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	configuredCommand := filepath.Join(dir, "configured-ai.sh")
	if err := os.WriteFile(configuredCommand, []byte(`#!/bin/sh
cat >/dev/null
cat <<'JSON'
{
  "schemaVersion": "1.0",
  "id": "should-not-be-used",
  "type": "module-summary",
  "status": "pending",
  "source": "ai:local-command",
  "evidence": [{"path": "src/modules/chat/index.ts", "reason": "configured command"}],
  "suggestedTarget": ".doc/modules/chat.json",
  "suggestedPatch": {"purpose": "configured"}
}
JSON
`), 0o755); err != nil {
		t.Fatal(err)
	}
	overrideCommand := filepath.Join(dir, "override-ai.sh")
	if err := os.WriteFile(overrideCommand, []byte(`#!/bin/sh
cat >/dev/null
cat <<'JSON'
{
  "schemaVersion": "1.0",
  "id": "override-local-ai-summary",
  "type": "module-summary",
  "status": "pending",
  "source": "ai:local-command",
  "evidence": [{"path": "src/modules/chat/index.ts", "reason": "override command"}],
  "suggestedTarget": ".doc/modules/chat.json",
  "suggestedPatch": {"purpose": "override"}
}
JSON
`), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, dir, ".docx.json", `{
  "schemaVersion": "1.0",
  "contextDir": ".doc",
  "contextSchemaVersion": "1.0",
  "entryFiles": ["AGENTS.md"],
  "ai": {
    "provider": "local-command",
    "command": "`+configuredCommand+`",
    "output": "proposal-json"
  }
}`)

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 6\n")
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed", "--ai", "--ai-command", overrideCommand}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update --ai should use CLI override command: %v\nstderr: %s", err, stderr.String())
	}

	change := readOnlyChangeRecord(t, dir)
	assertContains(t, change.Proposals, ".doc/proposals/override-local-ai-summary.json")
	if _, err := os.Stat(filepath.Join(dir, ".doc", "proposals", "should-not-be-used.json")); !os.IsNotExist(err) {
		t.Fatalf("configured command should not run when --ai-command is provided")
	}
}
