package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func runInstallHook(args []string, cwd string, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New("docx install-hook: expected hook name")
	}
	hook := args[0]
	if !supportedGitHook(hook) {
		return fmt.Errorf("docx install-hook: unsupported hook %q", hook)
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	if _, err := loadConfig(root); err != nil {
		return err
	}
	hooksDir := filepath.Join(root, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(hooksDir, hook)
	if err := upsertNamedManagedBlock(path, "# docx:start", "# docx:end", gitHookBlock(hook), 0o755); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Installed docx %s hook\n", hook)
	return nil
}

func supportedGitHook(hook string) bool {
	switch hook {
	case "pre-commit", "post-merge", "post-checkout":
		return true
	default:
		return false
	}
}

func gitHookBlock(hook string) string {
	command := "docx update --changed"
	if hook == "pre-commit" {
		command = "docx update --staged"
	}
	return "# docx:start\n" +
		"# Managed by docx. Preserve this block or run `docx install-hook " + hook + "` to refresh it.\n" +
		command + "\n" +
		"# docx:end\n"
}
