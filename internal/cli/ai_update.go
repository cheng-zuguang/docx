package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type aiUpdateInput struct {
	SchemaVersion string          `json:"schemaVersion"`
	ChangeID      string          `json:"changeId"`
	Source        string          `json:"source"`
	Modules       []aiInputModule `json:"modules"`
	Files         []changeFile    `json:"files"`
}

type aiInputModule struct {
	Name          string        `json:"name"`
	Status        string        `json:"status"`
	Paths         []string      `json:"paths"`
	Summary       moduleSummary `json:"summary"`
	ReadHints     readHints     `json:"readHints"`
	RiskRules     []string      `json:"riskRules"`
	RecentChanges []string      `json:"recentChanges"`
}

type aiUpdateOutput struct {
	SchemaVersion string          `json:"schemaVersion"`
	ChangeID      string          `json:"changeId"`
	Proposals     json.RawMessage `json:"proposals"`
}

func writeProposalTask(root string, contextDir string, index indexFile, modules []string, files []changeFile, changeID string, source string, stdout io.Writer) error {
	input, err := buildAIUpdateInput(root, contextDir, index, modules, files, changeID, source)
	if err != nil {
		return err
	}
	tmpDir := filepath.Join(root, contextDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}
	inputRel := contextDir + "/tmp/proposals-input.json"
	promptRel := contextDir + "/tmp/proposals-prompt.md"
	if err := writeJSON(filepath.Join(root, inputRel), input); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, promptRel), []byte(proposalTaskPrompt(inputRel, contextDir)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Agent proposal task:\n  %s\n  %s\n", inputRel, promptRel)
	fmt.Fprintf(stdout, "Ask the active agent to read %s, write %s, then run `docx apply proposals %s`.\n", promptRel, contextDir+"/tmp/proposals-output.json", contextDir+"/tmp/proposals-output.json")
	return nil
}

func proposalTaskPrompt(inputPath string, contextDir string) string {
	outputPath := contextDir + "/tmp/proposals-output.json"
	return "# docx Proposal Task\n\n" +
		"Read `" + inputPath + "` and propose semantic memory updates for the changed modules.\n\n" +
		"Return exactly one JSON object with this shape:\n\n" +
		"```json\n" +
		"{\"schemaVersion\":\"1.0\",\"changeId\":\"...\",\"proposals\":[{\"schemaVersion\":\"1.0\",\"id\":\"...\",\"type\":\"module-summary\",\"status\":\"pending\",\"source\":\"ai:active-agent\",\"evidence\":[{\"path\":\"...\",\"reason\":\"...\"}],\"suggestedTarget\":\".doc/modules/name.json\",\"suggestedPatch\":{\"purpose\":\"...\",\"ownedConcepts\":[\"...\"],\"nonGoals\":[]}}]}\n" +
		"```\n\n" +
		"Allowed proposal types are `module-summary`, `risk-rule`, `decision`, `mistake`, and `module-partition`.\n" +
		"Do not directly edit `.doc/decisions/`, `.doc/mistakes/`, or module `riskRules`; write proposals instead.\n" +
		"Write the result to `" + outputPath + "`, then run `docx apply proposals " + outputPath + "`.\n"
}

func runApplyProposals(args []string, cwd string, stdin io.Reader, stdout io.Writer) error {
	return runApplyProposalsWithLabel(args, cwd, stdin, stdout, "Applied proposal output")
}

func runApplyProposalsWithLabel(args []string, cwd string, stdin io.Reader, stdout io.Writer, label string) error {
	useStdin := false
	outputPath := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--stdin":
			useStdin = true
		default:
			if outputPath != "" {
				return errors.New("docx apply proposals: expected one output file")
			}
			outputPath = args[i]
		}
	}
	if !useStdin && outputPath == "" {
		return errors.New("docx apply proposals: expected output file or --stdin")
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
	var output aiUpdateOutput
	if err := json.Unmarshal([]byte(extractJSONObject(string(outputBytes))), &output); err != nil {
		return fmt.Errorf("docx apply proposals: expected proposal output JSON: %w", err)
	}
	if output.SchemaVersion != schemaVersion {
		return fmt.Errorf("docx apply proposals: unsupported schemaVersion %q", output.SchemaVersion)
	}
	records, err := decodeAIUpdateProposals(output.Proposals)
	if err != nil {
		return err
	}
	paths, err := writeAIUpdateOutput(root, config.ContextDir, output.ChangeID, records)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%s: proposals=%d\n", label, len(paths))
	return nil
}

