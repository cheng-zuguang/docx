package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func runFinish(args []string, cwd string, stdout io.Writer) error {
	propose := false
	for _, arg := range args {
		switch arg {
		case "--propose":
			propose = true
		default:
			return fmt.Errorf("docx finish: unknown option %q", arg)
		}
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	index, err := readIndex(filepath.Join(root, config.ContextDir, "index.json"))
	if err != nil {
		return err
	}
	if !hasGitMetadata(root) {
		fmt.Fprintln(stdout, "No git repository found")
		return nil
	}
	changed, err := gitChangedFiles(root, "changed", "")
	if err != nil {
		return err
	}
	if len(changed) == 0 {
		fmt.Fprintln(stdout, "No changed files found")
		return nil
	}
	for _, file := range changed {
		if len(modulesForPath(file.Path, index.ModuleMap)) > 0 {
			syncArgs := []string{}
			if propose {
				syncArgs = append(syncArgs, "--propose")
			}
			return runSync(syncArgs, cwd, stdout)
		}
	}
	fmt.Fprintln(stdout, "No changed files matched confirmed modules")
	return nil
}

func hasGitMetadata(root string) bool {
	_, err := os.Stat(filepath.Join(root, ".git"))
	return err == nil
}
