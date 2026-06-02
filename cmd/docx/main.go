package main

import (
	"fmt"
	"os"

	"github.com/cheng-zuguang/docx/internal/cli"
)

func main() {
	if err := cli.RunWithInput(os.Args[1:], ".", os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
