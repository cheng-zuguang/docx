package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseWorkflowPublishesGitHubAssetsAndNpmPackage(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	path := filepath.Join(root, ".github", "workflows", "release.yml")
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected release workflow to exist: %v", err)
	}
	workflow := string(bytes)

	for _, expected := range []string{
		"- \"v*\"",
		"refs/tags/v",
		"go test ./...",
		"docx_darwin_amd64.tar.gz",
		"docx_darwin_arm64.tar.gz",
		"docx_linux_amd64.tar.gz",
		"docx_linux_arm64.tar.gz",
		"docx_windows_amd64.zip",
		"docx_windows_arm64.zip",
		"softprops/action-gh-release",
		"id-token: write",
		"node-version: \"22\"",
		"registry-url: https://registry.npmjs.org",
		"npm publish --access public",
	} {
		if !strings.Contains(workflow, expected) {
			t.Fatalf("release workflow should contain %q, got:\n%s", expected, workflow)
		}
	}
}
