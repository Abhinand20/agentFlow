package model

import (
	"github.com/Abhinand20/agentFlow/internal/ast"
	"github.com/alecthomas/participle/v2/lexer"
)

type DeclKind int

const (
	DeclCapability DeclKind = iota
	DeclType
	DeclAgent
	DeclGate
	DeclFlow
)

type StepKind int

const (
	StepRef StepKind = iota
	StepChain
	StepBranch
	StepLoop
	StepRepeat
	StepParallel
	StepCall
)

type Program struct {
	Capabilities map[string]*Capability
	Types        map[string]*EnumType
	Agents       map[string]*Agent
	Gates        map[string]*Gate
	Flows        map[string]*Flow
	EntryFlow    string
	Order        []DeclRef
}

type DeclRef struct {
	Kind DeclKind
	Name string
}

type Capability struct {
	Name      string
	Kind      string
	Models    []string
	Tools     []string
	Transport string
	Command   string
	Args      []string
	Raw       map[string]*ast.Value
	Pos       lexer.Position `json:"-"`
}

type EnumType struct {
	Name   string
	Values []string
	Pos    lexer.Position `json:"-"`
}

type ToolRef struct {
	Capability string
	Tool       string
	Raw        string
	Pos        lexer.Position `json:"-"`
}

type PromptResolution struct {
	Path   string
	OK     bool
	Reason string
}

type Agent struct {
	Name           string
	ModelAlias     string
	ModelProvider  string
	ResolvedAlias  string
	In             string
	Out            string
	Permissions    string
	Prompt         string
	PromptPath     string
	PromptFromFile bool
	Tools          []ToolRef
	Retry          int
	Description    string
	Resolution     PromptResolution
	Raw            map[string]*ast.Value
	Pos            lexer.Position `json:"-"`
}

type GateFailAction int

const (
	FailHalt GateFailAction = iota
	FailRetryStep
	FailGotoStep
	FailEnterLoop
)

type Gate struct {
	Name         string
	Run          string
	OnFail       GateFailAction
	OnFailTarget string
	Behavior     string
	ScriptRetry  int
	Raw          map[string]*ast.Value
	Pos          lexer.Position `json:"-"`
}

type Flow struct {
	Name           string
	Entry          bool
	On             string
	In             string
	Out            string
	Return         string
	OutExplicit    bool
	ReturnExplicit bool
	Params         []Param
	Body           []Step
	Raw            map[string]*ast.Value `json:"Raw,omitempty"`
	Pos            lexer.Position `json:"-"`
}

type Param struct {
	Name string
	Type string
	Pos  lexer.Position `json:"-"`
}

type Step struct {
	Kind  StepKind
	Pos   lexer.Position `json:"-"`
	Ref   string
	Alias string
	Chain []Step
	BranchValue string
	BranchIsIt  bool
	Cases       []Case
	Cond     *Cond
	Max      *int
	HasMax   bool
	Body     []Step
	DoWhile  bool
	PFork   []Step
	Gather  *Step
	Each    *Each
	CallArgs  []*ast.Value
	CallBlock []Step
}

type Case struct {
	Values []string
	Step   Step
	Pos    lexer.Position `json:"-"`
}

type Cond struct {
	Value  string
	IsIt   bool
	Op     string
	Enum   string
	Pos    lexer.Position `json:"-"`
}

type Each struct {
	Item string
	As   string
	Pos  lexer.Position `json:"-"`
}

// Terminals are reserved leaf names usable in any flow (kept out of decl maps).
func Terminals() map[string]bool {
	return map[string]bool{"done": true, "fail": true}
}
