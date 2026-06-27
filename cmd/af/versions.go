package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/manifest"
)

func cmdVersions(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: af versions <list|diff|status> ...")
		return 2
	}
	switch args[0] {
	case "list":
		return cmdVersionsList(args[1:], stdout, stderr)
	case "diff":
		return cmdVersionsDiff(args[1:], stdout, stderr)
	case "status":
		return cmdVersionsStatus(args[1:], stdout, stderr)
	case "-h", "--help", "help":
		printVersionsUsage(stderr)
		return 0
	default:
		fmt.Fprintf(stderr, "versions: unknown subcommand %q\n\n", args[0])
		printVersionsUsage(stderr)
		return 2
	}
}

func printVersionsUsage(w io.Writer) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  af versions list [--target host] [--out dir] [<file>]")
	fmt.Fprintln(w, "  af versions diff <file> --target host [--out dir] [--from N] [--to M]")
	fmt.Fprintln(w, "  af versions status <file> --target host [--out dir]  (exit 1 when drift detected)")
}

func parseVersionsCommon(fs *flag.FlagSet, args []string) (target, out string, operands []string, code int) {
	targetFlag := fs.String("target", "", "host target ("+availableTargets()+")")
	outFlag := fs.String("out", ".", "output directory")
	operands, err := parseArgs(fs, args)
	if err != nil {
		return "", "", nil, 2
	}
	if *targetFlag == "" {
		fmt.Fprintf(fs.Output(), "versions: --target is required (%s)\n", availableTargets())
		return "", "", nil, 2
	}
	if _, ok := binding.Get(*targetFlag); !ok {
		fmt.Fprintf(fs.Output(), "versions: unknown target %q (%s)\n", *targetFlag, availableTargets())
		return "", "", nil, 2
	}
	return *targetFlag, *outFlag, operands, 0
}

func cmdVersionsList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("versions list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: af versions list --target <host> [--out dir] [<file>]")
		fs.PrintDefaults()
	}
	target, out, operands, code := parseVersionsCommon(fs, args)
	if code != 0 {
		return code
	}

	if len(operands) > 1 {
		fs.Usage()
		return 2
	}
	if len(operands) == 1 {
		return printSourceHistory(stdout, stderr, out, target, operands[0])
	}

	all, err := manifest.LoadAll(out, target, func(msg string) {
		fmt.Fprintln(stderr, msg)
	})
	if err != nil {
		fmt.Fprintf(stderr, "versions list: %v\n", err)
		return 1
	}
	if len(all) == 0 {
		fmt.Fprintf(stderr, "versions list: no manifests found under %s\n", manifest.ManifestsDir(target))
		return 1
	}
	for _, m := range all {
		fmt.Fprintf(stdout, "%s\tv%d\tbuilds=%d\tir=%s\n",
			m.Source.Path, m.Version, len(m.History), shortHash(m.IRHash))
	}
	return 0
}

func printSourceHistory(stdout, stderr io.Writer, out, target, sourcePath string) int {
	m, ok, err := manifest.Load(out, target, sourcePath)
	if err != nil {
		fmt.Fprintf(stderr, "versions list: %v\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintf(stderr, "versions list: no manifest found for %s\n", sourcePath)
		return 1
	}
	for i, rec := range m.History {
		changed := ""
		if i+1 < len(m.History) {
			changes := manifest.Diff(m.History[i+1], rec)
			changed = fmt.Sprintf(" +%d -%d ~%d", len(changes.Added), len(changes.Removed), len(changes.Changed))
		}
		fmt.Fprintf(stdout, "v%d\t%s\tir=%s\tartifacts=%d%s\n",
			rec.Version, rec.GeneratedAt, shortHash(rec.IRHash), len(rec.Artifacts), changed)
	}
	return 0
}

func cmdVersionsDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("versions diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	from := fs.Int("from", 0, "from version (default: previous)")
	to := fs.Int("to", 0, "to version (default: latest)")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: af versions diff <file> --target <host> [--out dir] [--from N] [--to M]")
		fs.PrintDefaults()
	}
	target, out, operands, code := parseVersionsCommon(fs, args)
	if code != 0 {
		return code
	}
	if len(operands) != 1 {
		fs.Usage()
		return 2
	}

	m, ok, err := manifest.Load(out, target, operands[0])
	if err != nil {
		fmt.Fprintf(stderr, "versions diff: %v\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintf(stderr, "versions diff: no manifest found for %s\n", operands[0])
		return 1
	}

	toVer := *to
	if toVer == 0 {
		toVer = m.Version
	}
	fromVer := *from
	if fromVer == 0 {
		fromVer = toVer - 1
	}
	if fromVer <= 0 || toVer <= 0 {
		fmt.Fprintf(stderr, "versions diff: need two valid versions (from=%d to=%d)\n", fromVer, toVer)
		return 1
	}
	if fromVer > toVer {
		fmt.Fprintf(stderr, "versions diff: --from must be <= --to (from=%d to=%d)\n", fromVer, toVer)
		return 2
	}

	a := m.FindRecord(fromVer)
	b := m.FindRecord(toVer)
	if a == nil || b == nil {
		fmt.Fprintf(stderr, "versions diff: version not found (from=%d to=%d)\n", fromVer, toVer)
		return 1
	}

	changes := manifest.Diff(*a, *b)
	fmt.Fprintf(stdout, "diff v%d -> v%d for %s\n", fromVer, toVer, m.Source.Path)
	for _, p := range changes.Added {
		fmt.Fprintf(stdout, "+ %s\n", p)
	}
	for _, p := range changes.Removed {
		fmt.Fprintf(stdout, "- %s\n", p)
	}
	for _, p := range changes.Changed {
		fmt.Fprintf(stdout, "~ %s\n", p)
	}
	if len(changes.Added)+len(changes.Removed)+len(changes.Changed) == 0 {
		fmt.Fprintln(stdout, "(no artifact changes)")
	}
	return 0
}

func cmdVersionsStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("versions status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: af versions status <file> --target <host> [--out dir]")
		fs.PrintDefaults()
	}
	target, out, operands, code := parseVersionsCommon(fs, args)
	if code != 0 {
		return code
	}
	if len(operands) != 1 {
		fs.Usage()
		return 2
	}

	m, ok, err := manifest.Load(out, target, operands[0])
	if err != nil {
		fmt.Fprintf(stderr, "versions status: %v\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintf(stderr, "versions status: no manifest found for %s\n", operands[0])
		return 1
	}

	report := manifest.DriftCheck(m, out)
	fmt.Fprintf(stdout, "status for %s (v%d)\n", m.Source.Path, m.Version)
	printArtifactGroup(stdout, "clean", report.Clean)
	printArtifactGroup(stdout, "modified", report.Modified)
	printArtifactGroup(stdout, "missing", report.Missing)
	printArtifactGroup(stdout, "unreadable", report.Unreadable)
	if len(report.Modified)+len(report.Missing)+len(report.Unreadable) > 0 {
		return 1
	}
	return 0
}

func printArtifactGroup(w io.Writer, label string, arts []manifest.Artifact) {
	fmt.Fprintf(w, "%s (%d):\n", label, len(arts))
	for _, a := range arts {
		fmt.Fprintf(w, "  %s\n", a.Path)
	}
}

func shortHash(hex string) string {
	if len(hex) <= 8 {
		return hex
	}
	return hex[:8]
}
