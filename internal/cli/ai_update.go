package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type aiUpdateInput struct {
	SchemaVersion string          `json:"schemaVersion"`
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

func writeAIUpdateProposals(root string, contextDir string, index indexFile, modules []string, files []changeFile, changeID string, source string, aiCommand string) ([]string, error) {
	if aiCommand != "" {
		return writeLocalAICommandProposals(root, contextDir, index, modules, files, source, aiCommand)
	}
	return writePlaceholderAIUpdateProposals(root, contextDir, modules, files, changeID)
}

func configuredAICommand(config configFile) string {
	if config.AI.Provider != "local-command" {
		return ""
	}
	if config.AI.Output != "" && config.AI.Output != "proposal-json" {
		return ""
	}
	return config.AI.Command
}

func writePlaceholderAIUpdateProposals(root string, contextDir string, modules []string, files []changeFile, changeID string) ([]string, error) {
	proposals := make([]string, 0, len(modules))
	for _, module := range modules {
		id := "ai-" + changeID + "-" + module
		record := proposalRecord{
			SchemaVersion:   schemaVersion,
			ID:              id,
			Type:            "module-summary",
			Status:          "pending",
			Source:          "ai:provider-agnostic",
			Evidence:        proposalEvidenceForFiles(files),
			SuggestedTarget: contextDir + "/modules/" + module + ".json",
			SuggestedPatch: map[string]interface{}{
				"purpose":       fmt.Sprintf("Review recent changes and update the %s module summary if the confirmed intent changed.", module),
				"ownedConcepts": []interface{}{},
				"nonGoals":      []interface{}{},
			},
		}
		if len(record.Evidence) == 0 {
			record.Evidence = []proposalEvidence{{Path: record.SuggestedTarget, Reason: "docx update --ai requested semantic review for this module."}}
		}
		if err := writeProposalRecord(root, contextDir, record); err != nil {
			return nil, err
		}
		proposals = append(proposals, contextDir+"/proposals/"+filepath.Base(id)+".json")
	}
	return proposals, nil
}

func writeLocalAICommandProposals(root string, contextDir string, index indexFile, modules []string, files []changeFile, source string, aiCommand string) ([]string, error) {
	input, err := buildAIUpdateInput(root, contextDir, index, modules, files, source)
	if err != nil {
		return nil, err
	}
	output, err := runLocalAICommand(root, aiCommand, input)
	if err != nil {
		return nil, err
	}
	records, err := decodeAICommandProposals(output)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(records))
	for _, record := range records {
		if err := validateAICommandProposal(record); err != nil {
			return nil, err
		}
		if err := writeProposalRecord(root, contextDir, record); err != nil {
			return nil, err
		}
		paths = append(paths, contextDir+"/proposals/"+record.ID+".json")
	}
	return paths, nil
}

func buildAIUpdateInput(root string, contextDir string, index indexFile, modules []string, files []changeFile, source string) (aiUpdateInput, error) {
	input := aiUpdateInput{SchemaVersion: schemaVersion, Source: source, Files: files}
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

func runLocalAICommand(root string, aiCommand string, input aiUpdateInput) ([]byte, error) {
	parts := strings.Fields(aiCommand)
	if len(parts) == 0 {
		return nil, errors.New("docx update --ai-command: command is empty")
	}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(inputBytes)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docx update --ai-command: local command failed: %w", err)
	}
	return output, nil
}

func decodeAICommandProposals(output []byte) ([]proposalRecord, error) {
	var records []proposalRecord
	if err := json.Unmarshal(output, &records); err == nil {
		return records, nil
	}
	var record proposalRecord
	if err := json.Unmarshal(output, &record); err != nil {
		return nil, fmt.Errorf("docx update --ai-command: expected proposal JSON or proposal array: %w", err)
	}
	return []proposalRecord{record}, nil
}

func validateAICommandProposal(record proposalRecord) error {
	if record.SchemaVersion != schemaVersion {
		return fmt.Errorf("docx update --ai-command: proposal %q has unsupported schemaVersion %q", record.ID, record.SchemaVersion)
	}
	if record.ID == "" || record.Type == "" || record.Status != "pending" || record.Source == "" || record.SuggestedTarget == "" || record.SuggestedPatch == nil {
		return fmt.Errorf("docx update --ai-command: proposal %q is missing required fields or is not pending", record.ID)
	}
	if len(record.Evidence) == 0 {
		return fmt.Errorf("docx update --ai-command: proposal %q must include evidence", record.ID)
	}
	for _, evidence := range record.Evidence {
		if evidence.Path == "" || evidence.Reason == "" {
			return fmt.Errorf("docx update --ai-command: proposal %q has invalid evidence", record.ID)
		}
	}
	return nil
}

func proposalEvidenceForFiles(files []changeFile) []proposalEvidence {
	evidence := make([]proposalEvidence, 0, len(files))
	for _, file := range files {
		evidence = append(evidence, proposalEvidence{Path: file.Path, Reason: "Changed file detected by docx update --ai."})
	}
	return evidence
}
