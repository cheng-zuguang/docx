package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDistributionArtifactsDocumentInstallPaths(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)

	for _, path := range []string{
		"install.sh",
		"install.ps1",
		"package.json",
		"npm/bin/docx.js",
		"npm/install.js",
		"README.md",
		"README.zh-CN.md",
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("expected distribution artifact %s to exist: %v", path, err)
		}
	}

	packageBytes, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var packageJSON struct {
		Name    string            `json:"name"`
		Bin     map[string]string `json:"bin"`
		Scripts map[string]string `json:"scripts"`
		Files   []string          `json:"files"`
	}
	if err := json.Unmarshal(packageBytes, &packageJSON); err != nil {
		t.Fatal(err)
	}
	if packageJSON.Name != "@cheng-zuguang/docx" {
		t.Fatalf("package name = %q, want @cheng-zuguang/docx", packageJSON.Name)
	}
	if packageJSON.Bin["docx"] != "npm/bin/docx.js" {
		t.Fatalf("npm package should expose docx bin, got: %#v", packageJSON.Bin)
	}
	if packageJSON.Scripts["postinstall"] != "node npm/install.js" {
		t.Fatalf("npm package should install platform binary during postinstall, got: %#v", packageJSON.Scripts)
	}
	for _, expected := range []string{"npm/", "install.sh", "install.ps1"} {
		if !stringSliceContains(packageJSON.Files, expected) {
			t.Fatalf("package files should include %q, got: %#v", expected, packageJSON.Files)
		}
	}

	assertFileContains(t, root, "install.sh", "github.com/cheng-zuguang/docx")
	assertFileContains(t, root, "install.sh", "darwin")
	assertFileContains(t, root, "install.sh", "linux")
	assertFileContains(t, root, "install.ps1", "windows")
	assertFileContains(t, root, "npm/install.js", "DOCX_SKIP_DOWNLOAD")
	assertFileContains(t, root, "npm/bin/docx.js", "child_process")
	assertFileContains(t, root, "README.md", "npm install -g @cheng-zuguang/docx")
	assertFileContains(t, root, "README.md", "npx @cheng-zuguang/docx --help")
	assertFileContains(t, root, "README.zh-CN.md", "npm install -g @cheng-zuguang/docx")
	assertFileContains(t, root, "README.zh-CN.md", "npx @cheng-zuguang/docx --help")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root")
		}
		dir = parent
	}
}

func assertFileContains(t *testing.T, root string, path string, expected string) {
	t.Helper()

	bytes, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bytes), expected) {
		t.Fatalf("%s should contain %q, got:\n%s", path, expected, string(bytes))
	}
}

func stringSliceContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
