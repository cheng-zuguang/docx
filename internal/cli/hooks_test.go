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

func TestInstallHookCanUseProposalTaskUpdateCommand(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"install-hook", "pre-commit", "--propose"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("install-hook --propose failed: %v\nstderr: %s", err, stderr.String())
	}

	hookBytes, err := os.ReadFile(filepath.Join(dir, ".git", "hooks", "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	hook := string(hookBytes)
	if !strings.Contains(hook, "docx update --staged --propose") {
		t.Fatalf("proposal task hook should run staged update with --propose, got:\n%s", hook)
	}
}

func TestInstallHookRejectsRemovedOptions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	runGit(t, dir, "init")
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	for _, option := range []string{"--ai", "--handoff"} {
		option := option
		t.Run(option, func(t *testing.T) {
			stdout.Reset()
			stderr.Reset()
			err := Run([]string{"install-hook", "pre-commit", option}, dir, &stdout, &stderr)
			if err == nil {
				t.Fatalf("install-hook %s should fail after the option was removed", option)
			}
			if !strings.Contains(err.Error(), `unknown option "`+option+`"`) {
				t.Fatalf("install-hook %s should report an unknown option, got %v", option, err)
			}
		})
	}
}
