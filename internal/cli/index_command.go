package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func runIndex(args []string, cwd string, stdout io.Writer) error {
	section := ""
	check := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--check":
			check = true
		case "--section":
			if i+1 >= len(args) {
				return errors.New("docx index: --section requires a value")
			}
			i++
			section = args[i]
		default:
			return fmt.Errorf("docx index: unknown option %q", args[i])
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
	if check {
		if err := checkIndexes(root, config.ContextDir, section); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "docx indexes are up to date")
		return nil
	}
	if err := rebuildIndexes(root, config.ContextDir, section); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Rebuilt docx indexes")
	return nil
}

func checkIndexes(root string, contextDir string, section string) error {
	expected, err := buildIndexOutputs(root, contextDir, section)
	if err != nil {
		return err
	}
	for path, content := range expected {
		current, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(current) != content {
			return fmt.Errorf("docx index: stale index %s", filepath.ToSlash(path))
		}
	}
	return nil
}

func buildIndexOutputs(root string, contextDir string, section string) (map[string]string, error) {
	outputs := map[string]string{}
	if section == "" || section == "changes" {
		path, value, err := buildChangesIndex(root, contextDir)
		if err != nil {
			return nil, err
		}
		outputs[path] = mustJSONString(value)
	}
	if section == "" || section == "proposals" {
		path, value, err := buildProposalsIndex(root, contextDir)
		if err != nil {
			return nil, err
		}
		outputs[path] = mustJSONString(value)
	}
	if section == "" || section == "decisions" {
		path, value, err := buildDecisionsIndex(root, contextDir)
		if err != nil {
			return nil, err
		}
		outputs[path] = mustJSONString(value)
	}
	if section == "" || section == "mistakes" {
		path, value, err := buildMistakesIndex(root, contextDir)
		if err != nil {
			return nil, err
		}
		outputs[path] = mustJSONString(value)
	}
	if section != "" && section != "changes" && section != "proposals" && section != "decisions" && section != "mistakes" {
		return nil, fmt.Errorf("docx index: unknown section %q", section)
	}
	return outputs, nil
}

func rebuildIndexes(root string, contextDir string, section string) error {
	if section == "" || section == "changes" {
		if err := rebuildChangesIndex(root, contextDir); err != nil {
			return err
		}
	}
	if section == "" || section == "proposals" {
		if err := rebuildProposalsIndex(root, contextDir); err != nil {
			return err
		}
	}
	if section == "" || section == "decisions" {
		if err := rebuildDecisionsIndex(root, contextDir); err != nil {
			return err
		}
	}
	if section == "" || section == "mistakes" {
		return rebuildMistakesIndex(root, contextDir)
	}
	if section != "" && section != "changes" && section != "proposals" && section != "decisions" && section != "mistakes" {
		return fmt.Errorf("docx index: unknown section %q", section)
	}
	return nil
}
