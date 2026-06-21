package main

import (
	"flag"
	"fmt"
	"io"
)

func cmdValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: af validate <file>")
		fs.PrintDefaults()
	}
	operands, err := parseArgs(fs, args)
	if err != nil {
		return 2
	}
	if len(operands) != 1 {
		fs.Usage()
		return 2
	}

	_, hasErrors := compileOrReport(operands[0], stderr)
	if hasErrors {
		return 1
	}
	fmt.Fprintln(stdout, "ok")
	return 0
}
