package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorJSONReportsHealthyInstallation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"index"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("index failed: %v\nstderr: %s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := Run([]string{"doctor", "--json"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("doctor failed: %v\nstderr: %s", err, stderr.String())
	}

	var report doctorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("doctor should emit JSON, got %v:\n%s", err, stdout.String())
	}
	if report.SchemaVersion != schemaVersion {
		t.Fatalf("schemaVersion = %q, want %q", report.SchemaVersion, schemaVersion)
	}
	for _, check := range []string{"config", "required-files", "schema-version", "indexes", "analyzers", "hooks"} {
		if status := reportStatus(report, check); status != "ok" && status != "warning" {
			t.Fatalf("expected %s check to be ok or warning, got %q in %#v", check, status, report.Checks)
		}
	}
	if reportStatus(report, "config") != "ok" {
		t.Fatalf("config should be ok: %#v", report.Checks)
	}
	if reportStatus(report, "required-files") != "ok" {
		t.Fatalf("required-files should be ok: %#v", report.Checks)
	}
	if reportStatus(report, "indexes") != "ok" {
		t.Fatalf("indexes should be ok: %#v", report.Checks)
	}
}

func TestDoctorStrictFailsWhenRequiredFilesAreMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := Run([]string{"init"}, dir, &stdout, &stderr); err != nil {
		t.Fatalf("init failed: %v\nstderr: %s", err, stderr.String())
	}
	if err := removePath(dir, ".doc/project.json"); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	err := Run([]string{"doctor", "--strict"}, dir, &stdout, &stderr)
	if err == nil {
		t.Fatalf("doctor --strict should fail when required files are missing")
	}
	if !strings.Contains(err.Error(), "required-files") {
		t.Fatalf("strict error should mention required-files, got: %v", err)
	}
}

func removePath(root string, path string) error {
	return os.Remove(filepath.Join(root, path))
}

func reportStatus(report doctorReport, name string) string {
	for _, check := range report.Checks {
		if check.Name == name {
			return check.Status
		}
	}
	return ""
}
