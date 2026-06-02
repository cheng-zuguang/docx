package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateChangedWritesChangeLogAndModuleRecentChange(t *testing.T) {
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
	if err := Run([]string{"update", "--changed"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update failed: %v\nstderr: %s", err, stderr.String())
	}

	changeFiles, err := filepath.Glob(filepath.Join(dir, ".doc", "changes", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	changeFiles = withoutIndexJSON(changeFiles)
	if len(changeFiles) != 1 {
		t.Fatalf("expected one change json, got %#v", changeFiles)
	}

	changeBytes, err := os.ReadFile(changeFiles[0])
	if err != nil {
		t.Fatal(err)
	}
	var change struct {
		SchemaVersion string   `json:"schemaVersion"`
		Source        string   `json:"source"`
		Modules       []string `json:"modules"`
		Files         []struct {
			Path       string   `json:"path"`
			ChangeType string   `json:"changeType"`
			Signals    []string `json:"signals"`
		} `json:"files"`
		FactsUpdated []string `json:"factsUpdated"`
	}
	if err := json.Unmarshal(changeBytes, &change); err != nil {
		t.Fatal(err)
	}
	if change.SchemaVersion != schemaVersion {
		t.Fatalf("schemaVersion = %q, want %q", change.SchemaVersion, schemaVersion)
	}
	if change.Source != "git:changed" {
		t.Fatalf("source = %q, want git:changed", change.Source)
	}
	assertContains(t, change.Modules, "chat")
	if len(change.Files) != 1 || change.Files[0].Path != "src/modules/chat/index.ts" || change.Files[0].ChangeType != "modified" {
		t.Fatalf("unexpected changed files: %#v", change.Files)
	}
	assertContains(t, change.FactsUpdated, ".doc/modules/chat.json")

	mdPath := strings.TrimSuffix(changeFiles[0], ".json") + ".md"
	mdBytes, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mdBytes), "src/modules/chat/index.ts") || !strings.Contains(string(mdBytes), "chat") {
		t.Fatalf("change markdown should mention file and module, got:\n%s", string(mdBytes))
	}

	moduleBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "chat.json"))
	if err != nil {
		t.Fatal(err)
	}
	var module struct {
		RecentChanges []string `json:"recentChanges"`
		RiskRules     []string `json:"riskRules"`
		Summary       struct {
			Purpose string `json:"purpose"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(moduleBytes, &module); err != nil {
		t.Fatal(err)
	}
	assertContains(t, module.RecentChanges, ".doc/changes/"+filepath.Base(changeFiles[0]))
	if module.Summary.Purpose == "" {
		t.Fatalf("update should not wipe semantic module summary")
	}
	if module.RiskRules == nil {
		t.Fatalf("update should preserve riskRules as a semantic-memory field")
	}
}

func TestUpdateStagedReadsStagedChanges(t *testing.T) {
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

	writeFixtureFile(t, dir, "src/modules/chat/new.ts", "export const added = true\n")
	runGit(t, dir, "add", "src/modules/chat/new.ts")

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--staged"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update failed: %v\nstderr: %s", err, stderr.String())
	}

	change := readOnlyChangeRecord(t, dir)
	if change.Source != "git:staged" {
		t.Fatalf("source = %q, want git:staged", change.Source)
	}
	if len(change.Files) != 1 || change.Files[0].Path != "src/modules/chat/new.ts" || change.Files[0].ChangeType != "added" {
		t.Fatalf("unexpected staged files: %#v", change.Files)
	}
}

func TestUpdateSinceReadsGitRange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "initial")
	base := gitOutput(t, dir, "rev-parse", "HEAD")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 3\n")
	runGit(t, dir, "add", "src/modules/chat/index.ts")
	runGit(t, dir, "-c", "user.name=Docx Test", "-c", "user.email=docx@example.com", "commit", "-m", "change chat")

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--since", base}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update failed: %v\nstderr: %s", err, stderr.String())
	}

	change := readOnlyChangeRecord(t, dir)
	if change.Source != "git:since" {
		t.Fatalf("source = %q, want git:since", change.Source)
	}
	if change.Range != base+"..HEAD" {
		t.Fatalf("range = %q, want %q", change.Range, base+"..HEAD")
	}
	if len(change.Files) != 1 || change.Files[0].Path != "src/modules/chat/index.ts" {
		t.Fatalf("unexpected range files: %#v", change.Files)
	}
}

func TestUpdateModuleRecordsSelectedModule(t *testing.T) {
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

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--module", "chat"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update failed: %v\nstderr: %s", err, stderr.String())
	}

	change := readOnlyChangeRecord(t, dir)
	if change.Source != "module:chat" {
		t.Fatalf("source = %q, want module:chat", change.Source)
	}
	assertContains(t, change.Modules, "chat")
	assertContains(t, change.FactsUpdated, ".doc/modules/chat.json")
	if len(change.Files) != 0 {
		t.Fatalf("module update without git diff should not invent changed files: %#v", change.Files)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
	return strings.TrimSpace(string(output))
}

func withoutIndexJSON(paths []string) []string {
	var result []string
	for _, path := range paths {
		if filepath.Base(path) == "index.json" {
			continue
		}
		result = append(result, path)
	}
	return result
}

func readOnlyChangeRecord(t *testing.T, dir string) changeRecord {
	t.Helper()
	changeFiles, err := filepath.Glob(filepath.Join(dir, ".doc", "changes", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	changeFiles = withoutIndexJSON(changeFiles)
	if len(changeFiles) != 1 {
		t.Fatalf("expected one change json, got %#v", changeFiles)
	}
	changeBytes, err := os.ReadFile(changeFiles[0])
	if err != nil {
		t.Fatal(err)
	}
	var change changeRecord
	if err := json.Unmarshal(changeBytes, &change); err != nil {
		t.Fatal(err)
	}
	return change
}
