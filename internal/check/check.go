package check

import (
	"fmt"
	"regexp"

	"github.com/ksaegusa/ConfdiffStudio/internal/diff"
	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func Run(pairs []model.Pair, set model.AssertionSet) model.CheckReport {
	report := model.CheckReport{}
	report.Files = make([]model.FileReport, 0, len(pairs))

	for _, pair := range pairs {
		d := diff.Compute(pair.Before, pair.After)
		fileReport := model.FileReport{File: pair.Name, Passed: true}

		for _, rule := range set.Rules {
			result := evaluateRule(rule, pair.After, d)
			if !result.Passed {
				fileReport.Passed = false
				incrementFailure(&report.Summary, rule.Severity)
			}
			fileReport.RuleResults = append(fileReport.RuleResults, result)
		}

		if fileReport.Passed {
			report.Summary.FilesPassed++
		} else {
			report.Summary.FilesFailed++
		}

		report.Files = append(report.Files, fileReport)
	}

	report.Summary.FilesTotal = len(pairs)
	return report
}

func evaluateRule(rule model.Rule, afterLines []string, d model.Diff) model.RuleResult {
	res := model.RuleResult{
		RuleID:   rule.ID,
		Type:     rule.Type,
		Severity: rule.Severity,
		Passed:   true,
		Message:  "ok",
	}

	mustCompile := func(pattern string) *regexp.Regexp {
		return regexp.MustCompile(pattern)
	}

	matchAny := func(lines []string, re *regexp.Regexp) (bool, string) {
		for _, line := range lines {
			if re.MatchString(line) {
				return true, line
			}
		}
		return false, ""
	}

	switch rule.Type {
	case model.RuleLinePresent:
		re := mustCompile(rule.Pattern)
		ok, sample := matchAny(afterLines, re)
		if !ok {
			res.Passed = false
			res.Message = fmt.Sprintf("pattern not found in resulting config: %s", rule.Pattern)
		} else {
			res.MatchSample = sample
		}
	case model.RuleLineAbsent:
		re := mustCompile(rule.Pattern)
		ok, sample := matchAny(afterLines, re)
		if ok {
			res.Passed = false
			res.Message = fmt.Sprintf("forbidden pattern found in resulting config: %s", rule.Pattern)
			res.MatchSample = sample
		}
	case model.RuleDiffRequireAdded:
		re := mustCompile(rule.Pattern)
		ok, sample := matchAny(d.Added, re)
		if !ok {
			res.Passed = false
			res.Message = fmt.Sprintf("expected added line not found: %s", rule.Pattern)
		} else {
			res.MatchSample = sample
		}
	case model.RuleDiffForbidRemove:
		re := mustCompile(rule.Pattern)
		ok, sample := matchAny(d.Removed, re)
		if ok {
			res.Passed = false
			res.Message = fmt.Sprintf("forbidden removal detected: %s", rule.Pattern)
			res.MatchSample = sample
		}
	case model.RuleDiffAllowlist:
		compiled := make([]*regexp.Regexp, 0, len(rule.AllowPatterns))
		for _, p := range rule.AllowPatterns {
			compiled = append(compiled, mustCompile(p))
		}
		for _, line := range append(append([]string{}, d.Added...), d.Removed...) {
			allowed := false
			for _, re := range compiled {
				if re.MatchString(line) {
					allowed = true
					break
				}
			}
			if !allowed {
				res.Passed = false
				res.Message = "change outside allowlist detected"
				res.MatchSample = line
				break
			}
		}
	}

	return res
}

func incrementFailure(summary *model.Summary, s model.Severity) {
	switch s {
	case model.SeverityHigh:
		summary.FailedHigh++
	case model.SeverityMedium:
		summary.FailedMed++
	default:
		summary.FailedLow++
	}
}
