package cursor

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Abhinand20/agentFlow/internal/binding"
	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/emit"
	"github.com/Abhinand20/agentFlow/internal/ir"
	"github.com/Abhinand20/agentFlow/internal/render"
)

type cursorBinding struct{}

// Binding returns the registered Cursor binding instance.
func Binding() binding.Binding {
	return cursorBinding{}
}

func init() {
	binding.Register(cursorBinding{})
}

func (cursorBinding) Name() string { return "cursor" }

func (c cursorBinding) Emit(p ir.Program) (*emit.FS, diag.Diagnostics) {
	fs := emit.NewFS()
	v := Vocabulary()
	diags := Negotiate(p)

	for _, agent := range sortedAgents(p.Agents) {
		doc := render.AgentDocument(p, agent, v)
		path := fmt.Sprintf(".cursor/rules/%s.mdc", agent.Name)
		fs.Write(path, []byte(formatRuleMDC(agent, doc.Body)))
	}

	cmdDoc := render.RunbookDocument(p, v)
	cmdName := commandBasename(p.Entry.Trigger)
	fs.Write(fmt.Sprintf(".cursor/commands/%s.md", cmdName), []byte(cmdDoc.Body))

	if mcpJSON := formatMCPJSON(p.Capabilities); len(mcpJSON) > 0 {
		fs.Write(".cursor/mcp.json", mcpJSON)
	}

	return fs, diags
}

func sortedAgents(agents []ir.Agent) []ir.Agent {
	out := make([]ir.Agent, len(agents))
	copy(out, agents)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func commandBasename(trigger string) string {
	trigger = strings.TrimPrefix(trigger, "/")
	if trigger == "" {
		return "command"
	}
	return trigger
}

func formatRuleMDC(agent ir.Agent, body string) string {
	desc := agent.Description
	if desc == "" {
		desc = fmt.Sprintf("AgentFlow agent %q instructions", agent.Name)
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("description: ")
	b.WriteString(strconvQuoteYAML(desc))
	b.WriteString("\nalwaysApply: false\n")
	b.WriteString("---\n\n")
	b.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		b.WriteByte('\n')
	}
	return b.String()
}

func strconvQuoteYAML(s string) string {
	if strings.ContainsAny(s, ":\n\"'#") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

type mcpConfig struct {
	MCPServers map[string]mcpServer `json:"mcpServers"`
}

type mcpServer struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Type    string   `json:"type,omitempty"`
}

func formatMCPJSON(caps []ir.Capability) []byte {
	servers := make(map[string]mcpServer)
	for _, c := range caps {
		if c.Kind != "mcp" {
			continue
		}
		srv := mcpServer{
			Command: c.Command,
			Args:    append([]string(nil), c.Args...),
		}
		if c.Transport != "" {
			srv.Type = c.Transport
		}
		servers[c.Name] = srv
	}
	if len(servers) == 0 {
		return nil
	}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	ordered := make(map[string]mcpServer, len(servers))
	for _, name := range names {
		ordered[name] = servers[name]
	}
	data, err := json.MarshalIndent(mcpConfig{MCPServers: ordered}, "", "  ")
	if err != nil {
		return nil
	}
	return append(data, '\n')
}
