package cli

import (
	"fmt"
	"io"
	"strings"
)

const (
	schemaVersion = "1.0"
	defaultDocDir = ".doc"
)

// Run executes the docx command against a working directory.
func Run(args []string, cwd string, stdout io.Writer, stderr io.Writer) error {
	return RunWithInput(args, cwd, strings.NewReader(""), stdout, stderr)
}

// RunWithInput executes the docx command against a working directory with stdin.
func RunWithInput(args []string, cwd string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		printHelp(stdout)
		return nil
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], cwd, stdin, stdout)
	case "scan":
		return runScan(args[1:], cwd, stdout)
	case "update":
		return runUpdate(args[1:], cwd, stdout)
	case "index":
		return runIndex(args[1:], cwd, stdout)
	case "doctor":
		return runDoctor(args[1:], cwd, stdout)
	case "proposals":
		return runProposals(args[1:], cwd, stdout)
	case "migrate":
		return runMigrate(args[1:], cwd, stdout)
	case "install-hook":
		return runInstallHook(args[1:], cwd, stdout)
	case "help", "-h", "--help":
		printHelp(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "docx commands: init, scan, update, index, doctor, proposals, migrate, install-hook")
}
