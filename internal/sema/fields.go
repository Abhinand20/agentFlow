package sema

import (
	"github.com/Abhinand20/agentFlow/internal/ast"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func interpUse(c *model.Capability, fields []*ast.Field, diags *diag.Diagnostics) {
	_ = c
	_ = fields
	_ = diags
}

func interpType(et *model.EnumType, td *ast.TypeDecl, diags *diag.Diagnostics) {
	_ = et
	_ = td
	_ = diags
}

func interpAgent(ag *model.Agent, fields []*ast.Field, diags *diag.Diagnostics) {
	for _, f := range fields {
		if f.Key == "model" {
			ag.ModelAlias = refOrStr(f.Value)
		}
	}
}

func interpGate(g *model.Gate, fields []*ast.Field, diags *diag.Diagnostics) {
	_ = g
	_ = fields
	_ = diags
}

func interpFlow(fl *model.Flow, fd *ast.Flow, diags *diag.Diagnostics) {
	_ = fl
	_ = fd
	_ = diags
}

func resolveModels(prog *model.Program, diags *diag.Diagnostics) {
	_ = prog
	_ = diags
}

func resolvePrompts(prog *model.Program, srcDir string, diags *diag.Diagnostics) {
	_ = prog
	_ = srcDir
	_ = diags
}

func accountEntryFlow(prog *model.Program, diags *diag.Diagnostics) {
	_ = prog
	_ = diags
}

func refOrStr(v *ast.Value) string {
	if v == nil {
		return ""
	}
	if v.Ref != nil {
		return qualNameStr(v.Ref)
	}
	if v.Str != nil {
		return *v.Str
	}
	return ""
}
