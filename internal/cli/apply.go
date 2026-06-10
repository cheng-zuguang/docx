package cli

import (
	"fmt"
	"io"
)

func runApply(args []string, cwd string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printApplyHelp(stdout)
		return nil
	}
	switch args[0] {
	case "init":
		return runApplyInit(args[1:], cwd, stdin, stdout)
	case "proposals":
		return runApplyProposals(args[1:], cwd, stdin, stdout)
	default:
		return fmt.Errorf("docx apply: unknown subcommand %q", args[0])
	}
}
