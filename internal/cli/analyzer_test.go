package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestScanUsesBundledTypeScriptAnalyzer(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not available")
	}

	dir := t.TempDir()
	writeFixtureFile(t, dir, "package.json", `{"scripts":{"test":"vitest"},"dependencies":{"react":"latest","vite":"latest"}}`)
	writeFixtureFile(t, dir, "src/main.tsx", "import React from 'react'\nimport { helper } from './lib/helper'\nexport const App = () => <div />\n")
	writeFixtureFile(t, dir, "src/routes/home.ts", "export default function Home() { return null }\n")
	writeFixtureFile(t, dir, "src/lib/helper.ts", "export function helper() { return true }\n")
	writeFixtureFile(t, dir, "src/main.test.ts", "import { App } from './main'\ntest('app', () => {})\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"scan", "--analyzer", "typescript", "--json"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("scan failed: %v\nstderr: %s", err, stderr.String())
	}

	var report scanReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("scan should emit JSON, got error %v and output:\n%s", err, stdout.String())
	}
	if report.Analyzer.Name != "typescript" {
		t.Fatalf("analyzer name = %q, want typescript", report.Analyzer.Name)
	}
	assertContains(t, report.Manifests, "package.json")
	assertContains(t, report.Languages, "typescript")
	assertContains(t, report.Frameworks, "react")
	assertContains(t, report.Frameworks, "vite")
	assertContains(t, report.Entrypoints, "src/main.tsx")
	assertContains(t, report.TestFiles, "src/main.test.ts")
	assertContains(t, report.Imports, "react")
	assertContains(t, report.Imports, "./lib/helper")
	assertContains(t, report.Exports, "App")
	assertContains(t, report.Exports, "helper")
	assertContains(t, report.Routes, "src/routes/home.ts")
}

func TestScanUsesBundledTypeScriptAnalyzerForJavaScriptProject(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not available")
	}

	dir := t.TempDir()
	writeFixtureFile(t, dir, "package.json", `{"dependencies":{"express":"latest"}}`)
	writeFixtureFile(t, dir, "src/index.js", "const express = require('express')\nmodule.exports = { app: express() }\n")
	writeFixtureFile(t, dir, "src/routes/users.js", "exports.users = function users() {}\n")
	writeFixtureFile(t, dir, "src/index.spec.js", "const { app } = require('./index')\ntest('app', () => {})\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"scan", "--analyzer", "typescript", "--json"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("scan failed: %v\nstderr: %s", err, stderr.String())
	}

	var report scanReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("scan should emit JSON, got error %v and output:\n%s", err, stdout.String())
	}
	assertContains(t, report.Languages, "javascript")
	assertContains(t, report.Frameworks, "express")
	assertContains(t, report.Entrypoints, "src/index.js")
	assertContains(t, report.Imports, "express")
	assertContains(t, report.Exports, "users")
	assertContains(t, report.Routes, "src/routes/users.js")
	assertContains(t, report.TestFiles, "src/index.spec.js")
}

func TestScanInvokesAnalyzerAndConsumesProtocolOutput(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell analyzer fixture is POSIX-only")
	}

	dir := t.TempDir()
	writeFixtureFile(t, dir, "package.json", `{"dependencies":{"react":"latest"}}`)
	writeFixtureFile(t, dir, "src/routes/home.ts", "export const route = '/'\n")
	analyzer := filepath.Join(dir, "fake-analyzer.sh")
	if err := os.WriteFile(analyzer, []byte(`#!/bin/sh
input=$(cat)
case "$input" in
  *'"schemaVersion":"1.0"'*|*'"schemaVersion": "1.0"'*) ;;
  *) exit 3 ;;
esac
cat <<'JSON'
{
  "schemaVersion": "1.0",
  "analyzer": {
    "name": "fake-typescript",
    "version": "0.1.0",
    "languages": ["typescript"],
    "capabilities": ["routes"]
  },
  "report": {
    "manifests": ["package.json"],
    "languages": ["typescript"],
    "frameworks": ["react", "custom-router"],
    "entrypoints": ["src/routes/home.ts"],
    "testFiles": [],
    "configFiles": [],
    "moduleCandidates": [
      {
        "name": "routes",
        "paths": ["src/routes/**"],
        "confidence": "high",
        "reason": "analyzer detected route ownership"
      }
    ]
  }
}
JSON
`), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"scan", "--analyzer", analyzer, "--json"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("scan failed: %v\nstderr: %s", err, stderr.String())
	}

	var report scanReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("scan should emit JSON, got error %v and output:\n%s", err, stdout.String())
	}
	assertContains(t, report.Languages, "typescript")
	assertContains(t, report.Frameworks, "custom-router")
	assertContains(t, report.Entrypoints, "src/routes/home.ts")
	if len(report.ModuleCandidates) != 1 || report.ModuleCandidates[0].Name != "routes" {
		t.Fatalf("scan should consume analyzer module candidates, got %#v", report.ModuleCandidates)
	}
}

