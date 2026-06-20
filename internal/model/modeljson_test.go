package model_test

import (
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/model"
)

func TestSnapshotOmitsPositionAndSortsMaps(t *testing.T) {
	p := &model.Program{
		Agents: map[string]*model.Agent{
			"b": {Name: "b", ModelAlias: "sonnet"},
			"a": {Name: "a", ModelAlias: "opus"},
		},
		Order: []model.DeclRef{{Kind: model.DeclAgent, Name: "a"}},
	}
	out, err := model.MarshalSnapshot(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "Pos") {
		t.Fatalf("snapshot must omit Pos: %s", s)
	}
	if strings.Index(s, `"a"`) > strings.Index(s, `"b"`) {
		t.Fatalf("map keys must be sorted: %s", s)
	}
}
