package bootstrap

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ksaegusa/ConfdiffStudio/internal/diff"
	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func Generate(pairs []model.Pair) model.AssertionSet {
	rules := []model.Rule{}
	seen := map[string]struct{}{}

	for _, p := range pairs {
		d := diff.Compute(p.Before, p.After)

		for _, line := range d.Added {
			id := makeRuleID("added", line)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			rules = append(rules, model.Rule{
				ID:       id,
				Type:     model.RuleDiffRequireAdded,
				Severity: model.SeverityMedium,
				Pattern:  regexp.QuoteMeta(line),
			})
		}

		for _, line := range d.Removed {
			id := makeRuleID("removed", line)
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			rules = append(rules, model.Rule{
				ID:       id,
				Type:     model.RuleDiffForbidRemove,
				Severity: model.SeverityHigh,
				Pattern:  regexp.QuoteMeta(line),
			})
		}
	}

	return model.AssertionSet{
		Version: 1,
		Defaults: model.AssertionConfig{
			MatchMode: "regex",
		},
		Rules: rules,
	}
}

func makeRuleID(prefix, line string) string {
	line = strings.ToLower(line)
	line = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(line, "-")
	line = strings.Trim(line, "-")
	if line == "" {
		line = "line"
	}
	if len(line) > 40 {
		line = line[:40]
	}
	return fmt.Sprintf("%s-%s", prefix, line)
}
