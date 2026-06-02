package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallHookPreservesExistingContentAndIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	hookPath := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho existing\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"install-hook", "pre-commit"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("install-hook failed: %v\nstderr: %s", err, stderr.String())
	}
	firstBytes, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	first := string(firstBytes)
	if !strings.Contains(first, "echo existing") || !strings.Contains(first, "# docx:start") || !strings.Contains(first, "docx update --staged") {
		t.Fatalf("hook should preserve existing content and include managed docx block, got:\n%s", first)
	}
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("hook should be executable, mode: %s", info.Mode())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"install-hook", "pre-commit"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("second install-hook failed: %v\nstderr: %s", err, stderr.String())
	}
	secondBytes, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != string(secondBytes) {
		t.Fatalf("install-hook should be idempotent\nfirst:\n%s\nsecond:\n%s", string(firstBytes), string(secondBytes))
	}
}
