package pair

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLoadsPairs(t *testing.T) {
	root := filepath.Join("..", "..", "testdata")
	before := filepath.Join(root, "before")
	after := filepath.Join(root, "after")

	pairs, err := Resolve(before, after, "*.cfg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0].Name != "fw1.cfg" {
		t.Fatalf("unexpected pair name: %s", pairs[0].Name)
	}
}

func TestResolveDefaultGlobMatchesLogAndTxt(t *testing.T) {
	root := t.TempDir()
	before := filepath.Join(root, "before")
	after := filepath.Join(root, "after")
	if err := os.MkdirAll(before, 0o755); err != nil {
		t.Fatalf("mkdir before: %v", err)
	}
	if err := os.MkdirAll(after, 0o755); err != nil {
		t.Fatalf("mkdir after: %v", err)
	}

	mustWrite := func(rel string) {
		beforePath := filepath.Join(before, rel)
		afterPath := filepath.Join(after, rel)
		if err := os.MkdirAll(filepath.Dir(beforePath), 0o755); err != nil {
			t.Fatalf("mkdir before file dir: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(afterPath), 0o755); err != nil {
			t.Fatalf("mkdir after file dir: %v", err)
		}
		if err := os.WriteFile(beforePath, []byte("before\n"), 0o644); err != nil {
			t.Fatalf("write before: %v", err)
		}
		if err := os.WriteFile(afterPath, []byte("after\n"), 0o644); err != nil {
			t.Fatalf("write after: %v", err)
		}
	}

	mustWrite("edge-01.log")
	mustWrite("edge-02.txt")
	mustWrite("edge-03.conf")

	pairs, err := Resolve(before, after, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}
}

func TestResolveSupportsCommaSeparatedGlobs(t *testing.T) {
	root := t.TempDir()
	before := filepath.Join(root, "before")
	after := filepath.Join(root, "after")
	if err := os.MkdirAll(before, 0o755); err != nil {
		t.Fatalf("mkdir before: %v", err)
	}
	if err := os.MkdirAll(after, 0o755); err != nil {
		t.Fatalf("mkdir after: %v", err)
	}

	for _, name := range []string{"edge-01.log", "edge-02.cfg"} {
		if err := os.WriteFile(filepath.Join(before, name), []byte("before\n"), 0o644); err != nil {
			t.Fatalf("write before: %v", err)
		}
		if err := os.WriteFile(filepath.Join(after, name), []byte("after\n"), 0o644); err != nil {
			t.Fatalf("write after: %v", err)
		}
	}

	pairs, err := Resolve(before, after, "*.log,*.cfg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
}
