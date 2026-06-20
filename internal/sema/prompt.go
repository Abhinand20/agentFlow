package sema

import (
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/model"
)

func resolvePrompts(prog *model.Program, srcDir string, diags *diag.Diagnostics) {
	_ = diags
	for _, ref := range prog.Order {
		if ref.Kind != model.DeclAgent {
			continue
		}
		resolveAgentPrompt(prog.Agents[ref.Name], srcDir)
	}
}

func resolveAgentPrompt(ag *model.Agent, srcDir string) {
	if ag == nil {
		return
	}

	hasPrompt := ag.Prompt != ""
	hasPromptFile := ag.PromptPath != ""

	if hasPromptFile && hasPrompt {
		readPromptFile(ag, ag.PromptPath, srcDir)
		ag.Resolution.Reason = "conflict"
		return
	}
	if hasPromptFile {
		readPromptFile(ag, ag.PromptPath, srcDir)
		return
	}
	if hasPrompt && strings.HasSuffix(strings.ToLower(ag.Prompt), ".md") {
		readPromptFile(ag, ag.Prompt, srcDir)
		return
	}
	// inline text already in ag.Prompt from field interp
}

func readPromptFile(ag *model.Agent, path, srcDir string) {
	ag.PromptPath = path
	ag.Resolution.Path = path

	if filepath.IsAbs(path) || isWindowsDrive(path) {
		ag.Resolution.Reason = "absolute"
		return
	}

	clean := filepath.Clean(filepath.Join(srcDir, path))
	rel, err := filepath.Rel(srcDir, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		ag.Resolution.Reason = "escapes-tree"
		return
	}

	data, err := os.ReadFile(clean)
	if err != nil {
		if os.IsNotExist(err) {
			ag.Resolution.Reason = "missing"
		} else {
			ag.Resolution.Reason = "unreadable"
		}
		return
	}
	if !utf8.Valid(data) {
		ag.Resolution.Reason = "not-utf8"
		return
	}

	ag.Prompt = string(data)
	ag.PromptFromFile = true
	ag.Resolution.OK = true
}

func isWindowsDrive(path string) bool {
	if len(path) < 2 {
		return false
	}
	return ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) && path[1] == ':'
}
