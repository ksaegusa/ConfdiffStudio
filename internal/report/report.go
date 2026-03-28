package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ksaegusa/ConfdiffStudio/internal/model"
)

func WriteJSON(path string, r model.CheckReport) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func WriteMarkdown(path string, r model.CheckReport) error {
	return os.WriteFile(path, []byte(RenderMarkdown(r)), 0o644)
}

func RenderMarkdown(r model.CheckReport) string {
	var b strings.Builder
	b.WriteString("# confdiff report\n\n")
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("- Files total: %d\n", r.Summary.FilesTotal))
	b.WriteString(fmt.Sprintf("- Files passed: %d\n", r.Summary.FilesPassed))
	b.WriteString(fmt.Sprintf("- Files failed: %d\n", r.Summary.FilesFailed))
	b.WriteString(fmt.Sprintf("- Failed high: %d\n", r.Summary.FailedHigh))
	b.WriteString(fmt.Sprintf("- Failed medium: %d\n", r.Summary.FailedMed))
	b.WriteString(fmt.Sprintf("- Failed low: %d\n", r.Summary.FailedLow))

	b.WriteString("\n## Files\n\n")
	for _, file := range r.Files {
		status := "PASS"
		if !file.Passed {
			status = "FAIL"
		}
		b.WriteString(fmt.Sprintf("### %s [%s]\n\n", file.File, status))
		for _, rr := range file.RuleResults {
			line := fmt.Sprintf("- `%s` (%s/%s): %s", rr.RuleID, rr.Type, rr.Severity, rr.Message)
			if rr.MatchSample != "" {
				line += fmt.Sprintf(" (`%s`)", rr.MatchSample)
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}

func BuildDiffReport(
	structured []model.StructuredDiffFile,
	checkReport model.CheckReport,
	profile model.DiffProfile,
	generatedAt time.Time,
) model.DiffReportPayload {
	groups, signatureCounts, itemSignatures := buildDiffGroups(structured)

	payload := model.DiffReportPayload{
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Profile:     profile,
		Summary: model.DiffReportSummary{
			TotalComparisons:   len(structured),
			ChangedComparisons: 0,
		},
		Items:  make([]model.DiffReportItem, 0, len(structured)),
		Groups: groups,
	}

	for index, file := range structured {
		item := model.DiffReportItem{
			ComparisonIndex: index,
			Label:           file.Name,
			Changed:         file.Changed,
			BlockCount:      len(file.Blocks),
		}
		if file.Changed {
			payload.Summary.ChangedComparisons++
		}
		contexts := map[string]struct{}{}
		for _, block := range file.Blocks {
			switch block.Kind {
			case model.StructuredDiffBlockAdd:
				item.AddedBlocks++
				payload.Summary.AddedBlocks++
			case model.StructuredDiffBlockRemove:
				item.RemovedBlocks++
				payload.Summary.RemovedBlocks++
			default:
				item.ChangedBlocks++
				payload.Summary.ChangedBlocks++
			}
			label := blockContextLabel(block)
			if label == "" {
				continue
			}
			if _, ok := contexts[label]; ok {
				continue
			}
			contexts[label] = struct{}{}
			item.TopContexts = append(item.TopContexts, label)
		}
		if len(item.TopContexts) > 3 {
			item.TopContexts = item.TopContexts[:3]
		}
		item.Highlights = buildHighlights(file.Blocks)
		if file.Changed {
			item.GroupSignature = itemSignatures[index]
			item.SameGroupSize = signatureCounts[item.GroupSignature]
		}
		payload.Items = append(payload.Items, item)
	}

	sort.Slice(payload.Items, func(i, j int) bool {
		if payload.Items[i].Changed != payload.Items[j].Changed {
			return payload.Items[i].Changed
		}
		if payload.Items[i].BlockCount != payload.Items[j].BlockCount {
			return payload.Items[i].BlockCount > payload.Items[j].BlockCount
		}
		return payload.Items[i].Label < payload.Items[j].Label
	})

	for _, item := range payload.Items {
		if !item.Changed {
			continue
		}
		if item.SameGroupSize > 1 {
			payload.Summary.SameChangeCount++
		} else {
			payload.Summary.DifferentChangeCount++
		}
	}

	payload.Markdown = RenderDiffMarkdown(payload, checkReport)
	return payload
}

func buildDiffGroups(
	structured []model.StructuredDiffFile,
) ([]model.DiffReportGroup, map[string]int, map[int]string) {
	grouped := map[string]*model.DiffReportGroup{}
	signatureCounts := map[string]int{}
	itemSignatures := map[int]string{}

	for index, file := range structured {
		if !file.Changed || len(file.Blocks) == 0 {
			continue
		}

		signature := buildFileSignature(file)
		itemSignatures[index] = signature
		signatureCounts[signature]++

		group, ok := grouped[signature]
		if !ok {
			group = &model.DiffReportGroup{
				Signature:      signature,
				FileCount:      0,
				ItemIndexes:    []int{},
				PrimaryContext: blockContextLabel(file.Blocks[0]),
				Labels:         []string{},
			}
			grouped[signature] = group
		}

		group.FileCount++
		group.ItemIndexes = append(group.ItemIndexes, index)
		group.Labels = append(group.Labels, file.Name)
	}

	groups := make([]model.DiffReportGroup, 0, len(grouped))
	for _, group := range grouped {
		groups = append(groups, *group)
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].FileCount != groups[j].FileCount {
			return groups[i].FileCount > groups[j].FileCount
		}
		return groups[i].PrimaryContext < groups[j].PrimaryContext
	})

	return groups, signatureCounts, itemSignatures
}

