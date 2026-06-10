package cli

import (
	"errors"
	"os"
	"strings"
)

func writeTextIfMissing(path string, text string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o644)
}

func upsertManagedBlock(path string, block string) error {
	existingBytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(path, []byte(block), 0o644)
	}
	if err != nil {
		return err
	}

	existing := string(existingBytes)
	start := "<!-- docx:start -->"
	end := "<!-- docx:end -->"
	startIndex := strings.Index(existing, start)
	endIndex := strings.Index(existing, end)
	if startIndex >= 0 && endIndex >= startIndex {
		endIndex += len(end)
		next := existing[:startIndex] + strings.TrimRight(block, "\n") + existing[endIndex:]
		if !strings.HasSuffix(next, "\n") {
			next += "\n"
		}
		return os.WriteFile(path, []byte(next), 0o644)
	}

	prefix := existing
	if strings.TrimSpace(prefix) != "" && !strings.HasSuffix(prefix, "\n\n") {
		if strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		} else {
			prefix += "\n\n"
		}
	}
	return os.WriteFile(path, []byte(prefix+block), 0o644)
}

func upsertNamedManagedBlock(path string, start string, end string, block string, perm os.FileMode) error {
	existingBytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(path, []byte(block), perm)
	}
	if err != nil {
		return err
	}
	existing := string(existingBytes)
	startIndex := strings.Index(existing, start)
	endIndex := strings.Index(existing, end)
	if startIndex >= 0 && endIndex >= startIndex {
		endIndex += len(end)
		next := existing[:startIndex] + strings.TrimRight(block, "\n") + existing[endIndex:]
		if !strings.HasSuffix(next, "\n") {
			next += "\n"
		}
		return os.WriteFile(path, []byte(next), perm)
	}
	prefix := existing
	if strings.TrimSpace(prefix) != "" && !strings.HasSuffix(prefix, "\n\n") {
		if strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		} else {
			prefix += "\n\n"
		}
	}
	return os.WriteFile(path, []byte(prefix+block), perm)
}

func agentBlock(contextDir string) string {
	return "<!-- docx:start -->\n" +
		"## Project Context\n\n" +
		"Before work, read `" + contextDir + "/index.json`.\n\n" +
		"Follow its `readOrder` progressively. Resolve edited paths with `moduleMap`; inspect decisions and recent changes for behavior changes; inspect mistakes while debugging or reviewing.\n\n" +
		"Keep `AGENTS.md` short; use `" + contextDir + "/rules/agent.md` for detailed behavior.\n\n" +
		"When an agent lifecycle hook is installed, let it run `docx finish`; otherwise run `docx sync` before finishing.\n\n" +
		"Use change records for audit trails, module `recentChanges`, proposal evidence, and future AI context.\n\n" +
		"Do not overwrite semantic memory in `" + contextDir + "/decisions/`, `" + contextDir + "/mistakes/`, or module `riskRules` without user confirmation. Write proposals instead.\n" +
		"<!-- docx:end -->\n"
}

func agentRulesText(contextDir string) string {
	return "# Agent Reading Protocol\n\n" +
		"Read `" + contextDir + "/index.json` first and follow `readOrder` progressively.\n\n" +
		"Read affected module files resolved from `moduleMap` before editing or summarizing module behavior.\n\n" +
		"When an agent lifecycle hook is installed, let it run `docx finish`; otherwise run `docx sync` before finishing.\n\n" +
		"`docx finish` is a safe lifecycle-hook wrapper around `docx sync`.\n\n" +
		"`docx sync` records changed files, updates deterministic module context, and writes an active-agent task under `" + contextDir + "/tmp/` when semantic follow-up is needed.\n\n" +
		"Deterministic facts may be refreshed directly; semantic memory requires proposals unless the user confirms the edit.\n\n" +
		"Use change records for audit trails, module `recentChanges`, proposal evidence, and future AI context.\n\n" +
		"Prefer finer modules around real workflows when a coarse module hides unrelated concepts.\n\n" +
		"When active-agent task files already exist under `" + contextDir + "/tmp/`, read the prompt, produce the requested JSON, then run the matching `docx apply ...` command.\n\n" +
		"Do not overwrite semantic memory in `" + contextDir + "/decisions/`, `" + contextDir + "/mistakes/`, or module `riskRules` without user confirmation. Write proposals instead.\n"
}

func gitignoreBlock(contextDir string) string {
	return "<!-- docx:start -->\n" +
		"# docx local state\n" +
		contextDir + "/.cache/\n" +
		contextDir + "/local/\n" +
		contextDir + "/tmp/\n" +
		"<!-- docx:end -->\n"
}
