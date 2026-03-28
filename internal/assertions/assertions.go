package assertions

import (
	"fmt"
	"os"
	"regexp"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
	"gopkg.in/yaml.v3"
)

func Load(path string) (model.AssertionSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.AssertionSet{}, err
	}

	var set model.AssertionSet
	if err := yaml.Unmarshal(data, &set); err != nil {
		return model.AssertionSet{}, err
	}
	return set, nil
}

func Save(path string, set model.AssertionSet) error {
	data, err := yaml.Marshal(set)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func Validate(set model.AssertionSet) error {
	if set.Version <= 0 {
		return fmt.Errorf("version must be set")
	}
	if len(set.Rules) == 0 {
		return fmt.Errorf("rules must not be empty")
	}

	seen := map[string]struct{}{}
	for _, rule := range set.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rule id is required")
		}
		if _, exists := seen[rule.ID]; exists {
			return fmt.Errorf("duplicate rule id: %s", rule.ID)
		}
		seen[rule.ID] = struct{}{}

		switch rule.Type {
		case model.RuleLinePresent, model.RuleLineAbsent, model.RuleDiffRequireAdded, model.RuleDiffForbidRemove:
			if rule.Pattern == "" {
				return fmt.Errorf("rule %s requires pattern", rule.ID)
			}
			if _, err := regexp.Compile(rule.Pattern); err != nil {
				return fmt.Errorf("rule %s invalid pattern: %w", rule.ID, err)
			}
		case model.RuleDiffAllowlist:
			if len(rule.AllowPatterns) == 0 {
				return fmt.Errorf("rule %s requires allow_patterns", rule.ID)
			}
			for _, p := range rule.AllowPatterns {
				if _, err := regexp.Compile(p); err != nil {
					return fmt.Errorf("rule %s invalid allow_patterns regex: %w", rule.ID, err)
				}
			}
		default:
			return fmt.Errorf("rule %s has unsupported type %q", rule.ID, rule.Type)
		}
	}
	return nil
}
