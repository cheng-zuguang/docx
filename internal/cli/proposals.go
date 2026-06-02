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
)

type proposalRecord struct {
	SchemaVersion   string                 `json:"schemaVersion"`
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	Source          string                 `json:"source"`
	Evidence        []proposalEvidence     `json:"evidence"`
	SuggestedTarget string                 `json:"suggestedTarget"`
	SuggestedPatch  map[string]interface{} `json:"suggestedPatch"`
}

type proposalEvidence struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

func runProposals(args []string, cwd string, stdout io.Writer) error {
	if len(args) == 0 {
		return errors.New("docx proposals: expected list, show, accept, or reject")
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		return runProposalsList(root, config.ContextDir, stdout)
	case "show":
		if len(args) != 2 {
			return errors.New("docx proposals show: expected proposal id")
		}
		return runProposalsShow(root, config.ContextDir, args[1], stdout)
	case "accept":
		id, target, err := parseProposalAcceptArgs(args[1:])
		if err != nil {
			return err
		}
		return runProposalsAccept(root, config.ContextDir, id, target, stdout)
	case "reject":
		if len(args) != 2 {
			return errors.New("docx proposals reject: expected proposal id")
		}
		return runProposalsReject(root, config.ContextDir, args[1], stdout)
	default:
		return fmt.Errorf("docx proposals: unknown subcommand %q", args[0])
	}
}

func parseProposalAcceptArgs(args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", errors.New("docx proposals accept: expected proposal id")
	}
	id := args[0]
	target := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--target":
			if i+1 >= len(args) {
				return "", "", errors.New("docx proposals accept: --target requires a value")
			}
			i++
			target = args[i]
		default:
			return "", "", fmt.Errorf("docx proposals accept: unknown option %q", args[i])
		}
	}
	return id, target, nil
}

func runProposalsList(root string, contextDir string, stdout io.Writer) error {
	records, err := readProposalRecords(root, contextDir)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Status != "pending" {
			continue
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", record.ID, record.Type, record.SuggestedTarget)
	}
	return nil
}

func runProposalsShow(root string, contextDir string, id string, stdout io.Writer) error {
	record, err := readProposalRecord(root, contextDir, id)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "ID: %s\n", record.ID)
	fmt.Fprintf(stdout, "Type: %s\n", record.Type)
	fmt.Fprintf(stdout, "Status: %s\n", record.Status)
	fmt.Fprintf(stdout, "Source: %s\n", record.Source)
	fmt.Fprintf(stdout, "Target: %s\n", record.SuggestedTarget)
	fmt.Fprintln(stdout, "Evidence:")
	for _, evidence := range record.Evidence {
		fmt.Fprintf(stdout, "- %s: %s\n", evidence.Path, evidence.Reason)
	}
	fmt.Fprintln(stdout, "Suggested Patch:")
	patch, err := json.MarshalIndent(record.SuggestedPatch, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(patch))
	return nil
}

func readProposalRecords(root string, contextDir string) ([]proposalRecord, error) {
	dir := filepath.Join(root, contextDir, "proposals")
	records, err := collectJSONFiles[proposalRecord](dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].ID < records[j].ID
	})
	return records, nil
}

func readProposalRecord(root string, contextDir string, id string) (proposalRecord, error) {
	path := filepath.Join(root, contextDir, "proposals", id+".json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return proposalRecord{}, err
	}
	var record proposalRecord
	if err := json.Unmarshal(bytes, &record); err != nil {
		return proposalRecord{}, err
	}
	return record, nil
}

func runProposalsAccept(root string, contextDir string, id string, target string, stdout io.Writer) error {
	record, err := readProposalRecord(root, contextDir, id)
	if err != nil {
		return err
	}
	if target != "" {
		record.SuggestedTarget = target
	}
	if record.Status != "pending" {
		return fmt.Errorf("docx proposals accept: proposal %s is %s", id, record.Status)
	}
	if err := applyProposal(root, contextDir, record); err != nil {
		return err
	}
	record.Status = "accepted"
	if err := writeProposalRecord(root, contextDir, record); err != nil {
		return err
	}
	if err := rebuildProposalsIndex(root, contextDir); err != nil {
		return err
	}
	if record.Type == "decision" {
		if err := rebuildDecisionsIndex(root, contextDir); err != nil {
			return err
		}
	}
	if record.Type == "mistake" {
		if err := rebuildMistakesIndex(root, contextDir); err != nil {
			return err
		}
	}
	fmt.Fprintf(stdout, "Accepted proposal %s\n", id)
	return nil
}