func TestScanFallsBackToGenericWhenAnalyzerFails(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell analyzer fixture is POSIX-only")
	}

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/modules/chat/index.ts", "export const chat = true\n")
	analyzer := filepath.Join(dir, "failing-analyzer.sh")
	if err := os.WriteFile(analyzer, []byte("#!/bin/sh\nexit 7\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"scan", "--analyzer", analyzer}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("scan should fall back to generic analyzer, got: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Analyzer fallback: generic") || !strings.Contains(output, "chat") {
		t.Fatalf("scan should report generic fallback and generic module candidates, got:\n%s", output)
	}
}

func TestScanReportsActionableAnalyzerDiagnosticsOnFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "src/index.ts", "export const app = true\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"scan", "--analyzer", "definitely-missing-docx-analyzer"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("scan should fall back when analyzer command is missing, got: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Analyzer fallback: generic") ||
		!strings.Contains(output, "Analyzer diagnostic:") ||
		!strings.Contains(output, "Install or configure") {
		t.Fatalf("fallback should include actionable analyzer diagnostic, got:\n%s", output)
	}
}

func TestScanFallsBackWhenAnalyzerViolatesProtocol(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("shell analyzer fixture is POSIX-only")
	}

	dir := t.TempDir()
	writeFixtureFile(t, dir, "main.go", "package main\nfunc main() {}\n")
	analyzer := filepath.Join(dir, "invalid-protocol-analyzer.sh")
	if err := os.WriteFile(analyzer, []byte(`#!/bin/sh
cat <<'JSON'
{"schemaVersion":"9.0","analyzer":{"name":"broken"},"report":{}}
JSON
`), 0o755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"scan", "--analyzer", analyzer}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("scan should fall back when analyzer violates protocol, got: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Analyzer fallback: generic") || !strings.Contains(output, "Languages: go") {
		t.Fatalf("scan should fall back to generic after protocol violation, got:\n%s", output)
	}
}

func TestInitRecordsAnalyzerCapabilitiesAndMissingRecommendations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "package.json", `{"dependencies":{"typescript":"latest"}}`)
	writeFixtureFile(t, dir, "go.mod", "module example.com/project\n")
	writeFixtureFile(t, dir, "main.go", "package main\n")
	writeFixtureFile(t, dir, "src/index.ts", "export const app = true\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	capabilityBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "capabilities.json"))
	if err != nil {
		t.Fatal(err)
	}
	var capabilities capabilitiesFile
	if err := json.Unmarshal(capabilityBytes, &capabilities); err != nil {
		t.Fatalf("capabilities should use stable JSON format: %v\n%s", err, string(capabilityBytes))
	}
	assertContains(t, analyzerCapabilityNames(capabilities.AvailableAnalyzers), "generic")
	var missing []string
	for _, analyzer := range capabilities.MissingRecommendedAnalyzers {
		missing = append(missing, analyzer.Name)
	}
	assertContains(t, missing, "go")
}

func TestInitRecordsBundledTypeScriptAnalyzerWhenNodeIsAvailable(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not available")
	}

	dir := t.TempDir()
	writeFixtureFile(t, dir, "package.json", `{"dependencies":{"react":"latest"}}`)
	writeFixtureFile(t, dir, "src/index.ts", "export const app = true\n")

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}

	capabilityBytes, err := os.ReadFile(filepath.Join(dir, ".doc", "capabilities.json"))
	if err != nil {
		t.Fatal(err)
	}
	var capabilities capabilitiesFile
	if err := json.Unmarshal(capabilityBytes, &capabilities); err != nil {
		t.Fatal(err)
	}
	available := analyzerCapabilityNames(capabilities.AvailableAnalyzers)
	assertContains(t, available, "generic")
	assertContains(t, available, "typescript")
	for _, missing := range capabilities.MissingRecommendedAnalyzers {
		if missing.Name == "typescript" {
			t.Fatalf("typescript analyzer should not be missing when bundled analyzer can run with node:\n%s", string(capabilityBytes))
		}
	}
}
