package license

import "testing"

func TestPolicyForStatus(t *testing.T) {
	free := PolicyForStatus(Status{Valid: false, Plan: "pro"})
	if free.Tier != "free" {
		t.Fatalf("expected free tier for invalid license, got %s", free.Tier)
	}

	pro := PolicyForStatus(Status{Valid: true, Plan: "pro"})
	if pro.Tier != "pro" {
		t.Fatalf("expected pro tier, got %s", pro.Tier)
	}
}

func TestValidateCheckInputFreeBlocksStrict(t *testing.T) {
	violation := ValidateCheckInput(FreePolicy(), CheckInput{
		Pairs: []CheckPair{{BeforeBytes: 100, AfterBytes: 100}},
		Profile: CheckProfile{
			OrderMode: "strict",
		},
	})
	if violation == nil {
		t.Fatal("expected strict mode violation for free plan")
	}
	if violation.Code != "LICENSE_FEATURE_BLOCKED" {
		t.Fatalf("unexpected code: %s", violation.Code)
	}
}

func TestValidateCheckInputFreePairLimit(t *testing.T) {
	violation := ValidateCheckInput(FreePolicy(), CheckInput{
		Pairs: []CheckPair{
			{BeforeBytes: 1, AfterBytes: 1},
			{BeforeBytes: 1, AfterBytes: 1},
			{BeforeBytes: 1, AfterBytes: 1},
			{BeforeBytes: 1, AfterBytes: 1},
		},
	})
	if violation == nil {
		t.Fatal("expected pair limit violation for free plan")
	}
	if violation.Code != "LICENSE_LIMIT_EXCEEDED" {
		t.Fatalf("unexpected code: %s", violation.Code)
	}
}

func TestValidateCheckInputProAllowsFeatures(t *testing.T) {
	violation := ValidateCheckInput(ProPolicy(), CheckInput{
		Pairs: []CheckPair{{BeforeBytes: 1024, AfterBytes: 1024}},
		Profile: CheckProfile{
			OrderMode:      "strict",
			IgnorePatterns: []string{"^ntp"},
			ReplaceRules:   2,
			TargetPrefixes: 2,
		},
	})
	if violation != nil {
		t.Fatalf("unexpected violation: %+v", violation)
	}
}

func TestProPolicyIncludesReportExport(t *testing.T) {
	if !HasFeature(ProPolicy(), FeatureReportExport) {
		t.Fatal("expected pro policy to include report export")
	}
	if HasFeature(FreePolicy(), FeatureReportExport) {
		t.Fatal("did not expect free policy to include report export")
	}
}
