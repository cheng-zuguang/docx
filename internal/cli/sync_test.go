package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncChangedRecordsChangeAndCreatesAgentTask(t *testing.T) {
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
	if err := Run([]string{"sync"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("sync failed: %v\nstderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Recorded change") || !strings.Contains(stdout.String(), "Agent sync task") {
		t.Fatalf("sync should record the change and create an agent task, got:\n%s", stdout.String())
	}
	change := readOnlyChangeRecord(t, dir)
	assertContains(t, change.Modules, "chat")
	if len(change.Files) != 1 || change.Files[0].Path != "src/modules/chat/index.ts" {
		t.Fatalf("sync should record changed module files, got %#v", change.Files)
	}

	moduleBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "chat.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(moduleBytes), ".doc/changes/"+change.ID+".json") {
		t.Fatalf("sync should update module recentChanges, got:\n%s", string(moduleBytes))
	}

	taskBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "tmp", "agent-sync.md"))
	if err != nil {
		t.Fatalf("expected active agent sync task: %v", err)
	}
	task := string(taskBytes)
	for _, expected := range []string{change.ID, "src/modules/chat/index.ts", ".doc/modules/chat.json", "active agent should complete semantic context sync directly"} {
		if !strings.Contains(task, expected) {
			t.Fatalf("agent task should include %q, got:\n%s", expected, task)
		}
	}
}

func TestSyncProposeCreatesProposalTaskAfterChangeRecord(t *testing.T) {
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

	if !strings.Contains(stdout.String(), "Recorded change") || !strings.Contains(stdout.String(), "Agent proposal task") {
		t.Fatalf("sync --propose should record changes and create a proposal task, got:\n%s", stdout.String())
	}
	for _, path := range []string{
		".doc/tmp/proposals-input.json",
		".doc/tmp/proposals-prompt.md",
		".doc/tmp/agent-sync.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	taskBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "tmp", "agent-sync.md"))
	if err != nil {
		t.Fatal(err)
	}
	task := string(taskBytes)
	for _, expected := range []string{
		".doc/tmp/proposals-prompt.md",
		".doc/tmp/proposals-output.json",
		"docx apply proposals .doc/tmp/proposals-output.json",
	} {
		if !strings.Contains(task, expected) {
			t.Fatalf("proposal task should include %q, got:\n%s", expected, task)
		}
	}
}

func TestSyncWritesReadableChangeSummary(t *testing.T) {
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
	if err := Run([]string{"sync"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("sync failed: %v\nstderr: %s", err, stderr.String())
	}

	change := readOnlyChangeRecord(t, dir)
	mdBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "changes", change.ID+".md"))
	if err != nil {
		t.Fatal(err)
	}
	md := string(mdBytes)
	for _, expected := range []string{
		"## Summary",
		"1 modified source file in `chat`.",
		"## Why This Matters",
		"This record feeds audit trails, module `recentChanges`, proposal evidence, and future AI context.",
	} {
		if !strings.Contains(md, expected) {
			t.Fatalf("change markdown should include %q, got:\n%s", expected, md)
		}
	}
}

func TestSyncRefreshesDeterministicModuleFacts(t *testing.T) {
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
	writeFixtureFile(t, dir, "src/modules/chat/chat.test.ts", "export const tests = true\n")

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"sync"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("sync failed: %v\nstderr: %s", err, stderr.String())
	}

	moduleBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "chat.json"))
	if err != nil {
		t.Fatal(err)
	}
	var module moduleFile
	if err := json.Unmarshal(moduleBytes, &module); err != nil {
		t.Fatal(err)
	}
	assertContains(t, module.Facts.Tests, "src/modules/chat/chat.test.ts")
	if module.Facts.LastScannedAt == "" {
		t.Fatalf("sync should update deterministic fact scan time, got:\n%s", string(moduleBytes))
	}
}
