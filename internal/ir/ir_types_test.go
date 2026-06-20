package ir_test

import (
	"reflect"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/ir"
)

func TestIRIsMapFree(t *testing.T) {
	var maps []string
	seen := make(map[reflect.Type]bool)
	walkTypes(reflect.TypeOf(ir.Program{}), "Program", seen, &maps)
	if len(maps) > 0 {
		t.Fatalf("IR must be map-free for deterministic JSON; found maps: %v", maps)
	}
}

func walkTypes(typ reflect.Type, path string, seen map[reflect.Type]bool, maps *[]string) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Map {
		*maps = append(*maps, path)
		return
	}
	if typ.Kind() != reflect.Struct {
		if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
			walkTypes(typ.Elem(), path+"[]", seen, maps)
		}
		return
	}
	if seen[typ] {
		return
	}
	seen[typ] = true
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}
		walkTypes(f.Type, path+"."+f.Name, seen, maps)
	}
}
