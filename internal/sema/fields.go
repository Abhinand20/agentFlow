package sema

import (
	"github.com/Abhinand20/agentFlow/internal/ast"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func interpUse(c *model.Capability, fields []*ast.Field, diags *diag.Diagnostics) {
	if c.Raw == nil {
		c.Raw = make(map[string]*ast.Value)
	}
	for _, f := range fields {
		switch f.Key {
		case "kind":
			c.Kind = scalarVal(f.Value)
		case "models":
			c.Models = identListVal(f.Value)
		case "tools":
			c.Tools = identListVal(f.Value)
		case "transport":
			c.Transport = strVal(f.Value)
		case "command":
			c.Command = strVal(f.Value)
		case "args":
			c.Args = strListVal(f.Value)
		default:
			diags.Add(diag.Diagnostic{
				Code:     "AF120",
				Severity: diag.Warning,
				Msg:      "unknown field " + f.Key,
				Pos:      f.Pos,
			})
			c.Raw[f.Key] = f.Value
		}
	}
	if c.Kind == "mcp" && c.Command == "" {
		diags.Add(diag.Diagnostic{
			Code:     "AF130",
			Severity: diag.Error,
			Msg:      "mcp capability requires command",
			Pos:      c.Pos,
		})
	}
	if c.Kind == "model-provider" && len(c.Models) == 0 {
		diags.Add(diag.Diagnostic{
			Code:     "AF131",
			Severity: diag.Error,
			Msg:      "model-provider requires models list",
			Pos:      c.Pos,
		})
	}
}

func strVal(v *ast.Value) string {
	if v == nil || v.Str == nil {
		return ""
	}
	return *v.Str
}

func identListVal(v *ast.Value) []string {
	if v == nil || v.List == nil {
		return nil
	}
	var out []string
	for _, item := range v.List.Items {
		if item.Ref != nil {
			out = append(out, qualNameStr(item.Ref))
		} else if item.Str != nil {
			out = append(out, *item.Str)
		}
	}
	return out
}

func strListVal(v *ast.Value) []string {
	if v == nil || v.List == nil {
		return nil
	}
	var out []string
	for _, item := range v.List.Items {
		if item.Str != nil {
			out = append(out, *item.Str)
		}
	}
	return out
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

func scalarVal(v *ast.Value) string {
	return refOrStr(v)
}
