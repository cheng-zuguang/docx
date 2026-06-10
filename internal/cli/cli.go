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
	if args[0] == "help" && len(args) > 1 {
		return printCommandHelp(args[1], stdout)
	}
	if len(args) > 1 && isHelpArg(args[1]) {
		return printCommandHelp(args[0], stdout)
	}

	switch args[0] {
	case "init":
		return runInit(args[1:], cwd, stdin, stdout)
	case "scan":
		return runScan(args[1:], cwd, stdout)
	case "update":
		return runUpdate(args[1:], cwd, stdout)
	case "sync":
		return runSync(args[1:], cwd, stdout)
	case "finish":
		return runFinish(args[1:], cwd, stdout)
	case "index":
		return runIndex(args[1:], cwd, stdout)
	case "doctor":
		return runDoctor(args[1:], cwd, stdout)
	case "proposals":
		return runProposals(args[1:], cwd, stdout)
	case "apply":
		return runApply(args[1:], cwd, stdin, stdout)
	case "migrate":
		return runMigrate(args[1:], cwd, stdout)
	case "install-hook":
		return runInstallHook(args[1:], cwd, stdout)
	case "install-agent-hook":
		return runInstallAgentHook(args[1:], cwd, stdout)
	case "help", "-h", "--help":
		printHelp(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `docx manages AI-readable project context.

Usage:
  docx <command> [options]
  docx <command> --help

Commands:
  init          Initialize .docx.json, .doc/ context, entry files, and optional AI summaries.
  scan          Inspect detected manifests, languages, entrypoints, tests, and module candidates.
  sync          Record changes and create an active-agent context sync task.
  finish        Run the active-agent end-of-turn context sync.
  update        Record code changes and refresh affected module context.
  proposals     List, show, accept, or reject semantic update proposals.
  apply         Apply active-agent output files or stdin JSON to context.
  index         Rebuild or check generated context indexes.
  doctor        Check configuration, schema, indexes, analyzers, and hooks.
  migrate       Migrate .docx.json/context metadata to the current schema.
  install-hook  Install managed git hooks for automatic context updates.
  install-agent-hook
                Install Codex or Claude Code lifecycle hooks.

Options:
  -h, --help    Show this help.

Run 'docx <command> --help' for command-specific options.
`)
}
