package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesContextAndPreservesAgentFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	agentsPath := filepath.Join(dir, "AGENTS.md")
	original := "# Team Rules\n\nKeep this line.\n"
	if err := os.WriteFile(agentsPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	for _, path := range []string{
		".docx.json",
		".doc/index.json",
		".doc/project.json",
		".doc/capabilities.json",
		".doc/README.md",
		".doc/rules/agent.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	configBytes, err := os.ReadFile(filepath.Join(dir, ".docx.json"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(configBytes)
	if !strings.Contains(config, `"contextDir": ".doc"`) {
		t.Fatalf("config should record default context dir, got:\n%s", config)
	}
	for _, unexpected := range []string{`"ai": {`, `"command"`, `"provider"`, `"output"`} {
		if strings.Contains(config, unexpected) {
			t.Fatalf("config should not include legacy AI command field %s, got:\n%s", unexpected, config)
		}
	}

	agentsBytes, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	agents := string(agentsBytes)
	if !strings.Contains(agents, original) {
		t.Fatalf("init should preserve existing AGENTS.md content, got:\n%s", agents)
	}
	if !strings.Contains(agents, "<!-- docx:start -->") || !strings.Contains(agents, "<!-- docx:end -->") {
		t.Fatalf("init should insert managed docx block, got:\n%s", agents)
	}
	for _, expected := range []string{
		"When an agent lifecycle hook is installed, let it run `docx finish`; otherwise run `docx sync` before finishing.",
		"Keep `AGENTS.md` short; use `.doc/rules/agent.md` for detailed behavior.",
		"Use change records for audit trails, module `recentChanges`, proposal evidence, and future AI context.",
	} {
		if !strings.Contains(agents, expected) {
			t.Fatalf("managed AGENTS.md block should include update protocol %q, got:\n%s", expected, agents)
		}
	}
	agentRulesBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "rules", "agent.md"))
	if err != nil {
		t.Fatal(err)
	}
	agentRules := string(agentRulesBytes)
	for _, expected := range []string{
		"When an agent lifecycle hook is installed, let it run `docx finish`; otherwise run `docx sync` before finishing.",
		"`docx finish` is a safe lifecycle-hook wrapper around `docx sync`.",
		"Read affected module files resolved from `moduleMap` before editing or summarizing module behavior.",
		"`docx sync` records changed files, updates deterministic module context, and writes an active-agent task",
		"Deterministic facts may be refreshed directly; semantic memory requires proposals unless the user confirms the edit.",
		"Use change records for audit trails, module `recentChanges`, proposal evidence, and future AI context.",
		"Prefer finer modules around real workflows when a coarse module hides unrelated concepts.",
	} {
		if !strings.Contains(agentRules, expected) {
			t.Fatalf("agent rules should include %q, got:\n%s", expected, agentRules)
		}
	}
	if strings.Contains(agentRules, "DOCX_AI_UPDATE_CMD") || strings.Contains(agentRules, "update --ai") {
		t.Fatalf("agent rules should not make configured AI scripts the default path, got:\n%s", agentRules)
	}
	if strings.Contains(agentRules, "rebuilds indexes") {
		t.Fatalf("agent rules should not claim docx sync rebuilds indexes, got:\n%s", agentRules)
	}
	if !strings.Contains(agentRules, "When an agent lifecycle hook is installed, let it run `docx finish`; otherwise run `docx sync` before finishing.") {
		t.Fatalf("agent rules should include update protocol, got:\n%s", agentRules)
	}

	before := agents
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("second init failed: %v\nstderr: %s", err, stderr.String())
	}
	agentsBytes, err = os.ReadFile(agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	after := string(agentsBytes)
	if before != after {
		t.Fatalf("init should be idempotent for managed AGENTS.md block\nbefore:\n%s\nafter:\n%s", before, after)
	}

	gitignoreBytes, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	gitignore := string(gitignoreBytes)
	for _, expected := range []string{".doc/.cache/", ".doc/local/", ".doc/tmp/"} {
		if !strings.Contains(gitignore, expected) {
			t.Fatalf(".gitignore should include %s, got:\n%s", expected, gitignore)
		}
	}
}

