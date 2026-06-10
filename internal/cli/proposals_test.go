package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProposalsListAndShowPendingProposal(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/proposals/prop-1.json", `{
  "schemaVersion": "1.0",
  "id": "prop-1",
  "type": "mistake",
  "status": "pending",
  "source": "test",
  "evidence": [
    {"path": "internal/chat/store.go", "reason": "State mutation caused a flaky test."}
  ],
  "suggestedTarget": ".doc/mistakes/testing.md",
  "suggestedPatch": {
    "id": "TEST-001",
    "title": "Keep state fixtures isolated",
    "body": "Tests should create isolated state instead of sharing mutable fixtures."
  }
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"proposals", "list"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("proposals list failed: %v\nstderr: %s", err, stderr.String())
	}
	listOutput := stdout.String()
	if !strings.Contains(listOutput, "prop-1") || !strings.Contains(listOutput, "mistake") || !strings.Contains(listOutput, ".doc/mistakes/testing.md") {
		t.Fatalf("list output should include pending proposal id, type, and target, got:\n%s", listOutput)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"proposals", "show", "prop-1"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("proposals show failed: %v\nstderr: %s", err, stderr.String())
	}
	showOutput := stdout.String()
	for _, expected := range []string{
		"prop-1",
		"State mutation caused a flaky test.",
		"internal/chat/store.go",
		".doc/mistakes/testing.md",
		"Keep state fixtures isolated",
	} {
		if !strings.Contains(showOutput, expected) {
			t.Fatalf("show output should contain %q, got:\n%s", expected, showOutput)
		}
	}
}

func TestProposalsAcceptWritesDecisionAndKeepsAuditRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/proposals/prop-decision.json", `{
  "schemaVersion": "1.0",
  "id": "prop-decision",
  "type": "decision",
  "status": "pending",
  "source": "test",
  "evidence": [
    {"path": "AGENTS.md", "reason": "Agents need a single context entrypoint."}
  ],
  "suggestedTarget": ".doc/decisions/ADR-0001-context-entrypoint.md",
  "suggestedPatch": {
    "title": "Use AGENTS.md as the default context entrypoint",
    "status": "accepted",
    "body": "New agents should start with AGENTS.md and follow the managed docx block."
  }
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"proposals", "accept", "prop-decision"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("proposals accept failed: %v\nstderr: %s", err, stderr.String())
	}

	decisionBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "decisions", "ADR-0001-context-entrypoint.md"))
	if err != nil {
		t.Fatal(err)
	}
	decision := string(decisionBytes)
	if !strings.Contains(decision, "# Use AGENTS.md as the default context entrypoint") || !strings.Contains(decision, "Status: accepted") {
		t.Fatalf("decision file was not written from proposal:\n%s", decision)
	}

	proposalBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "proposals", "prop-decision.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proposalBytes), `"status": "accepted"`) {
		t.Fatalf("proposal audit record should be retained and marked accepted:\n%s", string(proposalBytes))
	}
}

func TestProposalsRejectMarksRejectedWithoutDeletingAuditRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/proposals/prop-reject.json", `{
  "schemaVersion": "1.0",
  "id": "prop-reject",
  "type": "risk-rule",
  "status": "pending",
  "source": "test",
  "evidence": [
    {"path": "internal/cli/update.go", "reason": "Risk rule was speculative."}
  ],
  "suggestedTarget": ".doc/modules/cli.json",
  "suggestedPatch": {"rule": "Never rewrite user-authored rules automatically."}
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"proposals", "reject", "prop-reject"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("proposals reject failed: %v\nstderr: %s", err, stderr.String())
	}

	proposalBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "proposals", "prop-reject.json"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(proposalBytes)
	if !strings.Contains(content, `"status": "rejected"`) || !strings.Contains(content, "Never rewrite user-authored rules automatically.") {
		t.Fatalf("proposal audit record should be retained and marked rejected:\n%s", content)
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"proposals", "list"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("proposals list failed: %v\nstderr: %s", err, stderr.String())
	}
	if strings.Contains(stdout.String(), "prop-reject") {
		t.Fatalf("rejected proposals should not appear in pending list:\n%s", stdout.String())
	}
}

func TestProposalsAcceptAppendsMistakeEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/mistakes/testing.md", "# Testing\n\n")
	writeFixtureFile(t, dir, ".doc/proposals/prop-mistake.json", `{
  "schemaVersion": "1.0",
  "id": "prop-mistake",
  "type": "mistake",
  "status": "pending",
  "source": "test",
  "evidence": [
    {"path": "internal/cli/proposals_test.go", "reason": "Shared fixtures hid proposal state."}
  ],
  "suggestedTarget": ".doc/mistakes/testing.md",
  "suggestedPatch": {
    "id": "TEST-002",
    "title": "Isolate proposal fixtures",
    "appliesTo": ["cli", "tests"],
    "body": "Each proposal test should create its own proposal record."
  }
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"proposals", "accept", "prop-mistake"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("proposals accept failed: %v\nstderr: %s", err, stderr.String())
	}

	mistakeBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "mistakes", "testing.md"))
	if err != nil {
		t.Fatal(err)
	}
	mistake := string(mistakeBytes)
	for _, expected := range []string{"## [TEST-002] Isolate proposal fixtures", "**appliesTo**: cli, tests", "Each proposal test should create its own proposal record."} {
		if !strings.Contains(mistake, expected) {
			t.Fatalf("mistake file should contain %q, got:\n%s", expected, mistake)
		}
	}
}

func TestProposalsAcceptCanOverrideSuggestedTarget(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/mistakes/security.md", "# Security\n\n")
	writeFixtureFile(t, dir, ".doc/proposals/prop-target.json", `{
  "schemaVersion": "1.0",
  "id": "prop-target",
  "type": "mistake",
  "status": "pending",
  "source": "test",
  "evidence": [
    {"path": "internal/auth.go", "reason": "This belongs under security."}
  ],
  "suggestedTarget": ".doc/mistakes/testing.md",
  "suggestedPatch": {
    "id": "SEC-001",
    "title": "Keep auth checks explicit",
    "body": "Authentication behavior should not depend on implicit defaults."
  }
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"proposals", "accept", "prop-target", "--target", ".doc/mistakes/security.md"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("proposals accept with target override failed: %v\nstderr: %s", err, stderr.String())
	}

	securityBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "mistakes", "security.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(securityBytes), "## [SEC-001] Keep auth checks explicit") {
		t.Fatalf("override target should receive mistake entry:\n%s", string(securityBytes))
	}
}

