package sema

import (
	"sort"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func resolveModels(prog *model.Program, diags *diag.Diagnostics) {
	for _, ref := range prog.Order {
		if ref.Kind != model.DeclAgent {
			continue
		}
		ag := prog.Agents[ref.Name]
		if ag == nil || ag.ModelAlias == "" {
			continue
		}
		resolveAgentModel(ag, prog, diags)
	}
}

func resolveAgentModel(ag *model.Agent, prog *model.Program, diags *diag.Diagnostics) {
	alias := ag.ModelAlias
	if parts := strings.SplitN(alias, ".", 2); len(parts) == 2 {
		providerName, modelName := parts[0], parts[1]
		cap := prog.Capabilities[providerName]
		if cap == nil || cap.Kind != "model-provider" {
			diags.Add(diag.Diagnostic{
				Code:     "AF110",
				Severity: diag.Error,
				Msg:      "no provider " + providerName,
				Pos:      ag.Pos,
			})
			return
		}
		if !containsStr(cap.Models, modelName) {
			diags.Add(diag.Diagnostic{
				Code:     "AF110",
				Severity: diag.Error,
				Msg:      "provider " + providerName + " has no model alias " + modelName,
				Pos:      ag.Pos,
			})
			return
		}
		ag.ModelProvider = providerName
		ag.ResolvedAlias = modelName
		return
	}

	var candidates []string
	for name, cap := range prog.Capabilities {
		if cap.Kind == "model-provider" && containsStr(cap.Models, alias) {
			candidates = append(candidates, name)
		}
	}
	sort.Strings(candidates)
	switch len(candidates) {
	case 0:
		diags.Add(diag.Diagnostic{
			Code:     "AF110",
			Severity: diag.Error,
			Msg:      "unknown model alias '" + alias + "' (no provider lists it)",
			Pos:      ag.Pos,
		})
	case 1:
		ag.ModelProvider = candidates[0]
		ag.ResolvedAlias = alias
	default:
		diags.Add(diag.Diagnostic{
			Code:     "AF110",
			Severity: diag.Error,
			Msg:      "ambiguous model alias '" + alias + "'; candidates: " + strings.Join(candidates, ", ") + " (qualify as provider.alias)",
			Pos:      ag.Pos,
		})
	}
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
