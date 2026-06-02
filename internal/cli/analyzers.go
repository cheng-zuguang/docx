package cli

import "os/exec"

func availableAnalyzerCapabilities(report scanReport) []analyzerCapability {
	available := []analyzerCapability{
		{
			Name:         "generic",
			Kind:         "builtin",
			Languages:    []string{"*"},
			Capabilities: []string{"manifests", "languages", "frameworks", "entrypoints", "tests", "module-candidates"},
			Status:       "available",
		},
	}
	if reportUsesJavaScriptEcosystem(report) && nodeAvailable() {
		available = append(available, analyzerCapability{
			Name:         "typescript",
			Kind:         "external",
			Languages:    []string{"typescript", "javascript"},
			Capabilities: []string{"imports", "exports", "frameworks", "routes", "tests", "module-candidates"},
			Status:       "available",
		})
	}
	return available
}

func missingRecommendedAnalyzers(report scanReport) []analyzerCapability {
	languages := map[string]bool{}
	for _, language := range report.Languages {
		languages[language] = true
	}
	var missing []analyzerCapability
	if (languages["typescript"] || languages["javascript"]) && !nodeAvailable() {
		missing = append(missing, analyzerCapability{
			Name:         "typescript",
			Kind:         "external",
			Languages:    []string{"typescript", "javascript"},
			Capabilities: []string{"imports", "exports", "frameworks", "routes", "tests"},
			Status:       "missing",
			InstallHint:  "Install or configure the optional Node analyzer.",
		})
	}
	if languages["go"] {
		missing = append(missing, analyzerCapability{
			Name:         "go",
			Kind:         "external",
			Languages:    []string{"go"},
			Capabilities: []string{"packages", "imports", "exports", "tests"},
			Status:       "missing",
			InstallHint:  "Configure the optional Go analyzer when it becomes available.",
		})
	}
	return missing
}

func reportUsesJavaScriptEcosystem(report scanReport) bool {
	for _, language := range report.Languages {
		if language == "typescript" || language == "javascript" {
			return true
		}
	}
	return false
}

func nodeAvailable() bool {
	_, err := exec.LookPath("node")
	return err == nil
}

func analyzerCapabilityNames(values []analyzerCapability) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Name)
	}
	return names
}
