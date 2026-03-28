package bootstrap

import (
	"testing"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func TestGenerateCreatesRules(t *testing.T) {
	pairs := []model.Pair{{
		Name:   "fw1.cfg",
		Before: []string{"line a", "line b"},
		After:  []string{"line a", "line c"},
	}}

	set := Generate(pairs)
	if set.Version != 1 {
		t.Fatalf("expected version 1")
	}
	if len(set.Rules) < 2 {
		t.Fatalf("expected at least 2 rules, got %d", len(set.Rules))
	}
}
