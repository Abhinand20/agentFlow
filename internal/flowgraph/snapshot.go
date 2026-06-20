package flowgraph

import "encoding/json"

// MarshalSnapshot returns deterministic, position-free JSON for goldens.
func MarshalSnapshot(r *Resolved) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
