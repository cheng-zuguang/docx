package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func runInstallHook(args []string, cwd string, stdout io.Writer) error {
	hook := ""
	proposeMode := false
	for _, arg := range args {
		switch arg {
		case "--propose":
			proposeMode = true
		default:
			if strings.HasPrefix(arg, "--") {
				return fmt.Errorf("docx install-hook: unknown option %q", arg)
			}
			if hook != "" {
				return errors.New("docx install-hook: expected hook name")
			}
			hook = arg
		}
	}
	if hook == "" {
		return errors.New("docx install-hook: expected hook name")
	}
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
	if err := upsertNamedManagedBlock(path, "# docx:start", "# docx:end", gitHookBlock(hook, proposeMode), 0o755); err != nil {
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

func gitHookBlock(hook string, proposeMode bool) string {
	command := "docx update --changed"
	if hook == "pre-commit" {
		command = "docx update --staged"
	}
	if proposeMode {
		command += " --propose"
	}
	return "# docx:start\n" +
		"# Managed by docx. Preserve this block or run `docx install-hook " + hook + "` to refresh it.\n" +
		command + "\n" +
		"# docx:end\n"
}