func TestInitSupportsCustomContextDirAndEntryFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	claudePath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Claude Rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--dir", ".context", "--entry", "CLAUDE.md"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	if _, err := os.Stat(filepath.Join(dir, ".context", "index.json")); err != nil {
		t.Fatalf("expected custom context index to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".doc", "index.json")); !os.IsNotExist(err) {
		t.Fatalf("default .doc context should not be created when --dir is used")
	}

	configBytes, err := os.ReadFile(filepath.Join(dir, ".docx.json"))
	if err != nil {
		t.Fatal(err)
	}
	config := string(configBytes)
	if !strings.Contains(config, `"contextDir": ".context"`) {
		t.Fatalf("config should record custom context dir, got:\n%s", config)
	}

	claudeBytes, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatal(err)
	}
	claude := string(claudeBytes)
	if !strings.Contains(claude, "Before work, read `.context/index.json`.") {
		t.Fatalf("entry file should reference custom context dir, got:\n%s", claude)
	}
}

func TestInitChoosesExistingAgentEntryFilesByDefault(t *testing.T) {
	t.Parallel()

	t.Run("creates only AGENTS when no entry file exists", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		var stdout, stderr bytes.Buffer
		if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
			t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
		}

		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err != nil {
			t.Fatalf("expected AGENTS.md to be created: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); !os.IsNotExist(err) {
			t.Fatalf("CLAUDE.md should not be created when no entry file exists")
		}
	})

	t.Run("updates only existing CLAUDE when AGENTS is absent", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		claudePath := filepath.Join(dir, "CLAUDE.md")
		original := "# Claude Rules\n"
		if err := os.WriteFile(claudePath, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		var stdout, stderr bytes.Buffer
		if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
			t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
		}

		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
			t.Fatalf("AGENTS.md should not be created when CLAUDE.md already exists")
		}
		claudeBytes, err := os.ReadFile(claudePath)
		if err != nil {
			t.Fatal(err)
		}
		claude := string(claudeBytes)
		if !strings.Contains(claude, original) || !strings.Contains(claude, "<!-- docx:start -->") {
			t.Fatalf("init should preserve and update existing CLAUDE.md, got:\n%s", claude)
		}
	})

	t.Run("updates both existing entry files", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		for _, entry := range []string{"AGENTS.md", "CLAUDE.md"} {
			if err := os.WriteFile(filepath.Join(dir, entry), []byte("# Existing\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		var stdout, stderr bytes.Buffer
		if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
			t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
		}

		for _, entry := range []string{"AGENTS.md", "CLAUDE.md"} {
			entryBytes, err := os.ReadFile(filepath.Join(dir, entry))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(entryBytes), "<!-- docx:start -->") {
				t.Fatalf("init should update existing %s, got:\n%s", entry, string(entryBytes))
			}
		}
	})
}

func TestInitNonInteractiveWritesUnconfirmedModuleCandidates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = true\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--non-interactive"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	indexBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		ModuleMap map[string]struct {
			Paths      []string `json:"paths"`
			Context    string   `json:"context"`
			Confidence string   `json:"confidence"`
		} `json:"moduleMap"`
	}
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatal(err)
	}
	chat, ok := index.ModuleMap["chat"]
	if !ok {
		t.Fatalf("expected chat module candidate in moduleMap, got:\n%s", string(indexBytes))
	}
	assertContains(t, chat.Paths, "src/modules/chat/**")
	if chat.Context != ".doc/modules/chat.json" {
		t.Fatalf("context = %q, want .doc/modules/chat.json", chat.Context)
	}
	if chat.Confidence != "candidate" {
		t.Fatalf("confidence = %q, want candidate", chat.Confidence)
	}

	if _, err := os.Stat(filepath.Join(dir, ".doc", "modules", "chat.json")); err != nil {
		t.Fatalf("expected candidate module context file: %v", err)
	}
}

func TestInitCanAcceptDetectedModuleCandidates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/features/billing/index.ts", "export const billing = true\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	indexBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		ModuleMap map[string]struct {
			Confidence string `json:"confidence"`
		} `json:"moduleMap"`
	}
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatal(err)
	}
	if index.ModuleMap["billing"].Confidence != "confirmed" {
		t.Fatalf("expected billing candidate to be confirmed, got:\n%s", string(indexBytes))
	}

	moduleBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "billing.json"))
	if err != nil {
		t.Fatal(err)
	}
	var module struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(moduleBytes, &module); err != nil {
		t.Fatal(err)
	}
	if module.Status != "confirmed" {
		t.Fatalf("module status = %q, want confirmed", module.Status)
	}
}

