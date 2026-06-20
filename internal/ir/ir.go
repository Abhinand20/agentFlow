package ir

// Program is the binding-agnostic intermediate representation consumed by
// render (M6) and bindings (M7/M10). It carries no source positions.
type Program struct {
	Entry        Entry
	Agents       []Agent
	Gates        []Gate
	Capabilities []Capability
	Flow         Flow
}

type Entry struct {
	Trigger  string // on: value, e.g. "/ship"
	FlowName string
	InType   string
	OutType  string
}

type Agent struct {
	Name        string
	Provider    string
	Alias       string
	HostModelID string // empty in IR; bindings fill this
	In          string
	Out         string
	OutEnum     []string
	Tools       []ToolRef
	Prompt      string
	Permissions string
	Retry       int
	Description string
}

type ToolRef struct {
	Capability string
	Tool       string
}

type Gate struct {
	Name         string
	Run          string
	OnFail       string // halt | retry | goto | enter-loop
	OnFailTarget string
	Behavior     string
	ScriptRetry  int
}

type Capability struct {
	Name      string
	Kind      string // model-provider | mcp
	Models    []string
	Tools     []string
	Transport string
	Command   string
	Args      []string
}

type Flow struct {
	Name   string
	Return ReturnBinding
	Root   Node
	Order  []string // pre-order control-label walk
}

type ReturnBinding struct {
	ValueLabel     string
	OutType        string
	Defaulted      bool
	BranchTerminal bool
}

type NodeKind string

const (
	NodeSeq      NodeKind = "seq"
	NodeStep     NodeKind = "step"
	NodeBranch   NodeKind = "branch"
	NodeLoop     NodeKind = "loop"
	NodeParallel NodeKind = "parallel"
)

type Node struct {
	Kind NodeKind

	// seq
	Steps []Node `json:",omitempty"`

	// step (leaf)
	Step *StepRef `json:",omitempty"`

	// branch
	BranchValue string       `json:",omitempty"`
	BranchEnum  []string     `json:",omitempty"`
	Cases       []BranchCase `json:",omitempty"`

	// loop / repeat
	DoWhile bool  `json:",omitempty"`
	HasMax  bool
	Max     int   `json:",omitempty"`
	Cond    *Cond `json:",omitempty"`
	Body    *Node `json:",omitempty"`

	// parallel
	Branches []Node         `json:",omitempty"`
	Gather   *StepRef       `json:",omitempty"`
	Payload  *GatherPayload `json:",omitempty"`
}

type StepRef struct {
	ControlLabel string
	ValueLabel   string
	Decl         string
	Kind         string // agent | gate | terminal
	OutType      string
	OutEnum      []string
	PrevProducer string
}

type BranchCase struct {
	Values []string
	Body   Node
}

type Cond struct {
	ValueLabel string
	Op         string
	Enum       string
}

type GatherPayload struct {
	GatherControlLabel string
	Branches           []GatherBranch
}

type GatherBranch struct {
	ControlLabel string
	ValueLabel   string
}
