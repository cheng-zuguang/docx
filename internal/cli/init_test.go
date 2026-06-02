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
	for _, expected := range []string{
		`"ai": {`,
		`"provider": ""`,
		`"command": ""`,
		`"timeoutSeconds": 120`,
		`"contextSources": [`,
		`"docx"`,
		`"output": "proposal-json"`,
	} {
		if !strings.Contains(config, expected) {
			t.Fatalf("config should include default AI field %s, got:\n%s", expected, config)
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