func runProposalsReject(root string, contextDir string, id string, stdout io.Writer) error {
	record, err := readProposalRecord(root, contextDir, id)
	if err != nil {
		return err
	}
	if record.Status != "pending" {
		return fmt.Errorf("docx proposals reject: proposal %s is %s", id, record.Status)
	}
	record.Status = "rejected"
	if err := writeProposalRecord(root, contextDir, record); err != nil {
		return err
	}
	if err := rebuildProposalsIndex(root, contextDir); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Rejected proposal %s\n", id)
	return nil
}

func applyProposal(root string, contextDir string, record proposalRecord) error {
	switch record.Type {
	case "decision":
		return applyDecisionProposal(root, contextDir, record)
	case "mistake":
		return applyMistakeProposal(root, contextDir, record)
	case "module-summary":
		return applyModuleSummaryProposal(root, contextDir, record)
	case "risk-rule":
		return applyRiskRuleProposal(root, contextDir, record)
	default:
		return fmt.Errorf("docx proposals accept: unsupported proposal type %q", record.Type)
	}
}

func applyDecisionProposal(root string, contextDir string, record proposalRecord) error {
	target, err := resolveContextTarget(root, contextDir, record.SuggestedTarget)
	if err != nil {
		return err
	}
	title := patchString(record.SuggestedPatch, "title", record.ID)
	status := patchString(record.SuggestedPatch, "status", "accepted")
	body := patchString(record.SuggestedPatch, "body", "")
	content := fmt.Sprintf("# %s\n\nStatus: %s\n\n%s\n", title, status, body)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(content), 0o644)
}

func applyMistakeProposal(root string, contextDir string, record proposalRecord) error {
	target, err := resolveContextTarget(root, contextDir, record.SuggestedTarget)
	if err != nil {
		return err
	}
	id := patchString(record.SuggestedPatch, "id", record.ID)
	title := patchString(record.SuggestedPatch, "title", record.ID)
	body := patchString(record.SuggestedPatch, "body", "")
	content := fmt.Sprintf("\n## [%s] %s\n\n", id, title)
	if appliesTo := patchStringSlice(record.SuggestedPatch, "appliesTo"); len(appliesTo) > 0 {
		content += fmt.Sprintf("**appliesTo**: %s\n\n", strings.Join(appliesTo, ", "))
	}
	content += body + "\n"
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}

func applyModuleSummaryProposal(root string, contextDir string, record proposalRecord) error {
	module, target, err := readProposalTargetModule(root, contextDir, record)
	if err != nil {
		return err
	}
	if value, ok := record.SuggestedPatch["purpose"].(string); ok && value != "" {
		module.Summary.Purpose = value
	}
	if values := patchStringSlice(record.SuggestedPatch, "ownedConcepts"); len(values) > 0 {
		module.Summary.OwnedConcepts = values
	}
	if values := patchStringSlice(record.SuggestedPatch, "nonGoals"); len(values) > 0 {
		module.Summary.NonGoals = values
	}
	return writeJSON(target, module)
}

func applyRiskRuleProposal(root string, contextDir string, record proposalRecord) error {
	module, target, err := readProposalTargetModule(root, contextDir, record)
	if err != nil {
		return err
	}
	rule := patchString(record.SuggestedPatch, "rule", "")
	if rule == "" {
		return errors.New("docx proposals accept: risk-rule patch requires rule")
	}
	for _, existing := range module.RiskRules {
		if existing == rule {
			return writeJSON(target, module)
		}
	}
	module.RiskRules = append(module.RiskRules, rule)
	return writeJSON(target, module)
}

func readProposalTargetModule(root string, contextDir string, record proposalRecord) (moduleFile, string, error) {
	target, err := resolveContextTarget(root, contextDir, record.SuggestedTarget)
	if err != nil {
		return moduleFile{}, "", err
	}
	bytes, err := os.ReadFile(target)
	if err != nil {
		return moduleFile{}, "", err
	}
	var module moduleFile
	if err := json.Unmarshal(bytes, &module); err != nil {
		return moduleFile{}, "", err
	}
	return module, target, nil
}

func writeProposalRecord(root string, contextDir string, record proposalRecord) error {
	return writeJSON(filepath.Join(root, contextDir, "proposals", record.ID+".json"), record)
}

func resolveContextTarget(root string, contextDir string, target string) (string, error) {
	clean := filepath.Clean(target)
	contextPrefix := filepath.Clean(contextDir) + string(filepath.Separator)
	if clean != filepath.Clean(contextDir) && !strings.HasPrefix(clean, contextPrefix) {
		return "", fmt.Errorf("docx proposals: target must be inside %s", contextDir)
	}
	return filepath.Join(root, clean), nil
}

func patchString(patch map[string]interface{}, key string, fallback string) string {
	value, ok := patch[key].(string)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func patchStringSlice(patch map[string]interface{}, key string) []string {
	raw, ok := patch[key].([]interface{})
	if !ok {
		return nil
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		value, ok := item.(string)
		if ok && value != "" {
			values = append(values, value)
		}
	}
	return values
}
