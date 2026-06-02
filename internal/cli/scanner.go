package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type scanReport struct {
	SchemaVersion    string            `json:"schemaVersion"`
	Analyzer         analyzerInfo      `json:"analyzer,omitempty"`
	Manifests        []string          `json:"manifests"`
	Languages        []string          `json:"languages"`
	Frameworks       []string          `json:"frameworks"`
	Entrypoints      []string          `json:"entrypoints"`
	Imports          []string          `json:"imports,omitempty"`
	Exports          []string          `json:"exports,omitempty"`
	Routes           []string          `json:"routes,omitempty"`
	TestFiles        []string          `json:"testFiles"`
	ConfigFiles      []string          `json:"configFiles"`
	ModuleCandidates []moduleCandidate `json:"moduleCandidates"`
}

type analyzerInfo struct {
	Name         string   `json:"name,omitempty"`
	Version      string   `json:"version,omitempty"`
	Languages    []string `json:"languages,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type analyzerInput struct {
	SchemaVersion string `json:"schemaVersion"`
	Root          string `json:"root"`
}

type analyzerOutput struct {
	SchemaVersion string       `json:"schemaVersion"`
	Analyzer      analyzerInfo `json:"analyzer"`
	Report        scanReport   `json:"report"`
}

type moduleCandidate struct {
	Name       string   `json:"name"`
	Paths      []string `json:"paths"`
	Confidence string   `json:"confidence"`
	Reason     string   `json:"reason"`
}

func runScan(args []string, cwd string, stdout io.Writer) error {
	jsonOutput := false
	analyzer := "generic"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--analyzer":
			if i+1 >= len(args) {
				return fmt.Errorf("docx scan: --analyzer requires a value")
			}
			i++
			analyzer = args[i]
		default:
			return fmt.Errorf("docx scan: unknown option %q", args[i])
		}
	}

	report, fallback, diagnostic, err := scanWithAnalyzer(cwd, analyzer)
	if err != nil {
		return err
	}

	if jsonOutput {
		return writeJSONTo(stdout, report)
	}

	fmt.Fprintln(stdout, "Project scan")
	if fallback {
		fmt.Fprintln(stdout, "Analyzer fallback: generic")
		if diagnostic != "" {
			fmt.Fprintf(stdout, "Analyzer diagnostic: %s\n", diagnostic)
		}
	}
	fmt.Fprintf(stdout, "Manifests: %s\n", strings.Join(report.Manifests, ", "))
	fmt.Fprintf(stdout, "Languages: %s\n", strings.Join(report.Languages, ", "))
	fmt.Fprintf(stdout, "Frameworks: %s\n", strings.Join(report.Frameworks, ", "))
	fmt.Fprintf(stdout, "Entrypoints: %s\n", strings.Join(report.Entrypoints, ", "))
	fmt.Fprintf(stdout, "Tests: %s\n", strings.Join(report.TestFiles, ", "))
	fmt.Fprintf(stdout, "Configs: %s\n", strings.Join(report.ConfigFiles, ", "))
	fmt.Fprintf(stdout, "Module candidates: %d\n", len(report.ModuleCandidates))
	for _, candidate := range report.ModuleCandidates {
		fmt.Fprintf(stdout, "- %s (%s): %s\n", candidate.Name, candidate.Confidence, candidate.Reason)
	}
	return nil
}

func scanWithAnalyzer(cwd string, analyzer string) (scanReport, bool, string, error) {
	if analyzer == "" || analyzer == "generic" {
		report, err := scanProject(cwd)
		return report, false, "", err
	}
	if analyzer == "typescript" || analyzer == "javascript" {
		report, err := runTypeScriptAnalyzer(cwd)
		if err == nil {
			return report, false, "", nil
		}
		fallback, fallbackErr := scanProject(cwd)
		if fallbackErr != nil {
			return scanReport{}, false, "", err
		}
		return fallback, true, analyzerDiagnostic(analyzer, err), nil
	}
	report, err := runExternalAnalyzer(cwd, analyzer)
	if err == nil {
		return report, false, "", nil
	}
	fallback, fallbackErr := scanProject(cwd)
	if fallbackErr != nil {
		return scanReport{}, false, "", err
	}
	return fallback, true, analyzerDiagnostic(analyzer, err), nil
}

func analyzerDiagnostic(analyzer string, err error) string {
	return fmt.Sprintf("%s failed (%v). Install or configure the analyzer, then retry `docx scan --analyzer %s`.", analyzer, err, analyzer)
}

func runExternalAnalyzer(cwd string, analyzer string) (scanReport, error) {
	root, err := filepath.Abs(cwd)
	if err != nil {
		return scanReport{}, err
	}
	parts := strings.Fields(analyzer)
	if len(parts) == 0 {
		return scanReport{}, fmt.Errorf("docx scan: empty analyzer command")
	}
	input, err := json.Marshal(analyzerInput{SchemaVersion: schemaVersion, Root: root})
	if err != nil {
		return scanReport{}, err
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	output, err := runAnalyzerCommandWithInput(root, cmd, input)
	if err != nil {
		return scanReport{}, err
	}
	var decoded analyzerOutput
	if err := json.Unmarshal(output, &decoded); err != nil {
		return scanReport{}, err
	}
	if decoded.SchemaVersion != schemaVersion {
		return scanReport{}, errUnsupportedAnalyzerSchema(decoded.SchemaVersion)
	}
	report := decoded.Report
	report.SchemaVersion = schemaVersion
	report.Analyzer = decoded.Analyzer
	sortScanReport(&report)
	return report, nil
}

func runAnalyzerCommand(cwd string, cmd *exec.Cmd) ([]byte, error) {
	root, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	input, err := json.Marshal(analyzerInput{SchemaVersion: schemaVersion, Root: root})
	if err != nil {
		return nil, err
	}
	return runAnalyzerCommandWithInput(root, cmd, input)
}

func runAnalyzerCommandWithInput(root string, cmd *exec.Cmd, input []byte) ([]byte, error) {
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(input)
	return cmd.Output()
}

func errUnsupportedAnalyzerSchema(version string) error {
	return fmt.Errorf("docx scan: analyzer schemaVersion %q is not supported", version)
}

func scanProject(cwd string) (scanReport, error) {
	root, err := filepath.Abs(cwd)
	if err != nil {
		return scanReport{}, err
	}

	report := scanReport{SchemaVersion: schemaVersion}
	moduleCandidates := map[string]moduleCandidate{}
	seenLanguages := map[string]bool{}
	seenFrameworks := map[string]bool{}

	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if entry.IsDir() {
			if shouldSkipScanDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		name := entry.Name()
		switch name {
		case "package.json", "go.mod", "pyproject.toml", "Cargo.toml", "pom.xml":
			report.Manifests = append(report.Manifests, rel)
		}
		if strings.HasSuffix(name, ".config.ts") || strings.HasSuffix(name, ".config.js") || strings.HasSuffix(name, ".config.mjs") || strings.HasSuffix(name, ".config.cjs") {
			report.ConfigFiles = append(report.ConfigFiles, rel)
		}
		if isTestFile(name) {
			report.TestFiles = append(report.TestFiles, rel)
		}
		if isEntrypoint(rel) {
			report.Entrypoints = append(report.Entrypoints, rel)
		}

		for _, language := range languagesForFile(name) {
			seenLanguages[language] = true
		}
		if name == "package.json" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			detectPackageFrameworks(string(content), seenFrameworks)
		}

		if candidate, ok := moduleCandidateForPath(rel); ok {
			moduleCandidates[candidate.Name] = candidate
		}
		return nil
	})
	if err != nil {
		return scanReport{}, err
	}

	report.Languages = sortedKeys(seenLanguages)
	report.Frameworks = sortedKeys(seenFrameworks)
	report.ModuleCandidates = sortedCandidates(moduleCandidates)
	sortScanReport(&report)
	return report, nil
}

func sortScanReport(report *scanReport) {
	sort.Strings(report.Manifests)
	sort.Strings(report.Languages)
	sort.Strings(report.Frameworks)
	sort.Strings(report.Entrypoints)
	sort.Strings(report.Imports)
	sort.Strings(report.Exports)
	sort.Strings(report.Routes)
	sort.Strings(report.TestFiles)
	sort.Strings(report.ConfigFiles)
	sort.Slice(report.ModuleCandidates, func(i, j int) bool {
		return report.ModuleCandidates[i].Name < report.ModuleCandidates[j].Name
	})
}

func shouldSkipScanDir(rel string) bool {
	switch rel {
	case ".git", "node_modules", "dist", "vendor", ".doc":
		return true
	default:
		return strings.HasPrefix(rel, ".git/") || strings.HasPrefix(rel, "node_modules/")
	}
}

func languagesForFile(name string) []string {
	switch {
	case strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx"):
		return []string{"typescript"}
	case strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".jsx") || strings.HasSuffix(name, ".mjs") || strings.HasSuffix(name, ".cjs"):
		return []string{"javascript"}
	case strings.HasSuffix(name, ".go"):
		return []string{"go"}
	case strings.HasSuffix(name, ".py"):
		return []string{"python"}
	case strings.HasSuffix(name, ".rs"):
		return []string{"rust"}
	default:
		return nil
	}
}

func detectPackageFrameworks(content string, frameworks map[string]bool) {
	for _, framework := range []string{"react", "vue", "svelte", "next", "vite", "express"} {
		if strings.Contains(content, `"`+framework+`"`) {
			frameworks[framework] = true
		}
	}
}

func isTestFile(name string) bool {
	return strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") || strings.HasSuffix(name, "_test.go")
}

func isEntrypoint(rel string) bool {
	base := filepath.Base(rel)
	return rel == "main.go" ||
		rel == "src/main.ts" ||
		rel == "src/main.tsx" ||
		rel == "src/index.ts" ||
		rel == "src/index.tsx" ||
		strings.HasPrefix(rel, "cmd/") && base == "main.go"
}

func moduleCandidateForPath(rel string) (moduleCandidate, bool) {
	parts := strings.Split(rel, "/")
	for i := 0; i+2 < len(parts); i++ {
		if (parts[i] == "modules" || parts[i] == "features") && parts[i+1] != "" {
			name := parts[i+1]
			prefix := strings.Join(parts[:i+2], "/")
			return moduleCandidate{
				Name:       name,
				Paths:      []string{prefix + "/**"},
				Confidence: "high",
				Reason:     "directory follows " + parts[i] + "/* module convention",
			}, true
		}
	}
	if len(parts) >= 2 && parts[0] == "packages" && parts[1] != "" {
		return moduleCandidate{
			Name:       parts[1],
			Paths:      []string{"packages/" + parts[1] + "/**"},
			Confidence: "high",
			Reason:     "directory follows packages/* workspace convention",
		}, true
	}
	return moduleCandidate{}, false
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedCandidates(values map[string]moduleCandidate) []moduleCandidate {
	candidates := make([]moduleCandidate, 0, len(values))
	for _, candidate := range values {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Name < candidates[j].Name
	})
	return candidates
}
