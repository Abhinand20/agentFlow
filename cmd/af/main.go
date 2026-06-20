package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}

	switch args[0] {
	case "validate":
		return cmdNotImplemented("validate")
	case "build":
		return cmdNotImplemented("build")
	case "graph":
		return cmdNotImplemented("graph")
	case "-h", "--help", "help":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usageText())
	}
}

func cmdNotImplemented(name string) error {
	fmt.Printf("%s: not implemented\n", name)
	return nil
}

func printUsage(w *os.File) {
	fmt.Fprint(w, usageText())
}

func usageText() string {
	return `af — AgentFlow compiler

Usage:
  af <command> [arguments]

Commands:
  validate   Check an .af file for errors
  build      Compile an .af file to host configuration
  graph      Print the resolved flow graph as DOT

Run "af <command> -h" for command-specific flags (not yet implemented).
`
}
