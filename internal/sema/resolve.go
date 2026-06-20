package sema

import (
	"strings"

	"github.com/Abhinand20/agentFlow/internal/ast"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

// Resolve lowers an AST into a typed semantic model.
func Resolve(root *ast.AST, srcDir string) (*model.Program, diag.Diagnostics) {
	_ = srcDir
	prog := &model.Program{
		Capabilities: make(map[string]*model.Capability),
		Types:        make(map[string]*model.EnumType),
		Agents:       make(map[string]*model.Agent),
		Gates:        make(map[string]*model.Gate),
		Flows:        make(map[string]*model.Flow),
	}
	var diags diag.Diagnostics

	// Pass 1: declare — build symbol tables and declaration order.
	for _, decl := range root.Decls {
		switch {
		case decl.Use != nil:
			name := qualNameStr(decl.Use.Name)
			prog.Order = append(prog.Order, model.DeclRef{Kind: model.DeclCapability, Name: name})
			prog.Capabilities[name] = &model.Capability{Name: name, Pos: decl.Use.Pos}
			if decl.Use.Alias != nil {
				emitLevelB(&diags, decl.Use.Pos, "use … as …")
			}
		case decl.Type != nil:
			prog.Order = append(prog.Order, model.DeclRef{Kind: model.DeclType, Name: decl.Type.Name})
			prog.Types[decl.Type.Name] = &model.EnumType{Name: decl.Type.Name, Pos: decl.Type.Pos}
		case decl.Agent != nil:
			prog.Order = append(prog.Order, model.DeclRef{Kind: model.DeclAgent, Name: decl.Agent.Name})
			prog.Agents[decl.Agent.Name] = &model.Agent{Name: decl.Agent.Name, Pos: decl.Agent.Pos}
		case decl.Gate != nil:
			prog.Order = append(prog.Order, model.DeclRef{Kind: model.DeclGate, Name: decl.Gate.Name})
			prog.Gates[decl.Gate.Name] = &model.Gate{Name: decl.Gate.Name, Pos: decl.Gate.Pos}
		case decl.Flow != nil:
			prog.Order = append(prog.Order, model.DeclRef{Kind: model.DeclFlow, Name: decl.Flow.Name})
			prog.Flows[decl.Flow.Name] = &model.Flow{Name: decl.Flow.Name, Entry: decl.Flow.Entry, Pos: decl.Flow.Pos}
		}
	}

	// Pass 2: interpret fields per kind (expanded in fields.go).
	for _, decl := range root.Decls {
		switch {
		case decl.Use != nil:
			name := qualNameStr(decl.Use.Name)
			interpUse(prog.Capabilities[name], decl.Use.Fields, &diags)
		case decl.Type != nil:
			interpType(prog.Types[decl.Type.Name], decl.Type, &diags)
		case decl.Agent != nil:
			interpAgent(prog.Agents[decl.Agent.Name], decl.Agent.Fields, &diags)
		case decl.Gate != nil:
			interpGate(prog.Gates[decl.Gate.Name], decl.Gate.Fields, &diags)
		case decl.Flow != nil:
			interpFlow(prog.Flows[decl.Flow.Name], decl.Flow, &diags)
		}
	}

	resolveModels(prog, &diags)
	resolvePrompts(prog, srcDir, &diags)
	accountEntryFlow(prog, &diags)

	return prog, diags
}

func qualNameStr(q *ast.QualName) string {
	if q == nil {
		return ""
	}
	return strings.Join(q.Parts, ".")
}
