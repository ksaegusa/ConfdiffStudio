package assertions

import (
	"testing"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func TestValidateRejectsEmptyRules(t *testing.T) {
	set := model.AssertionSet{Version: 1}
	if err := Validate(set); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateAcceptsRule(t *testing.T) {
	set := model.AssertionSet{
		Version: 1,
		Rules: []model.Rule{{
			ID:       "r1",
			Type:     model.RuleLinePresent,
			Severity: model.SeverityMedium,
			Pattern:  "foo",
		}},
	}
	if err := Validate(set); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
