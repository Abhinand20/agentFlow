package model

import "encoding/json"

// MarshalSnapshot returns deterministic, position-free JSON for goldens.
// Go's encoding/json sorts map keys; Pos fields carry json:"-".
func MarshalSnapshot(p *Program) ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}
