package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanJSONReportsProjectSignalsAndModuleCandidates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "package.json", `{"scripts":{"test":"vitest"},"dependencies":{"react":"latest"}}`)
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = true\n")
	writeFixtureFile(t, dir, "src/modules/chat/chat.test.ts", "test('chat', () => {})\n")
	writeFixtureFile(t, dir, "vite.config.ts", "export default {}\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"scan", "--json"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("scan failed: %v\nstderr: %s", err, stderr.String())
	}

	var report scanReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("scan should emit JSON, got error %v and output:\n%s", err, stdout.String())
	}

	if report.SchemaVersion != schemaVersion {
		t.Fatalf("schema version = %q, want %q", report.SchemaVersion, schemaVersion)
	}
	assertContains(t, report.Manifests, "package.json")
	assertContains(t, report.Languages, "typescript")
	assertContains(t, report.Frameworks, "react")
	assertContains(t, report.ConfigFiles, "vite.config.ts")
	assertContains(t, report.TestFiles, "src/modules/chat/chat.test.ts")

	if len(report.ModuleCandidates) != 1 {
		t.Fatalf("expected one module candidate, got %#v", report.ModuleCandidates)
	}
	candidate := report.ModuleCandidates[0]
	if candidate.Name != "chat" {
		t.Fatalf("candidate name = %q, want chat", candidate.Name)
	}
	assertContains(t, candidate.Paths, "src/modules/chat/**")
	if candidate.Confidence == "" || candidate.Reason == "" {
		t.Fatalf("candidate should include confidence and reason, got %#v", candidate)
	}
}

func TestScanPrintsHumanReadableReport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "go.mod", "module example.com/project\n")
	writeFixtureFile(t, dir, "cmd/server/main.go", "package main\nfunc main() {}\n")
	writeFixtureFile(t, dir, "internal/auth/auth_test.go", "package auth\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"scan"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("scan failed: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	for _, expected := range []string{
		"Project scan",
		"Manifests: go.mod",
		"Languages: go",
		"cmd/server/main.go",
		"internal/auth/auth_test.go",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("scan output should contain %q, got:\n%s", expected, output)
		}
	}
}

func writeFixtureFile(t *testing.T, root string, path string, content string) {
	t.Helper()
	abs := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, value := range values {
		if value == expected {
			return
		}
	}
	t.Fatalf("expected %q in %#v", expected, values)
}