func TestInitAcceptCandidatesProposalTaskCreatesAgentProposalTask(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/features/billing/index.ts", "export const billing = true\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates", "--summarize"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init --summarize failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	for _, path := range []string{
		".doc/tmp/init-summary-input.json",
		".doc/tmp/init-summary-prompt.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected proposal task file %s: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), "Agent init summary task") {
		t.Fatalf("init should explain the proposal task, got:\n%s", stdout.String())
	}
	inputBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "tmp", "init-summary-input.json"))
	if err != nil {
		t.Fatal(err)
	}
	var input struct {
		SchemaVersion string `json:"schemaVersion"`
		Modules       []struct {
			Name string `json:"name"`
		} `json:"modules"`
	}
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		t.Fatal(err)
	}
	if input.SchemaVersion != schemaVersion || len(input.Modules) != 1 || input.Modules[0].Name != "billing" {
		t.Fatalf("unexpected proposal task input:\n%s", string(inputBytes))
	}
}

func TestInitRejectsRemovedAIAndAutoOptions(t *testing.T) {
	t.Parallel()

	for _, option := range []string{"--ai", "--auto", "--handoff"} {
		option := option
		t.Run(option, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			var stdout, stderr bytes.Buffer
			err := Run([]string{"init", option}, dir, &stdout, &stderr)
			if err == nil {
				t.Fatalf("init %s should fail after the option was removed", option)
			}
			if !strings.Contains(err.Error(), `unknown option "`+option+`"`) {
				t.Fatalf("init %s should report an unknown option, got %v", option, err)
			}
		})
	}
}

func TestInitAIIgnoresConfiguredAICommandAndCreatesProposalTask(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/features/billing/index.ts", "export const billing = true\n")
	script := filepath.Join(dir, "mock-init-summary.sh")
	writeFixtureFile(t, dir, "mock-init-summary.sh", "#!/bin/sh\ncat >/dev/null\ncat <<'EOF'\n{\"schemaVersion\":\"1.0\",\"project\":{\"summary\":\"auto init summary\"},\"modules\":[{\"name\":\"billing\",\"summary\":{\"purpose\":\"Auto billing summary.\",\"ownedConcepts\":[\"billing\"],\"nonGoals\":[]}}]}\nEOF\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCX_AI_INIT_CMD", script)

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates", "--summarize"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init --summarize failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Agent init summary task") {
		t.Fatalf("expected proposal task output, got:\n%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".doc", "tmp", "init-summary-prompt.md")); err != nil {
		t.Fatalf("agent proposal task should create prompt even when AI command is configured: %v", err)
	}
	projectBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(projectBytes), "auto init summary") {
		t.Fatalf("configured AI command should not be executed or auto-applied, got:\n%s", string(projectBytes))
	}
}

func TestAIInitApplyWritesSummariesFromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/features/billing/index.ts", "export const billing = true\n")
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates", "--summarize"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init --summarize failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	outputPath := filepath.Join(dir, ".doc", "tmp", "init-summary-output.json")
	writeFixtureFile(t, dir, ".doc/tmp/init-summary-output.json", `{
  "schemaVersion": "1.0",
  "project": {
    "summary": "A billing-focused TypeScript project."
  },
  "modules": [
    {
      "name": "billing",
      "summary": {
        "purpose": "Owns billing feature entrypoints.",
        "ownedConcepts": ["billing"],
        "nonGoals": []
      },
      "riskRules": ["must not be written during init"]
    }
  ]
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"apply", "init", outputPath}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("apply init failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Applied init summaries: project=updated modules=1") {
		t.Fatalf("apply should report updated summaries, got:\n%s", stdout.String())
	}

	projectBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	var project struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(projectBytes, &project); err != nil {
		t.Fatal(err)
	}
	if project.Summary != "A billing-focused TypeScript project." {
		t.Fatalf("project summary = %q", project.Summary)
	}

	moduleBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "billing.json"))
	if err != nil {
		t.Fatal(err)
	}
	var module struct {
		Summary   moduleSummary `json:"summary"`
		RiskRules []string      `json:"riskRules"`
	}
	if err := json.Unmarshal(moduleBytes, &module); err != nil {
		t.Fatal(err)
	}
	if module.Summary.Purpose != "Owns billing feature entrypoints." {
		t.Fatalf("module summary was not written, got:\n%s", string(moduleBytes))
	}
	assertContains(t, module.Summary.OwnedConcepts, "billing")
	if len(module.RiskRules) != 0 {
		t.Fatalf("apply init should not write riskRules, got: %#v", module.RiskRules)
	}
}

func TestAIInitApplyCanReadStdin(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/features/billing/index.ts", "export const billing = true\n")
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr:%s", err, stderr.String())
	}

	stdin := strings.NewReader(`{
  "schemaVersion": "1.0",
  "project": {
    "summary": "stdin project summary"
  },
  "modules": [
    {
      "name": "billing",
      "summary": {
        "purpose": "Owns stdin billing summary.",
        "ownedConcepts": ["billing"],
        "nonGoals": []
      }
    }
  ]
}`)

	stdout.Reset()
	stderr.Reset()
	if err := RunWithInput([]string{"apply", "init", "--stdin"}, dir, stdin, &stdout, &stderr); err != nil {
		t.Fatalf("apply init --stdin failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}
	moduleBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "billing.json"))
	if err != nil {
		t.Fatal(err)
	}
	var module struct {
		Summary moduleSummary `json:"summary"`
	}
	if err := json.Unmarshal(moduleBytes, &module); err != nil {
		t.Fatal(err)
	}
	if module.Summary.Purpose != "Owns stdin billing summary." {
		t.Fatalf("module summary was not decoded from stdin, got:\n%s", string(moduleBytes))
	}
}

func TestInitInteractiveCanAcceptIgnoreRenameAndMergeModuleCandidates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = true\n")
	writeFixtureFile(t, dir, "src/modules/billing/index.ts", "export const billing = true\n")
	writeFixtureFile(t, dir, "src/modules/temp/index.ts", "export const temp = true\n")
	writeFixtureFile(t, dir, "packages/web/index.ts", "export const web = true\n")
	writeFixtureFile(t, dir, "packages/api/index.ts", "export const api = true\n")

	input := strings.NewReader(strings.Join([]string{
		"accept chat",
		"rename billing payments",
		"ignore temp",
		"merge platform api,web",
		"done",
		"",
	}, "\n"))
	var stdout, stderr bytes.Buffer
	if err := RunWithInput([]string{"init", "--interactive"}, dir, input, &stdout, &stderr); err != nil {
		t.Fatalf("interactive init failed: %v\nstdout:%s\nstderr:%s", err, stdout.String(), stderr.String())
	}

	indexBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		ModuleMap map[string]struct {
			Paths      []string `json:"paths"`
			Context    string   `json:"context"`
			Confidence string   `json:"confidence"`
		} `json:"moduleMap"`
	}
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatal(err)
	}

	if index.ModuleMap["chat"].Confidence != "confirmed" {
		t.Fatalf("chat should be confirmed, got:\n%s", string(indexBytes))
	}
	if _, ok := index.ModuleMap["billing"]; ok {
		t.Fatalf("billing should have been renamed, got:\n%s", string(indexBytes))
	}
	if index.ModuleMap["payments"].Confidence != "confirmed" {
		t.Fatalf("payments should be confirmed, got:\n%s", string(indexBytes))
	}
	if _, ok := index.ModuleMap["temp"]; ok {
		t.Fatalf("temp should have been ignored, got:\n%s", string(indexBytes))
	}
	platform := index.ModuleMap["platform"]
	if platform.Confidence != "confirmed" {
		t.Fatalf("platform should be confirmed, got:\n%s", string(indexBytes))
	}
	assertContains(t, platform.Paths, "packages/api/**")
	assertContains(t, platform.Paths, "packages/web/**")

	for _, path := range []string{
		".doc/modules/chat.json",
		".doc/modules/payments.json",
		".doc/modules/platform.json",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".doc/modules/temp.json")); !os.IsNotExist(err) {
		t.Fatalf("ignored module should not have a module context file")
	}
}