func RenderDiffMarkdown(payload model.DiffReportPayload, checkReport model.CheckReport) string {
	var b strings.Builder
	b.WriteString("# ConfdiffStudio verification report\n\n")
	b.WriteString("## 確認内容\n\n")
	b.WriteString("ネットワーク機器の設定差分を確認\n")

	b.WriteString("\n## 確認対象\n\n")
	for _, item := range payload.Items {
		b.WriteString(fmt.Sprintf("- %s（Before）\n", item.Label))
		b.WriteString(fmt.Sprintf("- %s（After）\n", item.Label))
	}

	b.WriteString("\n## 検証方法\n\n")
	for _, method := range verificationMethods(payload.Profile) {
		b.WriteString(fmt.Sprintf("- %s\n", method))
	}

	b.WriteString("\n## 確認結果\n\n")
	b.WriteString(fmt.Sprintf("- %s\n", reportHeadline(payload.Summary)))
	if class := reportClassification(payload.Summary); class != "" {
		b.WriteString(fmt.Sprintf("- %s\n", class))
	}
	if payload.Summary.ChangedComparisons > 0 {
		b.WriteString(fmt.Sprintf("- ユニークな変更: %d件\n", payload.Summary.DifferentChangeCount))
		b.WriteString(fmt.Sprintf("- 共通の変更: %d件\n", payload.Summary.SameChangeCount))
		b.WriteString(fmt.Sprintf("- 差分ブロック: %d件\n", totalBlockCount(payload.Summary)))
		b.WriteString("※ ユニークな変更 = 他の比較対象に存在しない変更\n")
	}
	b.WriteString(fmt.Sprintf("- 生成日時: %s\n", payload.GeneratedAt))
	b.WriteString(fmt.Sprintf("- 対象: %d件\n", payload.Summary.TotalComparisons))
	b.WriteString(fmt.Sprintf("- 検査ファイル: %d件中 %d件で失敗\n", checkReport.Summary.FilesTotal, checkReport.Summary.FilesFailed))

	b.WriteString("\n## 変更内容（確認結果の根拠）\n\n")
	for _, item := range payload.Items {
		b.WriteString(fmt.Sprintf("### %s\n\n", item.Label))
		if item.Changed {
			b.WriteString("- 結果: 差分あり\n")
			if item.SameGroupSize > 1 {
				b.WriteString(fmt.Sprintf("- 分類: 共通の変更（%d件）\n", item.SameGroupSize))
			} else {
				b.WriteString("- 分類: ユニークな変更\n")
			}
		} else {
			b.WriteString("- 結果: 差分なし（完全一致）\n")
		}
		b.WriteString(fmt.Sprintf("- 差分ブロック: %d\n", item.BlockCount))
		if len(item.Highlights) > 0 {
			b.WriteString("- 根拠:\n")
			for _, highlight := range item.Highlights {
				if highlight.Detail != "" {
					b.WriteString(fmt.Sprintf("  - %s: %s\n", highlight.Title, highlight.Detail))
				} else {
					b.WriteString(fmt.Sprintf("  - %s\n", highlight.Title))
				}
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("## 検証条件\n\n")
	b.WriteString(fmt.Sprintf("- 比較順序: %s\n", emptyFallback(payload.Profile.OrderMode, "lenient")))
	b.WriteString(fmt.Sprintf("- 除外パターン: %s\n", countOrNone(len(payload.Profile.IgnorePatterns))))
	b.WriteString(fmt.Sprintf("- 正規化ルール: %s\n", countOrNone(len(payload.Profile.ReplaceRules))))
	b.WriteString(fmt.Sprintf("- 対象ブロック: %s\n", countOrNone(len(payload.Profile.TargetPrefixes))))

	return b.String()
}

func verificationMethods(profile model.DiffProfile) []string {
	methods := []string{
		"Before / After の設定ファイルを比較",
		"設定単位で構造差分を抽出",
	}
	if emptyFallback(profile.OrderMode, "lenient") == "strict" {
		methods = append(methods, "順序差分も比較（strictモード）")
	} else {
		methods = append(methods, "順序差分は無視（lenientモード）")
	}
	if len(profile.TargetPrefixes) > 0 {
		methods = append(methods, "対象ブロックを絞って確認")
	}
	if len(profile.IgnorePatterns) > 0 {
		methods = append(methods, "除外パターンを適用して確認")
	}
	if len(profile.ReplaceRules) > 0 {
		methods = append(methods, "正規化ルールを適用して確認")
	}
	return methods
}

func countOrNone(count int) string {
	if count == 0 {
		return "なし"
	}
	return fmt.Sprintf("%d件", count)
}

func buildHighlights(blocks []model.StructuredDiffBlock) []model.DiffReportHighlight {
	highlights := make([]model.DiffReportHighlight, 0, len(blocks))
	for _, block := range blocks {
		highlight := summarizeBlock(block)
		if highlight.Title == "" {
			continue
		}
		highlights = append(highlights, highlight)
		if len(highlights) == 3 {
			break
		}
	}
	return highlights
}

func summarizeBlock(block model.StructuredDiffBlock) model.DiffReportHighlight {
	switch block.Kind {
	case model.StructuredDiffBlockAdd:
		title := contextLabelForSummary(block)
		if title == "" {
			title = "設定を追加"
		} else {
			title += " に追加"
		}
		return model.DiffReportHighlight{
			Kind:   string(block.Kind),
			Title:  title,
			Detail: firstLineText(block.AfterLines),
		}
	case model.StructuredDiffBlockRemove:
		title := contextLabelForSummary(block)
		if title == "" {
			title = "設定を削除"
		} else {
			title += " から削除"
		}
		return model.DiffReportHighlight{
			Kind:   string(block.Kind),
			Title:  title,
			Detail: firstLineText(block.BeforeLines),
		}
	default:
		if len(block.BeforeLines) == 1 && len(block.AfterLines) == 1 {
			return summarizeSingleLineChange(block)
		}
		title := contextLabelForSummary(block)
		if title == "" {
			title = "設定（変更あり）"
		} else {
			title += "（変更あり）"
		}
		return model.DiffReportHighlight{
			Kind:   string(block.Kind),
			Title:  title,
			Detail: fmt.Sprintf("%d 行の差分", max(len(block.BeforeLines), len(block.AfterLines))),
		}
	}
}

func summarizeSingleLineChange(block model.StructuredDiffBlock) model.DiffReportHighlight {
	before := block.BeforeLines[0].Text
	after := block.AfterLines[0].Text
	subject := commonTokenPrefix(before, after)
	title := ""
	beforeValue := before
	afterValue := after
	if subject != "" {
		title = subject + "（変更）"
		beforeValue = strings.TrimSpace(strings.TrimPrefix(before, subject))
		afterValue = strings.TrimSpace(strings.TrimPrefix(after, subject))
	}
	if title == "" {
		context := contextLabelForSummary(block)
		if context != "" {
			title = context + "（変更）"
		} else {
			title = "設定値（変更）"
		}
	}
	if beforeValue == "" {
		beforeValue = before
	}
	if afterValue == "" {
		afterValue = after
	}
	return model.DiffReportHighlight{
		Kind:   string(block.Kind),
		Title:  title,
		Detail: fmt.Sprintf("%s -> %s", beforeValue, afterValue),
	}
}

func commonTokenPrefix(before, after string) string {
	beforeTokens := strings.Fields(before)
	afterTokens := strings.Fields(after)
	limit := min(len(beforeTokens), len(afterTokens))
	matched := 0
	for matched < limit && beforeTokens[matched] == afterTokens[matched] {
		matched++
	}
	if matched == 0 {
		return ""
	}
	return strings.Join(beforeTokens[:matched], " ")
}

func contextLabelForSummary(block model.StructuredDiffBlock) string {
	if len(block.ContextPath) == 0 {
		return ""
	}
	return block.ContextPath[len(block.ContextPath)-1]
}

func firstLineText(lines []model.StructuredDiffLine) string {
	if len(lines) == 0 {
		return ""
	}
	return lines[0].Text
}

func reportHeadline(summary model.DiffReportSummary) string {
	if summary.ChangedComparisons == 0 {
		return "差分なし"
	}
	return fmt.Sprintf("差分あり（%d件）", summary.ChangedComparisons)
}

func reportClassification(summary model.DiffReportSummary) string {
	if summary.ChangedComparisons == 0 {
		return ""
	}
	if summary.DifferentChangeCount == summary.ChangedComparisons {
		return "すべてユニークな変更"
	}
	if summary.SameChangeCount == summary.ChangedComparisons {
		return "すべて共通の変更"
	}
	return fmt.Sprintf("ユニークな変更 %d件 / 共通の変更 %d件", summary.DifferentChangeCount, summary.SameChangeCount)
}

func totalBlockCount(summary model.DiffReportSummary) int {
	return summary.AddedBlocks + summary.RemovedBlocks + summary.ChangedBlocks
}

func buildFileSignature(file model.StructuredDiffFile) string {
	blockSignatures := make([]string, 0, len(file.Blocks))
	for _, block := range file.Blocks {
		blockSignatures = append(blockSignatures, buildBlockSignature(block))
	}
	sort.Strings(blockSignatures)
	return strings.Join(blockSignatures, "\n---\n")
}

func buildBlockSignature(block model.StructuredDiffBlock) string {
	return strings.Join([]string{
		fmt.Sprintf("context=%s", strings.Join(block.ContextPath, " > ")),
		fmt.Sprintf("kind=%s", block.Kind),
		fmt.Sprintf("before=%s", serializeLines(block.BeforeLines)),
		fmt.Sprintf("after=%s", serializeLines(block.AfterLines)),
	}, "\n")
}

func serializeLines(lines []model.StructuredDiffLine) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d:%s", line.Depth, line.Text))
	}
	return strings.Join(parts, "\n")
}

func blockContextLabel(block model.StructuredDiffBlock) string {
	if len(block.ContextPath) > 0 {
		return strings.Join(block.ContextPath, " > ")
	}
	if len(block.AfterLines) > 0 {
		return fmt.Sprintf("%s: %s", block.Kind, block.AfterLines[0].Text)
	}
	if len(block.BeforeLines) > 0 {
		return fmt.Sprintf("%s: %s", block.Kind, block.BeforeLines[0].Text)
	}
	return ""
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func formatReplaceRules(values []model.ReplaceRule) string {
	if len(values) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%s => %s", value.Pattern, value.Replacement))
	}
	return strings.Join(parts, ", ")
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
