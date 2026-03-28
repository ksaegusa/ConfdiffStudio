package license

import "strings"

const (
	FeatureStrictMode   = "strict_mode"
	FeatureReplaceRules = "replace_rules"
	FeatureTargetPrefix = "target_prefixes"
	FeatureReportExport = "report_export"
)

type Limits struct {
	MaxPairs          int `json:"max_pairs"`
	MaxBytesPerSide   int `json:"max_bytes_per_side"`
	MaxIgnorePatterns int `json:"max_ignore_patterns"`
	MaxReplaceRules   int `json:"max_replace_rules"`
	MaxTargetPrefixes int `json:"max_target_prefixes"`
}

type Policy struct {
	Tier         string
	Entitlements map[string]struct{}
	Limits       Limits
}

type CheckProfile struct {
	OrderMode      string
	IgnorePatterns []string
	ReplaceRules   int
	TargetPrefixes int
}

type CheckPair struct {
	BeforeBytes int
	AfterBytes  int
}

type CheckInput struct {
	Pairs   []CheckPair
	Profile CheckProfile
}

type PolicyViolation struct {
	Code    string
	Message string
	Details map[string]any
}

func FreePolicy() Policy {
	limits := Limits{
		MaxPairs:          3,
		MaxBytesPerSide:   200 * 1024,
		MaxIgnorePatterns: 5,
		MaxReplaceRules:   0,
		MaxTargetPrefixes: 0,
	}
	return Policy{
		Tier:         "free",
		Entitlements: map[string]struct{}{},
		Limits:       limits,
	}
}

func ProPolicy() Policy {
	limits := Limits{
		MaxPairs:          50,
		MaxBytesPerSide:   2 * 1024 * 1024,
		MaxIgnorePatterns: 200,
		MaxReplaceRules:   100,
		MaxTargetPrefixes: 100,
	}
	return Policy{
		Tier: "pro",
		Entitlements: map[string]struct{}{
			FeatureStrictMode:   {},
			FeatureReplaceRules: {},
			FeatureTargetPrefix: {},
			FeatureReportExport: {},
		},
		Limits: limits,
	}
}

func PolicyForStatus(status Status) Policy {
	plan := strings.ToLower(strings.TrimSpace(status.Plan))
	if status.Valid && plan == "pro" {
		return ProPolicy()
	}
	return FreePolicy()
}

func HasFeature(policy Policy, feature string) bool {
	_, ok := policy.Entitlements[feature]
	return ok
}

func ValidateCheckInput(policy Policy, in CheckInput) *PolicyViolation {
	if len(in.Pairs) == 0 {
		return &PolicyViolation{
			Code:    "LICENSE_LIMIT_EXCEEDED",
			Message: "no pairs provided",
			Details: map[string]any{"limit": policy.Limits.MaxPairs},
		}
	}

	if len(in.Pairs) > policy.Limits.MaxPairs {
		return &PolicyViolation{
			Code:    "LICENSE_LIMIT_EXCEEDED",
			Message: "pair count exceeds plan limit",
			Details: map[string]any{"field": "pairs", "limit": policy.Limits.MaxPairs, "actual": len(in.Pairs)},
		}
	}

	for idx, pair := range in.Pairs {
		if pair.BeforeBytes > policy.Limits.MaxBytesPerSide || pair.AfterBytes > policy.Limits.MaxBytesPerSide {
			return &PolicyViolation{
				Code:    "LICENSE_LIMIT_EXCEEDED",
				Message: "config size exceeds plan limit",
				Details: map[string]any{
					"field":              "pair_size_bytes",
					"index":              idx,
					"limit":              policy.Limits.MaxBytesPerSide,
					"before_bytes":       pair.BeforeBytes,
					"after_bytes":        pair.AfterBytes,
					"max_bytes_per_side": policy.Limits.MaxBytesPerSide,
				},
			}
		}
	}

	if len(in.Profile.IgnorePatterns) > policy.Limits.MaxIgnorePatterns {
		return &PolicyViolation{
			Code:    "LICENSE_LIMIT_EXCEEDED",
			Message: "ignore patterns exceed plan limit",
			Details: map[string]any{
				"field":  "ignorePatterns",
				"limit":  policy.Limits.MaxIgnorePatterns,
				"actual": len(in.Profile.IgnorePatterns),
			},
		}
	}

	orderMode := strings.ToLower(strings.TrimSpace(in.Profile.OrderMode))
	if orderMode == "strict" && !HasFeature(policy, FeatureStrictMode) {
		return &PolicyViolation{
			Code:    "LICENSE_FEATURE_BLOCKED",
			Message: "strict mode is available in pro tier only",
			Details: map[string]any{
				"feature":       FeatureStrictMode,
				"required_tier": "pro",
				"current_tier":  policy.Tier,
			},
		}
	}

	if in.Profile.ReplaceRules > 0 {
		if !HasFeature(policy, FeatureReplaceRules) {
			return &PolicyViolation{
				Code:    "LICENSE_FEATURE_BLOCKED",
				Message: "replace rules are available in pro tier only",
				Details: map[string]any{
					"feature":       FeatureReplaceRules,
					"required_tier": "pro",
					"current_tier":  policy.Tier,
				},
			}
		}
		if in.Profile.ReplaceRules > policy.Limits.MaxReplaceRules {
			return &PolicyViolation{
				Code:    "LICENSE_LIMIT_EXCEEDED",
				Message: "replace rules exceed plan limit",
				Details: map[string]any{
					"field":  "replaceRules",
					"limit":  policy.Limits.MaxReplaceRules,
					"actual": in.Profile.ReplaceRules,
				},
			}
		}
	}

	if in.Profile.TargetPrefixes > 0 {
		if !HasFeature(policy, FeatureTargetPrefix) {
			return &PolicyViolation{
				Code:    "LICENSE_FEATURE_BLOCKED",
				Message: "target prefixes are available in pro tier only",
				Details: map[string]any{
					"feature":       FeatureTargetPrefix,
					"required_tier": "pro",
					"current_tier":  policy.Tier,
				},
			}
		}
		if in.Profile.TargetPrefixes > policy.Limits.MaxTargetPrefixes {
			return &PolicyViolation{
				Code:    "LICENSE_LIMIT_EXCEEDED",
				Message: "target prefixes exceed plan limit",
				Details: map[string]any{
					"field":  "targetPrefixes",
					"limit":  policy.Limits.MaxTargetPrefixes,
					"actual": in.Profile.TargetPrefixes,
				},
			}
		}
	}

	return nil
}
