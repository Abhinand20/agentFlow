// Package dot renders an IR flow as a Graphviz DOT digraph for `af graph`.
// Output is target-neutral and deterministic: nodes are emitted in
// control-label discovery order and edges in the order they are discovered
// during a structural walk of the flow tree.
package dot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/ir"
)

// Emit returns the DOT representation of the program's entry flow.
func Emit(p ir.Program) string {
	b := &builder{
		nodes:    map[string]string{},
		edgeSeen: map[string]bool{},
	}
	b.walk(p.Flow.Root)

	var sb strings.Builder
	fmt.Fprintf(&sb, "digraph %s {\n", strconv.Quote(p.Flow.Name))
	sb.WriteString("  rankdir=TB;\n")
	for _, id := range b.nodeOrder {
		fmt.Fprintf(&sb, "  %s [%s];\n", strconv.Quote(id), b.nodes[id])
	}
	for _, e := range b.edges {
		if e.label != "" {
			fmt.Fprintf(&sb, "  %s -> %s [label=%s];\n",
				strconv.Quote(e.from), strconv.Quote(e.to), strconv.Quote(e.label))
		} else {
			fmt.Fprintf(&sb, "  %s -> %s;\n", strconv.Quote(e.from), strconv.Quote(e.to))
		}
	}
	sb.WriteString("}\n")
	return sb.String()
}

type edge struct {
	from  string
	to    string
	label string
}

type builder struct {
	nodeOrder []string
	nodes     map[string]string // id -> DOT attribute body
	edges     []edge
	edgeSeen  map[string]bool
}

func (b *builder) addNode(id, label, shape string) {
	if _, ok := b.nodes[id]; ok {
		return
	}
	attr := "label=" + strconv.Quote(label)
	if shape != "" {
		attr += ", shape=" + shape
	}
	b.nodes[id] = attr
	b.nodeOrder = append(b.nodeOrder, id)
}

func (b *builder) addEdge(from, to, label string) {
	key := from + "\x00" + to + "\x00" + label
	if b.edgeSeen[key] {
		return
	}
	b.edgeSeen[key] = true
	b.edges = append(b.edges, edge{from: from, to: to, label: label})
}

func (b *builder) connect(froms, tos []string, label string) {
	for _, f := range froms {
		for _, t := range tos {
			b.addEdge(f, t, label)
		}
	}
}

// walk registers nodes/edges for n and returns the entry and exit control
// labels of the subgraph so the caller can wire it into surrounding flow.
func (b *builder) walk(n ir.Node) (entries, exits []string) {
	switch n.Kind {
	case ir.NodeStep:
		id := n.Step.ControlLabel
		b.addNode(id, id, shapeFor(n.Step.Kind))
		return []string{id}, []string{id}

	case ir.NodeSeq:
		var first, prevExits []string
		for i := range n.Steps {
			en, ex := b.walk(n.Steps[i])
			if len(en) == 0 {
				continue
			}
			if first == nil {
				first = en
			}
			if prevExits != nil {
				b.connect(prevExits, en, "")
			}
			prevExits = ex
		}
		return first, prevExits

	case ir.NodeParallel:
		gatherID := n.Gather.ControlLabel
		b.addNode(gatherID, gatherID, shapeFor(n.Gather.Kind))
		var entries []string
		for i := range n.Branches {
			en, ex := b.walk(n.Branches[i])
			entries = append(entries, en...)
			b.connect(ex, []string{gatherID}, "gather")
		}
		return entries, []string{gatherID}

	case ir.NodeLoop:
		if n.Body == nil {
			return nil, nil
		}
		en, ex := b.walk(*n.Body)
		b.connect(ex, en, loopLabel(n))
		return en, ex

	case ir.NodeBranch:
		// Internal node IDs are DOT-quoted at emit time; BranchValue labels are
		// spec identifiers and safe as opaque ID suffixes.
		decID := branchNodeID(n.BranchValue)
		b.addNode(decID, n.BranchValue+" ?", "diamond")
		var exits []string
		for _, c := range n.Cases {
			en, ex := b.walk(c.Body)
			b.connect([]string{decID}, en, strings.Join(c.Values, ", "))
			exits = append(exits, ex...)
		}
		return []string{decID}, exits

	default:
		id := fmt.Sprintf("unknown:%s", n.Kind)
		b.addNode(id, string(n.Kind)+" (unsupported)", "octagon")
		return []string{id}, []string{id}
	}
}

func branchNodeID(valueLabel string) string {
	return "branch:" + valueLabel
}

func loopLabel(n ir.Node) string {
	if n.Cond != nil {
		return fmt.Sprintf("%s %s %s", n.Cond.ValueLabel, n.Cond.Op, n.Cond.Enum)
	}
	return "loop"
}

func shapeFor(kind string) string {
	switch kind {
	case "gate":
		return "box"
	case "agent":
		return "ellipse"
	default:
		return ""
	}
}
