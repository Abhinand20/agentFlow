package ast_test

import (
	"strings"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/ast"
)

func TestMarshalSnapshotOmitsPosition(t *testing.T) {
	t.Parallel()

	root := &ast.AST{
		Decls: []*ast.Decl{{
			Agent: &ast.Agent{
				Name: "x",
				Fields: []*ast.Field{{
					Key: "enabled",
					Value: &ast.Value{Bool: boolPtr(true)},
				}},
			},
		}},
	}

	data, err := ast.MarshalSnapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if strings.Contains(out, "Pos") {
		t.Fatalf("snapshot should omit Pos fields: %s", out)
	}
	if !strings.Contains(out, `"Name": "x"`) {
		t.Fatalf("expected agent name in snapshot: %s", out)
	}
}

func boolPtr(b bool) *bool { return &b }
