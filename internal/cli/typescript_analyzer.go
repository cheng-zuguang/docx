package cli

import (
	_ "embed"
	"encoding/json"
	"os/exec"
)

//go:embed analyzers/typescript/analyzer.mjs
var typeScriptAnalyzerSource string

func runTypeScriptAnalyzer(cwd string) (scanReport, error) {
	root, err := exec.LookPath("node")
	if err != nil {
		return scanReport{}, err
	}
	cmd := exec.Command(root, "--input-type=module", "-e", typeScriptAnalyzerSource)
	output, err := runAnalyzerCommand(cwd, cmd)
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
