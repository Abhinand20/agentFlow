package cursor_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Abhinand20/agentFlow/internal/binding/cursor"
)

var update = flag.Bool("update", false, "rewrite golden FS snapshots")

func TestGoldenFSReview(t *testing.T) {
	p := loadReviewIR(t)
	fs, _ := cursor.Binding().Emit(p)

	for _, path := range fs.Paths() {
		content, _ := fs.Get(path)
		goldenPath := filepath.Join("testdata", filepath.FromSlash(path))
		if *update {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(goldenPath, content, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("read golden %s (run -update): %v", goldenPath, err)
		}
		if string(content) != string(want) {
			t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", path, content, want)
		}
	}
}
