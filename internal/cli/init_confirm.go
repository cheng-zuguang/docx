package cli

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

type decidedModule struct {
	Name   string
	Paths  []string
	Status string
}

func readModuleDecisions(stdin io.Reader, stdout io.Writer, candidates []moduleCandidate) ([]decidedModule, error) {
	candidateMap := map[string]moduleCandidate{}
	for _, candidate := range candidates {
		candidateMap[candidate.Name] = candidate
	}
	fmt.Fprintln(stdout, "Review module candidates. Commands: accept <name>, ignore <name>, rename <old> <new>, merge <new> <a,b>, done")

	scanner := bufio.NewScanner(stdin)
	var decisions []decidedModule
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "done" {
			break
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "accept":
			if len(fields) != 2 {
				return nil, fmt.Errorf("interactive init: accept requires a module name")
			}
			candidate, ok := candidateMap[fields[1]]
			if !ok {
				return nil, fmt.Errorf("interactive init: unknown module candidate %q", fields[1])
			}
			decisions = append(decisions, decidedModule{Name: candidate.Name, Paths: candidate.Paths, Status: "confirmed"})
		case "ignore":
			if len(fields) != 2 {
				return nil, fmt.Errorf("interactive init: ignore requires a module name")
			}
			if _, ok := candidateMap[fields[1]]; !ok {
				return nil, fmt.Errorf("interactive init: unknown module candidate %q", fields[1])
			}
		case "rename":
			if len(fields) != 3 {
				return nil, fmt.Errorf("interactive init: rename requires old and new module names")
			}
			candidate, ok := candidateMap[fields[1]]
			if !ok {
				return nil, fmt.Errorf("interactive init: unknown module candidate %q", fields[1])
			}
			decisions = append(decisions, decidedModule{Name: fields[2], Paths: candidate.Paths, Status: "confirmed"})
		case "merge":
			if len(fields) != 3 {
				return nil, fmt.Errorf("interactive init: merge requires new module name and comma-separated candidates")
			}
			var paths []string
			for _, name := range strings.Split(fields[2], ",") {
				candidate, ok := candidateMap[strings.TrimSpace(name)]
				if !ok {
					return nil, fmt.Errorf("interactive init: unknown module candidate %q", name)
				}
				paths = append(paths, candidate.Paths...)
			}
			sort.Strings(paths)
			decisions = append(decisions, decidedModule{Name: fields[1], Paths: uniqueStrings(paths), Status: "confirmed"})
		default:
			return nil, fmt.Errorf("interactive init: unknown command %q", fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return decisions, nil
}

func writeDecidedModules(docRoot string, contextDir string, report scanReport, decisions []decidedModule, moduleMap map[string]moduleMapEntry) error {
	for _, decision := range decisions {
		if err := writeModuleDecision(docRoot, contextDir, report, decision, moduleMap); err != nil {
			return err
		}
	}
	return nil
}

func writeModuleDecision(docRoot string, contextDir string, report scanReport, decision decidedModule, moduleMap map[string]moduleMapEntry) error {
	contextPath := contextDir + "/modules/" + decision.Name + ".json"
	moduleMap[decision.Name] = moduleMapEntry{
		Paths:      decision.Paths,
		Context:    contextPath,
		Confidence: decision.Status,
	}
	return writeJSON(filepath.Join(docRoot, "modules", decision.Name+".json"), moduleFile{
		SchemaVersion: schemaVersion,
		Module:        decision.Name,
		Status:        decision.Status,
		Paths:         decision.Paths,
		Summary: moduleSummary{
			Purpose:       "Module boundary confirmed during docx init.",
			OwnedConcepts: []string{},
			NonGoals:      []string{},
		},
		Facts: moduleFacts{
			Entrypoints:   entrypointsForModule(report.Entrypoints, decision.Paths),
			PublicAPI:     []string{},
			Dependencies:  []string{},
			Dependents:    []string{},
			Tests:         testsForModule(report.TestFiles, decision.Paths),
			LastScannedAt: "",
		},
		ReadHints:     readHints{AlwaysRead: []string{}, ReadFor: []interface{}{}},
		RiskRules:     []string{},
		RecentChanges: []string{},
	})
}

func entrypointsForModule(entrypoints []string, globs []string) []string {
	return pathsForModule(entrypoints, globs)
}

func testsForModule(testFiles []string, globs []string) []string {
	return pathsForModule(testFiles, globs)
}

func pathsForModule(paths []string, globs []string) []string {
	var result []string
	for _, path := range paths {
		for _, glob := range globs {
			prefix := strings.TrimSuffix(glob, "/**")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				result = append(result, path)
				break
			}
		}
	}
	return result
}
