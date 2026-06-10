package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runSync(args []string, cwd string, stdout io.Writer) error {
	propose := false
	for _, arg := range args {
		switch arg {
		case "--propose":
			propose = true
		default:
			return fmt.Errorf("docx sync: unknown option %q", arg)
		}
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	if err := runUpdate([]string{"--changed"}, cwd, stdout); err != nil {
		return err
	}
	change, err := latestChangeRecord(root, config.ContextDir)
	if err != nil {
		return err
	}
	if err := refreshDeterministicModuleFacts(root, config.ContextDir, change.Modules); err != nil {
		return err
	}
	if propose {
		index, err := readIndex(filepath.Join(root, config.ContextDir, "index.json"))
		if err != nil {
			return err
		}
		if err := writeProposalTask(root, config.ContextDir, index, change.Modules, change.Files, change.ID, change.Source, stdout); err != nil {
			return err
		}
	}
	taskPath := filepath.Join(root, config.ContextDir, "tmp", "agent-sync.md")
	if err := os.MkdirAll(filepath.Dir(taskPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(taskPath, []byte(agentSyncTask(config.ContextDir, change, propose)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Agent sync task: %s\n", config.ContextDir+"/tmp/agent-sync.md")
	return nil
}

func refreshDeterministicModuleFacts(root string, contextDir string, modules []string) error {
	if len(modules) == 0 {
		return nil
	}
	report, err := scanProject(root)
	if err != nil {
		return err
	}
	index, err := readIndex(filepath.Join(root, contextDir, "index.json"))
	if err != nil {
		return err
	}
	scannedAt := time.Now().UTC().Format(time.RFC3339)
	for _, moduleName := range modules {
		modulePath := filepath.Join(root, contextDir, "modules", moduleName+".json")
		bytes, err := os.ReadFile(modulePath)
		if err != nil {
			return err
		}
		var module moduleFile
		if err := json.Unmarshal(bytes, &module); err != nil {
			return err
		}
		paths := module.Paths
		if entry, ok := index.ModuleMap[moduleName]; ok && len(entry.Paths) > 0 {
			paths = entry.Paths
		}
		module.Facts.Entrypoints = entrypointsForModule(report.Entrypoints, paths)
		module.Facts.Tests = testsForModule(report.TestFiles, paths)
		module.Facts.LastScannedAt = scannedAt
		if err := writeJSON(modulePath, module); err != nil {
			return err
		}
	}
	return nil
}

func latestChangeRecord(root string, contextDir string) (changeRecord, error) {
	records, err := collectJSONFiles[changeRecord](filepath.Join(root, contextDir, "changes"))
	if err != nil {
		return changeRecord{}, err
	}
	if len(records) == 0 {
		return changeRecord{}, fmt.Errorf("docx sync: no change record was created")
	}
	latest := records[0]
	for _, record := range records[1:] {
		if record.ID > latest.ID {
			latest = record
		}
	}
	return latest, nil
}

func agentSyncTask(contextDir string, change changeRecord, propose bool) string {
	var builder strings.Builder
	builder.WriteString("# Active Agent Sync\n\n")
	builder.WriteString("Change: `" + change.ID + "`\n\n")
	builder.WriteString("The active agent should complete semantic context sync directly from this task.\n\n")
	builder.WriteString("## Changed Files\n\n")
	for _, file := range change.Files {
		builder.WriteString("- " + file.ChangeType + ": `" + file.Path + "`\n")
	}
	builder.WriteString("\n## Affected Modules\n\n")
	for _, module := range change.Modules {
		builder.WriteString("- `" + contextDir + "/modules/" + module + ".json`\n")
	}
	builder.WriteString("\n## Task\n\n")
	builder.WriteString("Review the changed files and affected module context. Update generated facts directly when they are deterministic. For semantic memory such as risk rules, decisions, and mistakes, write proposals instead of editing protected memory directly.\n")
	if propose {
		builder.WriteString("\n## Proposal Task\n\n")
		builder.WriteString("Read `" + contextDir + "/tmp/proposals-prompt.md`, write `" + contextDir + "/tmp/proposals-output.json`, then run `docx apply proposals " + contextDir + "/tmp/proposals-output.json` to create pending proposals.\n")
	}
	return builder.String()
}
