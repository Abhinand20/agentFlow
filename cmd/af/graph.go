package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/dot"
)

func cmdGraph(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetNames := strings.Join(binding.Names(), "|")
	if targetNames == "" {
		targetNames = "<host>"
	}
	target := fs.String("target", "", "host target (optional; "+availableTargets()+")")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: af graph <file> [--target %s]\n", targetNames)
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
	if *target != "" {
		if _, ok := binding.Get(*target); !ok {
			fmt.Fprintf(stderr, "graph: unknown target %q (%s)\n", *target, availableTargets())
			return 2
		}
	}

	res, hasErrors := compileOrReport(operands[0], stderr)
	if hasErrors {
		return 1
	}
	fmt.Fprint(stdout, dot.Emit(res.IR))
	return 0
}
