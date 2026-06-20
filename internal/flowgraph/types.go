package flowgraph

import "github.com/Abhinand20/agentFlow/internal/model"

type NodeKind int

const (
	KindSeq NodeKind = iota
	KindStep
	KindBranch
	KindLoop
	KindParallel
)

type Resolved struct {
	Entry     string
	Tree      *Node
	Instances map[string]*StepInstance
}

type Node struct {
	Kind          NodeKind
	Label         string // KindStep -> Instances[Label]
	Steps         []*Node
	BranchValue   string
	Cases         []*ResolvedCase
	Cond          *model.Cond
	Max           *int
	DoWhile       bool
	Gather        *Node
	GatherPayload map[string]string
}

type ResolvedCase struct {
	Values []string
	Step   *Node
}

type StepInstance struct {
	ControlLabel  string
	ValueLabel    string
	Decl          string
	Kind          model.StepKind
	OutEnum       []string
	Upstream      string
	ReturnsFrom   string
	GatherPayload map[string]string
}
