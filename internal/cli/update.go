package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type changeRecord struct {
	SchemaVersion string       `json:"schemaVersion"`
	ID            string       `json:"id"`
	Source        string       `json:"source"`
	Range         string       `json:"range,omitempty"`
	Modules       []string     `json:"modules"`
	Files         []changeFile `json:"files"`
	FactsUpdated  []string     `json:"factsUpdated"`
	Proposals     []string     `json:"proposals"`
}

type changeFile struct {
	Path       string   `json:"path"`
	ChangeType string   `json:"changeType"`
	Signals    []string `json:"signals"`
}

func runUpdate(args []string, cwd string, stdout io.Writer) error {
	mode := ""
	sinceRef := ""
	selectedModule := ""
	aiMode := false
	aiCommand := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ai":
			aiMode = true
		case "--ai-command":
			if i+1 >= len(args) {
				return errors.New("docx update: --ai-command requires a command")
			}
			i++
			aiCommand = args[i]
		case "--changed":
			mode = "changed"
		case "--staged":
			mode = "staged"
		case "--since":
			if i+1 >= len(args) {
				return errors.New("docx update: --since requires a git ref")
			}
			i++
			mode = "since"
			sinceRef = args[i]
		case "--module":
			if i+1 >= len(args) {
				return errors.New("docx update: --module requires a module name")
			}
			i++
			mode = "module"
			selectedModule = args[i]
		default:
			return fmt.Errorf("docx update: unknown option %q", args[i])
		}
	}
	if mode == "" {
		return errors.New("docx update: specify --changed, --staged, --since <ref>, or --module <name>")
	}
	if aiCommand != "" && !aiMode {
		return errors.New("docx update: --ai-command requires --ai")
	}

	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	if err := requireCompatibleSchema(config); err != nil {
		return err
	}
	if aiMode && aiCommand == "" {
		aiCommand = configuredAICommand(config)
	}
	indexPath := filepath.Join(root, config.ContextDir, "index.json")
	index, err := readIndex(indexPath)
	if err != nil {
		return err
	}

	modulesSet := map[string]bool{}
	var files []changeFile
	if mode == "module" {
		entry, ok := index.ModuleMap[selectedModule]
		if !ok {
			return fmt.Errorf("docx update: module %q not found in moduleMap", selectedModule)
		}
		if entry.Confidence != "confirmed" {
			return fmt.Errorf("docx update: module %q is not confirmed", selectedModule)
		}
		modulesSet[selectedModule] = true
	} else {
		changed, err := gitChangedFiles(root, mode, sinceRef)
		if err != nil {
			return err
		}
		if len(changed) == 0 {
			fmt.Fprintln(stdout, "No changed files found")
			return nil
		}
		for _, file := range changed {
			modules := modulesForPath(file.Path, index.ModuleMap)
			if len(modules) == 0 {
				continue
			}
			for _, module := range modules {
				modulesSet[module] = true
			}
			files = append(files, changeFile{Path: file.Path, ChangeType: file.ChangeType, Signals: signalsForPath(file.Path)})
		}
		if len(files) == 0 {
			fmt.Fprintln(stdout, "No changed files matched confirmed modules")
			return nil
		}
	}

	modules := sortedKeys(modulesSet)
	changeID := newChangeID()
	changeRel := config.ContextDir + "/changes/" + changeID + ".json"
	factsUpdated := make([]string, 0, len(modules))
	for _, module := range modules {
		factsUpdated = append(factsUpdated, config.ContextDir+"/modules/"+module+".json")
	}
	record := changeRecord{
		SchemaVersion: schemaVersion,
		ID:            changeID,
		Source:        updateSource(mode, selectedModule),
		Modules:       modules,
		Files:         files,
		FactsUpdated:  factsUpdated,
		Proposals:     []string{},
	}
	if aiMode {
		proposals, err := writeAIUpdateProposals(root, config.ContextDir, index, modules, files, changeID, record.Source, aiCommand)
		if err != nil {
			return err
		}
		record.Proposals = proposals
	}
	if mode == "since" {
		record.Range = sinceRef + "..HEAD"
	}

	changeJSONPath := filepath.Join(root, config.ContextDir, "changes", changeID+".json")
	if err := writeJSON(changeJSONPath, record); err != nil {
		return err
	}
	if err := writeChangeMarkdown(filepath.Join(root, config.ContextDir, "changes", changeID+".md"), record); err != nil {
		return err
	}
	for _, module := range modules {
		if !aiMode {
			if err := appendModuleRecentChange(filepath.Join(root, config.ContextDir, "modules", module+".json"), changeRel); err != nil {
				return err
			}
		}
	}
	if aiMode {
		if err := rebuildProposalsIndex(root, config.ContextDir); err != nil {
			return err
		}
	}

	fmt.Fprintf(stdout, "Recorded change %s\n", changeID)
	return nil
}

func updateSource(mode string, selectedModule string) string {
	if mode == "module" {
		return "module:" + selectedModule
	}
	return "git:" + mode
}

func modulesForPath(path string, moduleMap map[string]moduleMapEntry) []string {
	var modules []string
	for name, entry := range moduleMap {
		if entry.Confidence != "confirmed" {
			continue
		}
		for _, glob := range entry.Paths {
			prefix := strings.TrimSuffix(glob, "/**")
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				modules = append(modules, name)
				break
			}
		}
	}
	sort.Strings(modules)
	return modules
}

func signalsForPath(path string) []string {
	var signals []string
	base := filepath.Base(path)
	if isTestFile(base) {
		signals = append(signals, "testsTouched")
	}
	if strings.HasSuffix(base, ".config.ts") || strings.HasSuffix(base, ".config.js") || strings.HasSuffix(base, ".config.mjs") || strings.HasSuffix(base, ".config.cjs") {
		signals = append(signals, "configTouched")
	}
	if len(signals) == 0 {
		signals = append(signals, "sourceTouched")
	}
	return signals
}

func newChangeID() string {
	return time.Now().UTC().Format("20060102T150405.000000000Z")
}

func writeChangeMarkdown(path string, record changeRecord) error {
	var builder strings.Builder
	builder.WriteString("# Change " + record.ID + "\n\n")
	builder.WriteString("## Source\n\n")
	builder.WriteString(record.Source + "\n\n")
	builder.WriteString("## Modules Affected\n\n")
	for _, module := range record.Modules {
		builder.WriteString("- " + module + "\n")
	}
	builder.WriteString("\n## Files Changed\n\n")
	for _, file := range record.Files {
		builder.WriteString("- " + file.ChangeType + ": `" + file.Path + "`\n")
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}

func appendModuleRecentChange(path string, changeRel string) error {
	var module moduleFile
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, &module); err != nil {
		return err
	}
	for _, existing := range module.RecentChanges {
		if existing == changeRel {
			return nil
		}
	}
	module.RecentChanges = append([]string{changeRel}, module.RecentChanges...)
	return writeJSON(path, module)
}
