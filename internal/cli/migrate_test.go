package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateRepairsSchemaMismatchAndUpdateRequiresIt(t *testing.T) {
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
	writeFixtureFile(t, dir, ".docx.json", `{
  "schemaVersion": "2.0",
  "contextDir": ".doc",
  "contextSchemaVersion": "2.0",
  "entryFiles": ["AGENTS.md"]
}`)
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 2\n")

	stdout.Reset()
	stderr.Reset()
	err := Run([]string{"update", "--changed"}, dir, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "docx migrate") {
		t.Fatalf("update should require migration for major schema mismatch, got: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"migrate"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("migrate failed: %v\nstderr: %s", err, stderr.String())
	}
	firstConfig, err := os.ReadFile(filepath.Join(dir, ".docx.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(firstConfig), `"schemaVersion": "1.0"`) || !strings.Contains(string(firstConfig), `"contextSchemaVersion": "1.0"`) {
		t.Fatalf("migrate should repair schema versions, got:\n%s", string(firstConfig))
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"migrate"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("second migrate failed: %v\nstderr: %s", err, stderr.String())
	}
	secondConfig, err := os.ReadFile(filepath.Join(dir, ".docx.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(firstConfig) != string(secondConfig) {
		t.Fatalf("migrate should be idempotent\nfirst:\n%s\nsecond:\n%s", string(firstConfig), string(secondConfig))
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"update", "--changed"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("update after migrate failed: %v\nstderr: %s", err, stderr.String())
	}
}
