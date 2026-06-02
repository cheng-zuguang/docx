package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func runMigrate(args []string, cwd string, stdout io.Writer) error {
	if len(args) != 0 {
		return fmt.Errorf("docx migrate: unknown option %q", args[0])
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	config.SchemaVersion = schemaVersion
	config.ContextSchemaVersion = schemaVersion
	if config.ContextDir == "" {
		config.ContextDir = defaultDocDir
	}
	if err := writeJSON(filepath.Join(root, ".docx.json"), config); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Migrated docx context to schema 1.0")
	return nil
}

func requireCompatibleSchema(config configFile) error {
	if majorVersion(config.SchemaVersion) != majorVersion(schemaVersion) || majorVersion(config.ContextSchemaVersion) != majorVersion(schemaVersion) {
		return fmt.Errorf("docx schema mismatch: run `docx migrate` before updating context")
	}
	return nil
}

func majorVersion(version string) string {
	before, _, ok := strings.Cut(version, ".")
	if !ok {
		return version
	}
	return before
}
