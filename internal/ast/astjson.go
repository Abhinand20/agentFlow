package ast

import "encoding/json"

// MarshalSnapshot returns deterministic, position-free JSON of the AST for goldens.
func MarshalSnapshot(root *AST) ([]byte, error) {
	return json.MarshalIndent(root, "", "  ")
}