func buildAIUpdateInput(root string, contextDir string, index indexFile, modules []string, files []changeFile, changeID string, source string) (aiUpdateInput, error) {
	input := aiUpdateInput{SchemaVersion: schemaVersion, ChangeID: changeID, Source: source, Files: files}
	for _, moduleName := range modules {
		modulePath := filepath.Join(root, contextDir, "modules", moduleName+".json")
		bytes, err := os.ReadFile(modulePath)
		if err != nil {
			return aiUpdateInput{}, err
		}
		var module moduleFile
		if err := json.Unmarshal(bytes, &module); err != nil {
			return aiUpdateInput{}, err
		}
		paths := module.Paths
		if entry, ok := index.ModuleMap[moduleName]; ok && len(entry.Paths) > 0 {
			paths = entry.Paths
		}
		input.Modules = append(input.Modules, aiInputModule{
			Name:          module.Module,
			Status:        module.Status,
			Paths:         paths,
			Summary:       module.Summary,
			ReadHints:     module.ReadHints,
			RiskRules:     module.RiskRules,
			RecentChanges: module.RecentChanges,
		})
	}
	return input, nil
}

func decodeAIUpdateProposals(raw json.RawMessage) ([]proposalRecord, error) {
	var records []proposalRecord
	if err := json.Unmarshal(raw, &records); err == nil {
		return records, nil
	}
	var record proposalRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("docx apply proposals: expected proposal JSON or proposal array: %w", err)
	}
	return []proposalRecord{record}, nil
}

func writeAIUpdateOutput(root string, contextDir string, changeID string, records []proposalRecord) ([]string, error) {
	if changeID == "" {
		return nil, errors.New("docx apply proposals: changeId is required")
	}
	for _, record := range records {
		if err := validateAIUpdateProposal(record); err != nil {
			return nil, err
		}
	}
	paths := make([]string, 0, len(records))
	for _, record := range records {
		if err := writeProposalRecord(root, contextDir, record); err != nil {
			return nil, err
		}
		paths = append(paths, contextDir+"/proposals/"+record.ID+".json")
	}
	if err := appendChangeProposals(filepath.Join(root, contextDir, "changes", changeID+".json"), paths); err != nil {
		return nil, err
	}
	if err := rebuildProposalsIndex(root, contextDir); err != nil {
		return nil, err
	}
	return paths, nil
}

func validateAIUpdateProposal(record proposalRecord) error {
	if record.SchemaVersion != schemaVersion {
		return fmt.Errorf("docx apply proposals: proposal %q has unsupported schemaVersion %q", record.ID, record.SchemaVersion)
	}
	if record.ID == "" || record.Type == "" || record.Status != "pending" || record.Source == "" || record.SuggestedTarget == "" || record.SuggestedPatch == nil {
		return fmt.Errorf("docx apply proposals: proposal %q is missing required fields or is not pending", record.ID)
	}
	if len(record.Evidence) == 0 {
		return fmt.Errorf("docx apply proposals: proposal %q must include evidence", record.ID)
	}
	for _, evidence := range record.Evidence {
		if evidence.Path == "" || evidence.Reason == "" {
			return fmt.Errorf("docx apply proposals: proposal %q has invalid evidence", record.ID)
		}
	}
	return nil
}

func appendChangeProposals(path string, proposals []string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var record changeRecord
	if err := json.Unmarshal(bytes, &record); err != nil {
		return err
	}
	existing := map[string]bool{}
	for _, proposal := range record.Proposals {
		existing[proposal] = true
	}
	for _, proposal := range proposals {
		if !existing[proposal] {
			record.Proposals = append(record.Proposals, proposal)
		}
	}
	return writeJSON(path, record)
}
