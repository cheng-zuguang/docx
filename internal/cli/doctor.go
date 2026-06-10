package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type doctorReport struct {
	SchemaVersion string        `json:"schemaVersion"`
	Checks        []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

func runDoctor(args []string, cwd string, stdout io.Writer) error {
	jsonOutput := false
	strict := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--strict":
			strict = true
		default:
			return fmt.Errorf("docx doctor: unknown option %q", args[i])
		}
	}

	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	report := runDoctorChecks(root)
	if jsonOutput {
		if err := writeJSONTo(stdout, report); err != nil {
			return err
		}
	} else {
		for _, check := range report.Checks {
			fmt.Fprintf(stdout, "%s: %s - %s\n", check.Name, check.Status, check.Message)
		}
	}
	if strict {
		for _, check := range report.Checks {
			if check.Status == "error" {
				return fmt.Errorf("docx doctor: %s: %s", check.Name, check.Message)
			}
		}
	}
	return nil
}

func runDoctorChecks(root string) doctorReport {
	report := doctorReport{SchemaVersion: schemaVersion}
	config, err := loadConfig(root)
	if err != nil {
		report.Checks = append(report.Checks, doctorCheck{Name: "config", Status: "error", Message: ".docx.json is missing or invalid", Details: []string{err.Error()}})
		return appendFallbackDoctorChecks(report)
	}
	report.Checks = append(report.Checks, doctorCheck{Name: "config", Status: "ok", Message: ".docx.json loaded"})

	report.Checks = append(report.Checks, checkRequiredFiles(root, config.ContextDir))
	report.Checks = append(report.Checks, checkSchemaVersion(config))
	report.Checks = append(report.Checks, checkIndexFreshness(root, config.ContextDir))
	report.Checks = append(report.Checks, checkAnalyzers(root, config.ContextDir))
	report.Checks = append(report.Checks, checkAgentTasks())
	report.Checks = append(report.Checks, checkHooks(root))
	return report
}

func appendFallbackDoctorChecks(report doctorReport) doctorReport {
	for _, name := range []string{"required-files", "schema-version", "indexes", "analyzers", "agent-tasks", "hooks"} {
		report.Checks = append(report.Checks, doctorCheck{Name: name, Status: "error", Message: "skipped because config is invalid"})
	}
	return report
}

func checkRequiredFiles(root string, contextDir string) doctorCheck {
	required := []string{
		".docx.json",
		contextDir + "/index.json",
		contextDir + "/project.json",
		contextDir + "/capabilities.json",
		contextDir + "/rules/agent.md",
		contextDir + "/changes/index.json",
		contextDir + "/proposals/index.json",
		contextDir + "/decisions/index.json",
		contextDir + "/mistakes/index.json",
	}
	var missing []string
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			missing = append(missing, rel)
		}
	}
	if len(missing) > 0 {
		return doctorCheck{Name: "required-files", Status: "error", Message: "required context files are missing", Details: missing}
	}
	return doctorCheck{Name: "required-files", Status: "ok", Message: "required context files exist"}
}

func checkSchemaVersion(config configFile) doctorCheck {
	if config.SchemaVersion != schemaVersion || config.ContextSchemaVersion != schemaVersion {
		return doctorCheck{Name: "schema-version", Status: "error", Message: "schema version mismatch"}
	}
	return doctorCheck{Name: "schema-version", Status: "ok", Message: "schema versions match"}
}

func checkIndexFreshness(root string, contextDir string) doctorCheck {
	if err := checkIndexes(root, contextDir, ""); err != nil {
		return doctorCheck{Name: "indexes", Status: "error", Message: "indexes are stale", Details: []string{err.Error()}}
	}
	return doctorCheck{Name: "indexes", Status: "ok", Message: "indexes are up to date"}
}

func checkAnalyzers(root string, contextDir string) doctorCheck {
	var capabilities capabilitiesFile
	bytes, err := os.ReadFile(filepath.Join(root, contextDir, "capabilities.json"))
	if err != nil {
		return doctorCheck{Name: "analyzers", Status: "error", Message: "capabilities file is missing"}
	}
	if err := json.Unmarshal(bytes, &capabilities); err != nil {
		return doctorCheck{Name: "analyzers", Status: "error", Message: "capabilities file is invalid"}
	}
	if len(capabilities.AvailableAnalyzers) == 0 {
		return doctorCheck{Name: "analyzers", Status: "warning", Message: "no analyzers are available"}
	}
	return doctorCheck{Name: "analyzers", Status: "ok", Message: "analyzer capabilities are recorded", Details: analyzerCapabilityNames(capabilities.AvailableAnalyzers)}
}

func checkHooks(root string) doctorCheck {
	details := installedHookDetails(root)
	if len(details) > 0 {
		return doctorCheck{Name: "hooks", Status: "ok", Message: "at least one optional docx hook is installed", Details: details}
	}
	return doctorCheck{Name: "hooks", Status: "warning", Message: "optional docx hooks are not installed"}
}

func installedHookDetails(root string) []string {
	var details []string
	for _, hook := range []string{"pre-commit", "post-merge", "post-checkout"} {
		if _, err := os.Stat(filepath.Join(root, ".git", "hooks", hook)); err == nil {
			details = append(details, ".git/hooks/"+hook)
		}
	}
	for _, rel := range []string{".codex/hooks.json", ".claude/settings.json"} {
		if agentLifecycleHookInstalled(filepath.Join(root, rel)) {
			details = append(details, rel)
		}
	}
	return details
}

func agentLifecycleHookInstalled(path string) bool {
	config, err := readHookConfig(path)
	if err != nil {
		return false
	}
	hooks := hookMap(config)
	stopHooks := hookMatcherList(hooks["Stop"])
	return lifecycleHookHasCommand(stopHooks, "docx finish") || lifecycleHookHasCommand(stopHooks, "docx finish --propose")
}

func checkAgentTasks() doctorCheck {
	return doctorCheck{
		Name:    "agent-tasks",
		Status:  "ok",
		Message: "semantic updates use active-agent task files",
	}
}
