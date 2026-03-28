package check

import (
	"testing"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func TestRunPassAndFail(t *testing.T) {
	pairs := []model.Pair{{
		Name: "fw1.cfg",
		Before: []string{
			"permit tcp 10.0.0.1 20.0.0.1 eq 80",
			"deny ip any any",
		},
		After: []string{
			"permit tcp 10.0.0.1 20.0.0.1 eq 443",
			"deny ip any any",
		},
	}}

	set := model.AssertionSet{
		Version: 1,
		Rules: []model.Rule{
			{ID: "r1", Type: model.RuleDiffRequireAdded, Severity: model.SeverityMedium, Pattern: "eq 443"},
			{ID: "r2", Type: model.RuleDiffForbidRemove, Severity: model.SeverityHigh, Pattern: "deny ip any any"},
		},
	}

	report := Run(pairs, set)
	if report.Summary.FilesFailed != 0 {
		t.Fatalf("expected pass, got fail")
	}

	set.Rules = append(set.Rules, model.Rule{
		ID: "r3", Type: model.RuleDiffAllowlist, Severity: model.SeverityHigh, AllowPatterns: []string{"eq 22"},
	})
	report = Run(pairs, set)
	if report.Summary.FilesFailed != 1 {
		t.Fatalf("expected 1 failed file, got %d", report.Summary.FilesFailed)
	}
}