func TestProposalsAcceptUpdatesModuleSummaryAndRiskRules(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = 1\n")
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init", "--accept-candidates"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/proposals/prop-summary.json", `{
  "schemaVersion": "1.0",
  "id": "prop-summary",
  "type": "module-summary",
  "status": "pending",
  "source": "test",
  "evidence": [
    {"path": "src/modules/chat/index.ts", "reason": "The module exposes chat behavior."}
  ],
  "suggestedTarget": ".doc/modules/chat.json",
  "suggestedPatch": {
    "purpose": "Owns chat user workflows.",
    "ownedConcepts": ["conversation"],
    "nonGoals": ["billing"]
  }
}`)
	writeFixtureFile(t, dir, ".doc/proposals/prop-risk.json", `{
  "schemaVersion": "1.0",
  "id": "prop-risk",
  "type": "risk-rule",
  "status": "pending",
  "source": "test",
  "evidence": [
    {"path": "src/modules/chat/index.ts", "reason": "Chat state is user-visible."}
  ],
  "suggestedTarget": ".doc/modules/chat.json",
  "suggestedPatch": {
    "rule": "Preserve message ordering when changing chat state."
  }
}`)

	for _, id := range []string{"prop-summary", "prop-risk"} {
		stdout.Reset()
		stderr.Reset()
		if err := Run([]string{"proposals", "accept", id}, dir, &stdout, &stderr); err != nil {
			t.Fatalf("proposals accept %s failed: %v\nstderr: %s", id, err, stderr.String())
		}
	}

	moduleBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "chat.json"))
	if err != nil {
		t.Fatal(err)
	}
	module := string(moduleBytes)
	for _, expected := range []string{
		`"purpose": "Owns chat user workflows."`,
		`"conversation"`,
		`"billing"`,
		`"Preserve message ordering when changing chat state."`,
	} {
		if !strings.Contains(module, expected) {
			t.Fatalf("module file should contain %q, got:\n%s", expected, module)
		}
	}
}

func TestProposalsAcceptAppliesAgentModulePartition(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/app/chat/index.ts", "export const chat = 1\n")
	writeFixtureFile(t, dir, "src/app/billing/index.ts", "export const billing = 1\n")
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	writeFixtureFile(t, dir, ".doc/proposals/prop-partition.json", `{
  "schemaVersion": "1.0",
  "id": "prop-partition",
  "type": "module-partition",
  "status": "pending",
  "source": "ai:active-agent",
  "evidence": [
    {"path": "src/app/chat/index.ts", "reason": "Chat and billing own separate workflows."},
    {"path": "src/app/billing/index.ts", "reason": "Billing has independent concepts and tests."}
  ],
  "suggestedTarget": ".doc/index.json",
  "suggestedPatch": {
    "modules": [
      {
        "name": "chat-workflow",
        "paths": ["src/app/chat/**"],
        "purpose": "Owns chat user workflows.",
        "ownedConcepts": ["conversation"],
        "nonGoals": ["billing"]
      },
      {
        "name": "billing-workflow",
        "paths": ["src/app/billing/**"],
        "purpose": "Owns billing user workflows.",
        "ownedConcepts": ["invoice"],
        "nonGoals": ["conversation"]
      }
    ]
  }
}`)

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"proposals", "accept", "prop-partition"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("proposals accept failed: %v\nstderr: %s", err, stderr.String())
	}

	indexBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index indexFile
	if err := json.Unmarshal(indexBytes, &index); err != nil {
		t.Fatal(err)
	}
	for _, moduleName := range []string{"chat-workflow", "billing-workflow"} {
		entry, ok := index.ModuleMap[moduleName]
		if !ok {
			t.Fatalf("moduleMap should include %s, got:\n%s", moduleName, string(indexBytes))
		}
		if entry.Confidence != "confirmed" || entry.Context != ".doc/modules/"+moduleName+".json" {
			t.Fatalf("moduleMap entry for %s should be confirmed with context path, got %#v", moduleName, entry)
		}
	}

	moduleBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "modules", "chat-workflow.json"))
	if err != nil {
		t.Fatal(err)
	}
	module := string(moduleBytes)
	for _, expected := range []string{
		`"module": "chat-workflow"`,
		`"status": "confirmed"`,
		`"purpose": "Owns chat user workflows."`,
		`"conversation"`,
		`"billing"`,
	} {
		if !strings.Contains(module, expected) {
			t.Fatalf("partition should write module file containing %q, got:\n%s", expected, module)
		}
	}
}
