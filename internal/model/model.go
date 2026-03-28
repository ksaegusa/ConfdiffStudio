package model

type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

type RuleType string

const (
	RuleLinePresent      RuleType = "line_present"
	RuleLineAbsent       RuleType = "line_absent"
	RuleDiffRequireAdded RuleType = "diff_require_added"
	RuleDiffForbidRemove RuleType = "diff_forbid_removed"
	RuleDiffAllowlist    RuleType = "diff_allowlist"
)

type AssertionSet struct {
	Version  int             `yaml:"version" json:"version"`
	Defaults AssertionConfig `yaml:"defaults" json:"defaults"`
	Rules    []Rule          `yaml:"rules" json:"rules"`
}

type AssertionConfig struct {
	MatchMode string `yaml:"match_mode" json:"match_mode"`
}

type Rule struct {
	ID            string   `yaml:"id" json:"id"`
	Type          RuleType `yaml:"type" json:"type"`
	Severity      Severity `yaml:"severity" json:"severity"`
	Pattern       string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	AllowPatterns []string `yaml:"allow_patterns,omitempty" json:"allow_patterns,omitempty"`
}

type Pair struct {
	Name   string
	Before []string
	After  []string
}

type Diff struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
}

type ReplaceRule struct {
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
}

type DiffProfile struct {
	OrderMode      string        `json:"orderMode"`
	IgnorePatterns []string      `json:"ignorePatterns,omitempty"`
	ReplaceRules   []ReplaceRule `json:"replaceRules,omitempty"`
	TargetPrefixes []string      `json:"targetPrefixes,omitempty"`
}

type StructuredDiffLine struct {
	Text  string `json:"text"`
	Depth int    `json:"depth"`
}

type StructuredDiffBlockKind string

const (
	StructuredDiffBlockAdd    StructuredDiffBlockKind = "add"
	StructuredDiffBlockRemove StructuredDiffBlockKind = "remove"
	StructuredDiffBlockChange StructuredDiffBlockKind = "change"
)

type StructuredDiffBlock struct {
	ContextPath []string                `json:"contextPath"`
	Kind        StructuredDiffBlockKind `json:"kind"`
	BeforeLines []StructuredDiffLine    `json:"beforeLines,omitempty"`
	AfterLines  []StructuredDiffLine    `json:"afterLines,omitempty"`
}

type StructuredDiffFile struct {
	Name    string                `json:"name"`
	Changed bool                  `json:"changed"`
	Blocks  []StructuredDiffBlock `json:"blocks"`
}

type DiffReportSummary struct {
	TotalComparisons     int `json:"totalComparisons"`
	ChangedComparisons   int `json:"changedComparisons"`
	SameChangeCount      int `json:"sameChangeCount"`
	DifferentChangeCount int `json:"differentChangeCount"`
	AddedBlocks          int `json:"addedBlocks"`
	RemovedBlocks        int `json:"removedBlocks"`
	ChangedBlocks        int `json:"changedBlocks"`
}

type DiffReportItem struct {
	ComparisonIndex int                   `json:"comparisonIndex"`
	Label           string                `json:"label"`
	Changed         bool                  `json:"changed"`
	BlockCount      int                   `json:"blockCount"`
	TopContexts     []string              `json:"topContexts,omitempty"`
	SameGroupSize   int                   `json:"sameGroupSize,omitempty"`
	GroupSignature  string                `json:"groupSignature,omitempty"`
	AddedBlocks     int                   `json:"addedBlocks"`
	RemovedBlocks   int                   `json:"removedBlocks"`
	ChangedBlocks   int                   `json:"changedBlocks"`
	Highlights      []DiffReportHighlight `json:"highlights,omitempty"`
}

type DiffReportGroup struct {
	Signature      string   `json:"signature"`
	FileCount      int      `json:"fileCount"`
	ItemIndexes    []int    `json:"itemIndexes"`
	PrimaryContext string   `json:"primaryContext"`
	Labels         []string `json:"labels,omitempty"`
}

type DiffReportPayload struct {
	GeneratedAt string            `json:"generatedAt"`
	Profile     DiffProfile       `json:"profile"`
	Summary     DiffReportSummary `json:"summary"`
	Items       []DiffReportItem  `json:"items"`
	Groups      []DiffReportGroup `json:"groups,omitempty"`
	Markdown    string            `json:"markdown"`
}

type DiffReportHighlight struct {
	Kind   string `json:"kind"`
	Title  string `json:"title"`
	Detail string `json:"detail,omitempty"`
}

type RuleResult struct {
	RuleID      string   `json:"rule_id"`
	Type        RuleType `json:"type"`
	Severity    Severity `json:"severity"`
	Passed      bool     `json:"passed"`
	Message     string   `json:"message"`
	MatchSample string   `json:"match_sample,omitempty"`
}

type FileReport struct {
	File        string       `json:"file"`
	Passed      bool         `json:"passed"`
	RuleResults []RuleResult `json:"rule_results"`
}

type Summary struct {
	FilesTotal  int `json:"files_total"`
	FilesPassed int `json:"files_passed"`
	FilesFailed int `json:"files_failed"`
	FailedHigh  int `json:"failed_high"`
	FailedMed   int `json:"failed_medium"`
	FailedLow   int `json:"failed_low"`
}

type CheckReport struct {
	Summary Summary      `json:"summary"`
	Files   []FileReport `json:"files"`
}
