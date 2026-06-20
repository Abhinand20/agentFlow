package ir

import "encoding/json"

// Marshal serializes the IR to deterministic indented JSON.
// The IR is map-free and collections are pre-sorted, so stdlib encoding is stable.
func Marshal(p Program) ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// UnmarshalJSON loads an IR program from JSON (round-trip testing).
func UnmarshalJSON(data []byte) (Program, error) {
	var p Program
	err := json.Unmarshal(data, &p)
	return p, err
}
