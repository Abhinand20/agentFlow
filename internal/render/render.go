package render

import (
	"fmt"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/ir"
)

// Document holds neutral frontmatter values and a rendered markdown body.
type Document struct {
	Frontmatter []FMField
	Body        string
}

// FMField is one ordered frontmatter key/value pair (binding maps to host format).
type FMField struct {
	Key   string
	Value string
}

// AgentDocument renders one subagent's instruction file body and neutral frontmatter values.
func AgentDocument(p ir.Program, a ir.Agent, v Vocabulary) Document {
	fm := []FMField{
		{Key: "name", Value: a.Name},
	}
	if a.Description != "" {
		fm = append(fm, FMField{Key: "description", Value: a.Description})
	}
	if a.Alias != "" {
		fm = append(fm, FMField{Key: "model-alias", Value: a.Alias})
	}
	for _, t := range a.Tools {
		fm = append(fm, FMField{Key: "tool", Value: fmt.Sprintf("%s:%s", t.Capability, t.Tool)})
	}
	return Document{
		Frontmatter: fm,
		Body:        AgentPrompt(a, v),
	}
}

// RunbookDocument renders the entry flow as a numbered runbook (command file body).
func RunbookDocument(p ir.Program, v Vocabulary) Document {
	trigger := p.Entry.Trigger
	if trigger == "" {
		trigger = p.Flow.Name
	}
	fm := []FMField{
		{Key: "trigger", Value: trigger},
		{Key: "flow", Value: p.Flow.Name},
	}
	return Document{
		Frontmatter: fm,
		Body:        Runbook(p, v),
	}
}

// FormatDocument renders frontmatter and body for golden comparison.
func FormatDocument(d Document) string {
	var b strings.Builder
	for _, f := range d.Frontmatter {
		b.WriteString(f.Key)
		b.WriteString(": ")
		b.WriteString(f.Value)
		b.WriteString("\n")
	}
	if d.Body != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(d.Body)
	}
	return b.String()
}
