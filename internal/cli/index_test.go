package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexRebuildsAllIndexesDeterministically(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	writeFixtureFile(t, dir, ".doc/changes/20260101T000000Z.json", `{"schemaVersion":"1.0","id":"20260101T000000Z","source":"git:changed","modules":["chat"],"files":[],"factsUpdated":[],"proposals":[]}`)
	writeFixtureFile(t, dir, ".doc/proposals/prop-1.json", `{"schemaVersion":"1.0","id":"prop-1","type":"mistake","status":"pending","source":"test","evidence":[],"suggestedTarget":".doc/mistakes/runtime-environment.md","suggestedPatch":{}}`)
	writeFixtureFile(t, dir, ".doc/decisions/ADR-0001-context.md", "# Context Contract\n\nStatus: accepted\n")
	writeFixtureFile(t, dir, ".doc/mistakes/runtime-environment.md", "## [ENV-001] Do not assume browser globals\n\n**appliesTo**: react-native, cli\n")

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"index"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("index failed: %v\nstderr: %s", err, stderr.String())
	}

	first := readIndexOutputs(t, dir)
	for path, content := range first {
		if !strings.Contains(content, `"schemaVersion": "1.0"`) {
			t.Fatalf("%s should include schema version, got:\n%s", path, content)
		}
	}
	if !strings.Contains(first[".doc/changes/index.json"], `"20260101T000000Z"`) {
		t.Fatalf("changes index missing change id:\n%s", first[".doc/changes/index.json"])
	}
	if !strings.Contains(first[".doc/proposals/index.json"], `"prop-1"`) {
		t.Fatalf("proposals index missing proposal id:\n%s", first[".doc/proposals/index.json"])
	}
	if !strings.Contains(first[".doc/decisions/index.json"], `"ADR-0001-context"`) {
		t.Fatalf("decisions index missing ADR id:\n%s", first[".doc/decisions/index.json"])
	}
	if !strings.Contains(first[".doc/mistakes/index.json"], `"ENV-001"`) {
		t.Fatalf("mistakes index missing mistake id:\n%s", first[".doc/mistakes/index.json"])
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"index"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("second index failed: %v\nstderr: %s", err, stderr.String())
	}
	second := readIndexOutputs(t, dir)
	for path, content := range first {
		if second[path] != content {
			t.Fatalf("%s should be deterministic\nfirst:\n%s\nsecond:\n%s", path, content, second[path])
		}
	}
}

func TestIndexSectionRebuildsOnlySelectedIndex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/changes/20260101T000000Z.json", `{"schemaVersion":"1.0","id":"20260101T000000Z","source":"git:changed","modules":["chat"],"files":[],"factsUpdated":[],"proposals":[]}`)
	writeFixtureFile(t, dir, ".doc/proposals/index.json", `{"schemaVersion":"1.0","items":[{"id":"keep","path":"x"}]}`)

	if err := Run([]string{"index", "--section", "changes"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("index failed: %v\nstderr: %s", err, stderr.String())
	}

	changesBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "changes", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(changesBytes), "20260101T000000Z") {
		t.Fatalf("changes index was not rebuilt:\n%s", string(changesBytes))
	}
	proposalsBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "proposals", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proposalsBytes), `"keep"`) {
		t.Fatalf("section rebuild should not rewrite proposals index:\n%s", string(proposalsBytes))
	}
}

func TestIndexCheckReportsStaleIndexWithoutWriting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/changes/20260101T000000Z.json", `{"schemaVersion":"1.0","id":"20260101T000000Z","source":"git:changed","modules":["chat"],"files":[],"factsUpdated":[],"proposals":[]}`)
	writeFixtureFile(t, dir, ".doc/changes/index.json", `{"schemaVersion":"1.0","items":[]}`)
	beforeBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "changes", "index.json"))
	if err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	err = Run([]string{"index", "--check", "--section", "changes"}, dir, &stdout, &stderr)
	if err == nil {
		t.Fatalf("index --check should fail for stale index")
	}
	if !strings.Contains(err.Error(), "stale") {
		t.Fatalf("check error should mention stale index, got: %v", err)
	}
	afterBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "changes", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(afterBytes) != string(beforeBytes) {
		t.Fatalf("index --check should not modify files\nbefore:%s\nafter:%s", beforeBytes, afterBytes)
	}
}

func readIndexOutputs(t *testing.T, dir string) map[string]string {
	t.Helper()
	paths := []string{
		".doc/changes/index.json",
		".doc/proposals/index.json",
		".doc/decisions/index.json",
		".doc/mistakes/index.json",
	}
	outputs := map[string]string{}
	for _, path := range paths {
		bytes, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Fatal(err)
		}
		outputs[path] = string(bytes)
	}
	return outputs
}
