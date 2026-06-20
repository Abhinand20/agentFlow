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
	caps := c.Capabilities()
	var diags diag.Diagnostics
	diags.Add(Negotiate(p, caps)...)

	for _, agent := range sortedAgents(p.Agents) {
		doc := render.AgentDocument(p, agent, v)
		path := fmt.Sprintf(".cursor/rules/%s.mdc", agent.Name)
		content, agentDiags := formatRuleMDC(agent, doc)
		diags.Add(agentDiags...)
		fs.Write(path, []byte(content))
	}

	cmdDoc := render.RunbookDocument(p, v)
	cmdName := commandBasename(p.Entry.Trigger)
	fs.Write(fmt.Sprintf(".cursor/commands/%s.md", cmdName), []byte(formatCommandMD(p, cmdDoc)))

	mcpJSON, mcpDiags := formatMCPJSON(p.Capabilities)
	diags.Add(mcpDiags...)
	if len(mcpJSON) > 0 {
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

func formatRuleMDC(agent ir.Agent, doc render.Document) (string, diag.Diagnostics) {
	desc := agent.Description
	tools := make([]string, 0)
	modelAlias := ""
	for _, f := range doc.Frontmatter {
		switch f.Key {
		case "description":
			if f.Value != "" {
				desc = f.Value
			}
		case "model-alias":
			modelAlias = f.Value
		case "tool":
			tools = append(tools, f.Value)
		}
	}
	if desc == "" {
		desc = fmt.Sprintf("AgentFlow agent %q instructions", agent.Name)
	}

	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("description: ")
	b.WriteString(strconvQuoteYAML(desc))
	b.WriteString("\nalwaysApply: false\n")
	b.WriteString("---\n")

	meta := agentflowMetaComment(modelAlias, tools)
	if meta != "" {
		b.WriteString("\n")
		b.WriteString(meta)
		b.WriteString("\n")
	}
	if doc.Body != "" {
		b.WriteString("\n")
		b.WriteString(doc.Body)
		if !strings.HasSuffix(doc.Body, "\n") {
			b.WriteByte('\n')
		}
	}

	return b.String(), agentMappingDiags(agent)
}

func agentflowMetaComment(modelAlias string, tools []string) string {
	var parts []string
	if modelAlias != "" {
		parts = append(parts, "model="+modelAlias)
	}
	if len(tools) > 0 {
		parts = append(parts, "tools="+strings.Join(tools, ","))
	}
	if len(parts) == 0 {
		return ""
	}
	return "<!-- agentflow: " + strings.Join(parts, " ") + " -->"
}

func formatCommandMD(p ir.Program, doc render.Document) string {
	var meta []string
	for _, f := range doc.Frontmatter {
		meta = append(meta, fmt.Sprintf("%s=%s", f.Key, f.Value))
	}
	if p.Entry.InType != "" {
		meta = append(meta, "in="+p.Entry.InType)
	}

	var b strings.Builder
	if len(meta) > 0 {
		b.WriteString("<!-- agentflow: ")
		b.WriteString(strings.Join(meta, " "))
		b.WriteString(" -->\n\n")
	}
	b.WriteString(doc.Body)
	if doc.Body != "" && !strings.HasSuffix(doc.Body, "\n") {
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

func formatMCPJSON(caps []ir.Capability) ([]byte, diag.Diagnostics) {
	servers := make(map[string]mcpServer)
	for _, c := range caps {
		if c.Kind != "mcp" {
			continue
		}
		transport := c.Transport
		if transport == "" {
			transport = "stdio"
		}
		servers[c.Name] = mcpServer{
			Command: c.Command,
			Args:    append([]string(nil), c.Args...),
			Type:    transport,
		}
	}
	if len(servers) == 0 {
		return nil, nil
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
		return nil, diag.Diagnostics{warn("AF308", fmt.Sprintf("failed to encode .cursor/mcp.json: %v", err))}
	}
	return append(data, '\n'), nil
}
