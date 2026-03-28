package report

import (
	"strings"
	"testing"
	"time"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func TestBuildDiffReportSummarizesDiffs(t *testing.T) {
	structured := []model.StructuredDiffFile{
		{
			Name:    "edge-01.log",
			Changed: true,
			Blocks: []model.StructuredDiffBlock{
				{
					ContextPath: []string{"interface Gi0/0"},
					Kind:        model.StructuredDiffBlockChange,
					BeforeLines: []model.StructuredDiffLine{{Depth: 0, Text: "description old"}},
					AfterLines:  []model.StructuredDiffLine{{Depth: 0, Text: "description new"}},
				},
			},
		},
		{
			Name:    "edge-02.log",
			Changed: true,
			Blocks: []model.StructuredDiffBlock{
				{
					ContextPath: []string{"interface Gi0/0"},
					Kind:        model.StructuredDiffBlockChange,
					BeforeLines: []model.StructuredDiffLine{{Depth: 0, Text: "description old"}},
					AfterLines:  []model.StructuredDiffLine{{Depth: 0, Text: "description new"}},
				},
			},
		},
		{
			Name:    "edge-03.log",
			Changed: true,
			Blocks: []model.StructuredDiffBlock{
				{
					ContextPath: []string{"ip access-list extended LAN-OUT"},
					Kind:        model.StructuredDiffBlockAdd,
					AfterLines:  []model.StructuredDiffLine{{Depth: 0, Text: "permit ip any any"}},
				},
			},
		},
		{
			Name:    "edge-04.log",
			Changed: false,
			Blocks:  nil,
		},
	}

	checkReport := model.CheckReport{
		Summary: model.Summary{
			FilesTotal:  4,
			FilesPassed: 3,
			FilesFailed: 1,
			FailedHigh:  1,
		},
	}

	payload := BuildDiffReport(structured, checkReport, model.DiffProfile{
		OrderMode:      "lenient",
		IgnorePatterns: []string{"^ntp"},
		TargetPrefixes: []string{"interface"},
	}, time.Date(2025, time.January, 2, 3, 4, 5, 0, time.UTC))

	if payload.Summary.TotalComparisons != 4 {
		t.Fatalf("unexpected total comparisons: %d", payload.Summary.TotalComparisons)
	}
	if payload.Summary.ChangedComparisons != 3 {
		t.Fatalf("unexpected changed comparisons: %d", payload.Summary.ChangedComparisons)
	}
	if payload.Summary.SameChangeCount != 2 {
		t.Fatalf("unexpected same change count: %d", payload.Summary.SameChangeCount)
	}
	if payload.Summary.DifferentChangeCount != 1 {
		t.Fatalf("unexpected different change count: %d", payload.Summary.DifferentChangeCount)
	}
	if len(payload.Groups) != 2 {
		t.Fatalf("unexpected group count: %d", len(payload.Groups))
	}
	if payload.Groups[0].FileCount != 2 {
		t.Fatalf("unexpected first group size: %+v", payload.Groups[0])
	}
	if payload.Items[0].ComparisonIndex != 0 {
		t.Fatalf("unexpected comparison index: %+v", payload.Items[0])
	}
	if payload.Items[0].SameGroupSize != 2 {
		t.Fatalf("unexpected same group size: %+v", payload.Items[0])
	}
	if payload.Items[0].GroupSignature == "" {
		t.Fatalf("expected group signature, got %+v", payload.Items[0])
	}
	if len(payload.Items[0].Highlights) == 0 {
		t.Fatalf("expected highlights, got %+v", payload.Items[0])
	}
	if payload.Items[0].Highlights[0].Title != "description（変更）" {
		t.Fatalf("unexpected highlight title: %+v", payload.Items[0].Highlights[0])
	}
	if payload.Items[0].Highlights[0].Detail != "old -> new" {
		t.Fatalf("unexpected highlight detail: %+v", payload.Items[0].Highlights[0])
	}
	if !strings.Contains(payload.Markdown, "ConfdiffStudio verification report") {
		t.Fatalf("expected markdown report title, got %q", payload.Markdown)
	}
	if !strings.Contains(payload.Markdown, "## 確認内容") {
		t.Fatalf("expected markdown to include verification purpose, got %q", payload.Markdown)
	}
	if !strings.Contains(payload.Markdown, "## 検証方法") {
		t.Fatalf("expected markdown to include verification methods, got %q", payload.Markdown)
	}
	if !strings.Contains(payload.Markdown, "edge-01.log") {
		t.Fatalf("expected markdown to include item label, got %q", payload.Markdown)
	}
	if !strings.Contains(payload.Markdown, "差分あり（3件）") {
		t.Fatalf("expected markdown to include conclusion headline, got %q", payload.Markdown)
	}
	if !strings.Contains(payload.Markdown, "description（変更）: old -> new") {
		t.Fatalf("expected markdown to include before-after summary, got %q", payload.Markdown)
	}
	if !strings.Contains(payload.Markdown, "比較順序: lenient") {
		t.Fatalf("expected markdown to include verification conditions, got %q", payload.Markdown)
	}
}
