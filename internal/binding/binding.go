package binding

import (
	"sort"

	"github.com/Abhinand20/agentFlow/internal/diag"
	"github.com/Abhinand20/agentFlow/internal/emit"
	"github.com/Abhinand20/agentFlow/internal/ir"
)

// Capability names a host feature for negotiation (fully defined in M10).
type Capability string

// Binding emits host-native artifacts from IR.
type Binding interface {
	Name() string
	Capabilities() map[Capability]bool
	Emit(p ir.Program) (*emit.FS, diag.Diagnostics)
}

var registry = map[string]Binding{}

// Register adds a binding implementation.
func Register(b Binding) {
	registry[b.Name()] = b
}

// Get returns a registered binding by name.
func Get(name string) (Binding, bool) {
	b, ok := registry[name]
	return b, ok
}

// Names returns registered binding names in sorted order.
func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
