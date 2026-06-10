package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type aiInitInput struct {
	SchemaVersion string                    `json:"schemaVersion"`
	Project       aiInitInputProject        `json:"project"`
	Modules       []aiInitInputModule       `json:"modules"`
	ModuleMap     map[string]moduleMapEntry `json:"moduleMap"`
}

type aiInitInputProject struct {
	Name  string   `json:"name"`
	Types []string `json:"types"`
}

type aiInitInputModule struct {
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Paths  []string    `json:"paths"`
	Facts  moduleFacts `json:"facts"`
}

type aiInitOutput struct {
	SchemaVersion string               `json:"schemaVersion"`
	Project       aiInitOutputProject  `json:"project"`
	Modules       []aiInitOutputModule `json:"modules"`
}

type aiInitOutputProject struct {
	Summary string `json:"summary"`
}

type aiInitOutputModule struct {
	Name    string        `json:"name"`
	Summary moduleSummary `json:"summary"`
}

type aiInitApplyStats struct {
	ProjectUpdated bool
	ModulesUpdated int
}

func writeInitSummaryTask(root string, contextDir string, stdout io.Writer) error {
	input, err := buildAIInitInput(root, contextDir)
	if err != nil {
		return err
	}
	tmpDir := filepath.Join(root, contextDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	inputRel := contextDir + "/tmp/init-summary-input.json"
	promptRel := contextDir + "/tmp/init-summary-prompt.md"
	if err := writeJSON(filepath.Join(root, inputRel), input); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, promptRel), []byte(initSummaryTaskPrompt(inputRel, contextDir)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Agent init summary task:\n  %s\n  %s\n", inputRel, promptRel)
	fmt.Fprintf(stdout, "Ask the active agent to read %s, write %s, then run `docx apply init %s`.\n", promptRel, contextDir+"/tmp/init-summary-output.json", contextDir+"/tmp/init-summary-output.json")
	return nil
}

func initSummaryTaskPrompt(inputPath string, contextDir string) string {
	outputPath := contextDir + "/tmp/init-summary-output.json"
	return "# docx Init Summary Task\n\n" +
		"Read `" + inputPath + "` and generate initial project context summaries.\n\n" +
		"Return exactly one JSON object with this shape:\n\n" +
		"```json\n" +
		"{\"schemaVersion\":\"1.0\",\"project\":{\"summary\":\"...\"},\"modules\":[{\"name\":\"...\",\"summary\":{\"purpose\":\"...\",\"ownedConcepts\":[\"...\"],\"nonGoals\":[]}}]}\n" +
		"```\n\n" +
		"Only generate `project.summary` and `module.summary`. Do not include or infer `riskRules`, `decisions`, or `mistakes`.\n" +
		"Write the result to `" + outputPath + "`, then run `docx apply init " + outputPath + "`.\n"
}

func runApplyInit(args []string, cwd string, stdin io.Reader, stdout io.Writer) error {
	return runInitSummaryApplyWithLabel(args, cwd, stdin, stdout, "Applied init summaries", "docx apply init")
}

func runInitSummaryApplyWithLabel(args []string, cwd string, stdin io.Reader, stdout io.Writer, label string, commandName string) error {
	useStdin := false
	outputPath := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stdin":
			useStdin = true
		default:
			if outputPath != "" {
				return fmt.Errorf("%s: expected one output file", commandName)
			}
			outputPath = args[i]
		}
	}
	if !useStdin && outputPath == "" {
		return fmt.Errorf("%s: expected output file or --stdin", commandName)
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	var outputBytes []byte
	if useStdin {
		outputBytes, err = io.ReadAll(stdin)
	} else {
		path := filepath.Clean(outputPath)
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		outputBytes, err = os.ReadFile(path)
	}
	if err != nil {
		return err
	}
	var output aiInitOutput
	if err := json.Unmarshal([]byte(extractJSONObject(string(outputBytes))), &output); err != nil {
		return fmt.Errorf("%s: expected init summary JSON: %w", commandName, err)
	}
	if output.SchemaVersion != schemaVersion {
		return fmt.Errorf("%s: unsupported schemaVersion %q", commandName, output.SchemaVersion)
	}
	stats, err := writeAIInitOutput(root, config.ContextDir, output)
	if err != nil {
		return err
	}
	projectStatus := "unchanged"
	if stats.ProjectUpdated {
		projectStatus = "updated"
	}
	fmt.Fprintf(stdout, "%s: project=%s modules=%d\n", label, projectStatus, stats.ModulesUpdated)
	return nil
}

func buildAIInitInput(root string, contextDir string) (aiInitInput, error) {
	projectPath := filepath.Join(root, contextDir, "project.json")
	projectBytes, err := os.ReadFile(projectPath)
	if err != nil {
		return aiInitInput{}, err
	}
	var project projectFile
	if err := json.Unmarshal(projectBytes, &project); err != nil {
		return aiInitInput{}, err
	}
	index, err := readIndex(filepath.Join(root, contextDir, "index.json"))
	if err != nil {
		return aiInitInput{}, err
	}
	input := aiInitInput{
		SchemaVersion: schemaVersion,
		Project:       aiInitInputProject{Name: project.Name, Types: project.Types},
		ModuleMap:     index.ModuleMap,
	}
	for moduleName := range index.ModuleMap {
		moduleBytes, err := os.ReadFile(filepath.Join(root, contextDir, "modules", moduleName+".json"))
		if err != nil {
			return aiInitInput{}, err
		}
		var module moduleFile
		if err := json.Unmarshal(moduleBytes, &module); err != nil {
			return aiInitInput{}, err
		}
		input.Modules = append(input.Modules, aiInitInputModule{
			Name:   module.Module,
			Status: module.Status,
			Paths:  module.Paths,
			Facts:  module.Facts,
		})
	}
	return input, nil
}

func writeAIInitOutput(root string, contextDir string, output aiInitOutput) (aiInitApplyStats, error) {
	stats := aiInitApplyStats{}
	projectPath := filepath.Join(root, contextDir, "project.json")
	projectBytes, err := os.ReadFile(projectPath)
	if err != nil {
		return stats, err
	}
	var project projectFile
	if err := json.Unmarshal(projectBytes, &project); err != nil {
		return stats, err
	}
	if output.Project.Summary != "" {
		project.Summary = output.Project.Summary
		if err := writeJSON(projectPath, project); err != nil {
			return stats, err
		}
		indexPath := filepath.Join(root, contextDir, "index.json")
		index, err := readIndex(indexPath)
		if err != nil {
			return stats, err
		}
		index.Project.Summary = output.Project.Summary
		if err := writeJSON(indexPath, index); err != nil {
			return stats, err
		}
		stats.ProjectUpdated = true
	}
	for _, outputModule := range output.Modules {
		if outputModule.Name == "" || outputModule.Summary.Purpose == "" {
			continue
		}
		modulePath := filepath.Join(root, contextDir, "modules", outputModule.Name+".json")
		moduleBytes, err := os.ReadFile(modulePath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return stats, err
		}
		var module moduleFile
		if err := json.Unmarshal(moduleBytes, &module); err != nil {
			return stats, err
		}
		module.Summary = outputModule.Summary
		if err := writeJSON(modulePath, module); err != nil {
			return stats, err
		}
		stats.ModulesUpdated++
	}
	return stats, nil
}

func extractJSONObject(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end <= start {
		return text
	}
	return text[start : end+1]
}
