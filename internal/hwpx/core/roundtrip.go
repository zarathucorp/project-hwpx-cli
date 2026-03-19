package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type roundtripState struct {
	snapshot RoundtripSnapshot
	analysis TemplateAnalysis
}

func RoundtripCheck(targetPath string) (RoundtripCheckReport, error) {
	before, err := buildRoundtripState(targetPath)
	if err != nil {
		return RoundtripCheckReport{}, err
	}

	tempDir, err := os.MkdirTemp("", "hwpxctl-roundtrip-*")
	if err != nil {
		return RoundtripCheckReport{}, err
	}
	defer os.RemoveAll(tempDir)

	roundtripTarget, err := createRoundtripArtifact(targetPath, tempDir)
	if err != nil {
		return RoundtripCheckReport{}, err
	}

	after, err := buildRoundtripState(roundtripTarget)
	if err != nil {
		return RoundtripCheckReport{}, err
	}

	issues := compareRoundtripStates(before, after)
	return RoundtripCheckReport{
		Passed: len(issues) == 0,
		Before: before.snapshot,
		After:  after.snapshot,
		Issues: issues,
	}, nil
}

func createRoundtripArtifact(targetPath, tempDir string) (string, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		outputFile := filepath.Join(tempDir, "roundtrip.hwpx")
		if err := Pack(targetPath, outputFile); err != nil {
			return "", err
		}
		return outputFile, nil
	}

	unpackedDir := filepath.Join(tempDir, "unpacked")
	if err := Unpack(targetPath, unpackedDir); err != nil {
		return "", err
	}

	outputFile := filepath.Join(tempDir, "repacked.hwpx")
	if err := Pack(unpackedDir, outputFile); err != nil {
		return "", err
	}
	return outputFile, nil
}

func buildRoundtripState(targetPath string) (roundtripState, error) {
	report, err := Validate(targetPath)
	if err != nil {
		return roundtripState{}, err
	}
	if !report.Valid {
		return roundtripState{}, fmt.Errorf("cannot roundtrip-check invalid target: %s", strings.Join(report.Errors, "; "))
	}

	analysis, err := AnalyzeTemplate(targetPath)
	if err != nil {
		return roundtripState{}, err
	}

	text, err := extractTextFromTarget(targetPath)
	if err != nil {
		return roundtripState{}, err
	}

	return roundtripState{
		analysis: analysis,
		snapshot: RoundtripSnapshot{
			Valid:            report.Valid,
			RenderSafe:       report.RenderSafe,
			RiskHints:        append([]string{}, report.RiskHints...),
			SectionPaths:     append([]string{}, report.Summary.SectionPath...),
			SectionCount:     analysis.SectionCount,
			TableCount:       analysis.TableCount,
			ParagraphCount:   analysis.ParagraphCount,
			PlaceholderCount: analysis.PlaceholderCount,
			GuideCount:       analysis.GuideCount,
			TextLength:       len(text),
			LineCount:        roundtripLineCount(text),
			TextDigest:       hashString(text),
			ParagraphDigest:  hashStrings(paragraphSignatures(analysis.Paragraphs)),
			TableDigest:      hashStrings(tableSignatures(analysis.Tables)),
		},
	}, nil
}

func extractTextFromTarget(targetPath string) (string, error) {
	entries, err := readEntriesFromTarget(targetPath)
	if err != nil {
		return "", err
	}

	report, err := inspectEntries(entries)
	if err != nil {
		return "", err
	}
	if !report.Valid {
		return "", fmt.Errorf("cannot extract text from invalid HWPX package: %s", strings.Join(report.Errors, "; "))
	}

	var paragraphs []string
	for _, sectionPath := range report.Summary.SectionPath {
		texts, extractErr := extractParagraphs(entries[sectionPath])
		if extractErr != nil {
			return "", fmt.Errorf("extract section text %s: %w", sectionPath, extractErr)
		}
		paragraphs = append(paragraphs, texts...)
	}
	return strings.Join(paragraphs, "\n"), nil
}

func compareRoundtripStates(before, after roundtripState) []RoundtripIssue {
	var issues []RoundtripIssue
	beforeSnapshot := before.snapshot
	afterSnapshot := after.snapshot

	appendDiffIssue := func(code string, beforeValue, afterValue int) {
		if beforeValue == afterValue {
			return
		}
		issues = append(issues, RoundtripIssue{
			Code:     code,
			Severity: "error",
			Message:  fmt.Sprintf("%s changed from %d to %d", code, beforeValue, afterValue),
		})
	}

	if beforeSnapshot.Valid != afterSnapshot.Valid {
		issues = append(issues, RoundtripIssue{
			Code:     "validity-changed",
			Severity: "error",
			Message:  fmt.Sprintf("valid changed from %t to %t", beforeSnapshot.Valid, afterSnapshot.Valid),
		})
	}
	if beforeSnapshot.RenderSafe != afterSnapshot.RenderSafe {
		issues = append(issues, RoundtripIssue{
			Code:     "render-safe-changed",
			Severity: "error",
			Message:  fmt.Sprintf("renderSafe changed from %t to %t", beforeSnapshot.RenderSafe, afterSnapshot.RenderSafe),
		})
	}

	appendDiffIssue("section-count-changed", beforeSnapshot.SectionCount, afterSnapshot.SectionCount)
	appendDiffIssue("table-count-changed", beforeSnapshot.TableCount, afterSnapshot.TableCount)
	appendDiffIssue("paragraph-count-changed", beforeSnapshot.ParagraphCount, afterSnapshot.ParagraphCount)
	appendDiffIssue("placeholder-count-changed", beforeSnapshot.PlaceholderCount, afterSnapshot.PlaceholderCount)
	appendDiffIssue("guide-count-changed", beforeSnapshot.GuideCount, afterSnapshot.GuideCount)
	appendDiffIssue("text-length-changed", beforeSnapshot.TextLength, afterSnapshot.TextLength)
	appendDiffIssue("line-count-changed", beforeSnapshot.LineCount, afterSnapshot.LineCount)

	if beforeSnapshot.TextDigest != afterSnapshot.TextDigest {
		issues = append(issues, RoundtripIssue{
			Code:     "text-digest-changed",
			Severity: "error",
			Message:  "ordered document text changed after roundtrip",
			Before:   beforeSnapshot.TextDigest,
			After:    afterSnapshot.TextDigest,
		})
	}
	if beforeSnapshot.ParagraphDigest != afterSnapshot.ParagraphDigest {
		issues = append(issues, compareRoundtripParagraphs(before.analysis.Paragraphs, after.analysis.Paragraphs)...)
	}
	if beforeSnapshot.TableDigest != afterSnapshot.TableDigest {
		issues = append(issues, compareRoundtripTables(before.analysis.Tables, after.analysis.Tables)...)
	}

	for index, beforePath := range beforeSnapshot.SectionPaths {
		if index >= len(afterSnapshot.SectionPaths) {
			issues = append(issues, RoundtripIssue{
				Code:     "section-path-removed",
				Severity: "error",
				Message:  fmt.Sprintf("section path %q was removed", beforePath),
				Before:   beforePath,
			})
			continue
		}
		if beforePath == afterSnapshot.SectionPaths[index] {
			continue
		}
		issues = append(issues, RoundtripIssue{
			Code:     "section-path-changed",
			Severity: "error",
			Message:  fmt.Sprintf("section path changed at index %d", index),
			Before:   beforePath,
			After:    afterSnapshot.SectionPaths[index],
		})
	}
	for _, riskHint := range afterSnapshot.RiskHints {
		if slices.Contains(beforeSnapshot.RiskHints, riskHint) {
			continue
		}
		issues = append(issues, RoundtripIssue{
			Code:     "risk-added",
			Severity: "warning",
			Message:  fmt.Sprintf("roundtrip introduced risk hint %q", riskHint),
			After:    riskHint,
		})
	}
	for _, riskHint := range beforeSnapshot.RiskHints {
		if slices.Contains(afterSnapshot.RiskHints, riskHint) {
			continue
		}
		issues = append(issues, RoundtripIssue{
			Code:     "risk-removed",
			Severity: "warning",
			Message:  fmt.Sprintf("roundtrip removed risk hint %q", riskHint),
			Before:   riskHint,
		})
	}

	return issues
}

func roundtripLineCount(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hashStrings(values []string) string {
	return hashString(strings.Join(values, "\n"))
}

func paragraphSignatures(paragraphs []TemplateParagraph) []string {
	signatures := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		var builder strings.Builder
		builder.WriteString(strconv.Itoa(paragraph.SectionIndex))
		builder.WriteString("|")
		builder.WriteString(paragraph.SectionPath)
		builder.WriteString("|")
		builder.WriteString(strconv.Itoa(paragraph.ParagraphIndex))
		builder.WriteString("|")
		builder.WriteString(optionalIntSignature(paragraph.TableIndex))
		builder.WriteString("|")
		builder.WriteString(optionalCellSignature(paragraph.Cell))
		builder.WriteString("|")
		builder.WriteString(paragraph.StyleSummary)
		builder.WriteString("|")
		builder.WriteString(paragraph.Text)
		signatures = append(signatures, builder.String())
	}
	return signatures
}

func tableSignatures(tables []TemplateTable) []string {
	signatures := make([]string, 0, len(tables))
	for _, table := range tables {
		var builder strings.Builder
		builder.WriteString(strconv.Itoa(table.SectionIndex))
		builder.WriteString("|")
		builder.WriteString(table.SectionPath)
		builder.WriteString("|")
		builder.WriteString(strconv.Itoa(table.TableIndex))
		builder.WriteString("|")
		builder.WriteString(strconv.Itoa(table.Rows))
		builder.WriteString("|")
		builder.WriteString(strconv.Itoa(table.Cols))
		builder.WriteString("|")
		builder.WriteString(strconv.Itoa(table.MergedCellCount))
		builder.WriteString("|")
		builder.WriteString(table.LabelText)
		builder.WriteString("|")
		for _, cell := range table.Cells {
			builder.WriteString(strconv.Itoa(cell.Row))
			builder.WriteString(",")
			builder.WriteString(strconv.Itoa(cell.Col))
			builder.WriteString(",")
			builder.WriteString(strconv.Itoa(cell.RowSpan))
			builder.WriteString(",")
			builder.WriteString(strconv.Itoa(cell.ColSpan))
			builder.WriteString(",")
			builder.WriteString(cell.Text)
			builder.WriteString(";")
		}
		signatures = append(signatures, builder.String())
	}
	return signatures
}

func optionalIntSignature(value *int) string {
	if value == nil {
		return "-"
	}
	return strconv.Itoa(*value)
}

func optionalCellSignature(cell *AnalysisCell) string {
	if cell == nil {
		return "-"
	}
	return strconv.Itoa(cell.Row) + "," + strconv.Itoa(cell.Col)
}

func compareRoundtripParagraphs(before, after []TemplateParagraph) []RoundtripIssue {
	var issues []RoundtripIssue
	limit := max(len(before), len(after))
	for index := 0; index < limit; index++ {
		if index >= len(before) {
			afterParagraph := after[index]
			sectionIndex := afterParagraph.SectionIndex
			paragraphIndex := afterParagraph.ParagraphIndex
			issues = append(issues, RoundtripIssue{
				Code:           "paragraph-added",
				Severity:       "error",
				Message:        fmt.Sprintf("paragraph added at ordered index %d", index),
				SectionIndex:   &sectionIndex,
				SectionPath:    afterParagraph.SectionPath,
				ParagraphIndex: &paragraphIndex,
				TableIndex:     afterParagraph.TableIndex,
				Cell:           afterParagraph.Cell,
				After:          afterParagraph.Text,
			})
			continue
		}
		if index >= len(after) {
			beforeParagraph := before[index]
			sectionIndex := beforeParagraph.SectionIndex
			paragraphIndex := beforeParagraph.ParagraphIndex
			issues = append(issues, RoundtripIssue{
				Code:           "paragraph-removed",
				Severity:       "error",
				Message:        fmt.Sprintf("paragraph removed at ordered index %d", index),
				SectionIndex:   &sectionIndex,
				SectionPath:    beforeParagraph.SectionPath,
				ParagraphIndex: &paragraphIndex,
				TableIndex:     beforeParagraph.TableIndex,
				Cell:           beforeParagraph.Cell,
				Before:         beforeParagraph.Text,
			})
			continue
		}

		beforeParagraph := before[index]
		afterParagraph := after[index]
		if paragraphSignatures([]TemplateParagraph{beforeParagraph})[0] == paragraphSignatures([]TemplateParagraph{afterParagraph})[0] {
			continue
		}

		sectionIndex := beforeParagraph.SectionIndex
		paragraphIndex := beforeParagraph.ParagraphIndex
		issues = append(issues, RoundtripIssue{
			Code:           "paragraph-changed",
			Severity:       "error",
			Message:        fmt.Sprintf("paragraph changed at ordered index %d", index),
			SectionIndex:   &sectionIndex,
			SectionPath:    beforeParagraph.SectionPath,
			ParagraphIndex: &paragraphIndex,
			TableIndex:     beforeParagraph.TableIndex,
			Cell:           beforeParagraph.Cell,
			Before:         beforeParagraph.Text,
			After:          afterParagraph.Text,
		})
	}
	return issues
}

func compareRoundtripTables(before, after []TemplateTable) []RoundtripIssue {
	var issues []RoundtripIssue
	limit := max(len(before), len(after))
	for index := 0; index < limit; index++ {
		if index >= len(before) {
			afterTable := after[index]
			sectionIndex := afterTable.SectionIndex
			tableIndex := afterTable.TableIndex
			issues = append(issues, RoundtripIssue{
				Code:         "table-added",
				Severity:     "error",
				Message:      fmt.Sprintf("table added at ordered index %d", index),
				SectionIndex: &sectionIndex,
				SectionPath:  afterTable.SectionPath,
				TableIndex:   &tableIndex,
				After:        afterTable.TextPreview,
			})
			continue
		}
		if index >= len(after) {
			beforeTable := before[index]
			sectionIndex := beforeTable.SectionIndex
			tableIndex := beforeTable.TableIndex
			issues = append(issues, RoundtripIssue{
				Code:         "table-removed",
				Severity:     "error",
				Message:      fmt.Sprintf("table removed at ordered index %d", index),
				SectionIndex: &sectionIndex,
				SectionPath:  beforeTable.SectionPath,
				TableIndex:   &tableIndex,
				Before:       beforeTable.TextPreview,
			})
			continue
		}

		beforeTable := before[index]
		afterTable := after[index]
		if tableSignatures([]TemplateTable{beforeTable})[0] == tableSignatures([]TemplateTable{afterTable})[0] {
			continue
		}

		sectionIndex := beforeTable.SectionIndex
		tableIndex := beforeTable.TableIndex
		issues = append(issues, RoundtripIssue{
			Code:         "table-changed",
			Severity:     "error",
			Message:      fmt.Sprintf("table changed at ordered index %d", index),
			SectionIndex: &sectionIndex,
			SectionPath:  beforeTable.SectionPath,
			TableIndex:   &tableIndex,
			Before:       beforeTable.TextPreview,
			After:        afterTable.TextPreview,
		})
		issues = append(issues, compareRoundtripCells(beforeTable, afterTable)...)
	}
	return issues
}

func compareRoundtripCells(beforeTable, afterTable TemplateTable) []RoundtripIssue {
	var issues []RoundtripIssue
	beforeCells := beforeTable.Cells
	afterCells := afterTable.Cells
	limit := max(len(beforeCells), len(afterCells))
	for index := 0; index < limit; index++ {
		if index >= len(beforeCells) {
			afterCell := afterCells[index]
			sectionIndex := afterTable.SectionIndex
			tableIndex := afterTable.TableIndex
			cell := AnalysisCell{Row: afterCell.Row, Col: afterCell.Col}
			issues = append(issues, RoundtripIssue{
				Code:         "cell-added",
				Severity:     "error",
				Message:      fmt.Sprintf("cell added in table %d at ordered index %d", afterTable.TableIndex, index),
				SectionIndex: &sectionIndex,
				SectionPath:  afterTable.SectionPath,
				TableIndex:   &tableIndex,
				Cell:         &cell,
				After:        afterCell.Text,
			})
			continue
		}
		if index >= len(afterCells) {
			beforeCell := beforeCells[index]
			sectionIndex := beforeTable.SectionIndex
			tableIndex := beforeTable.TableIndex
			cell := AnalysisCell{Row: beforeCell.Row, Col: beforeCell.Col}
			issues = append(issues, RoundtripIssue{
				Code:         "cell-removed",
				Severity:     "error",
				Message:      fmt.Sprintf("cell removed in table %d at ordered index %d", beforeTable.TableIndex, index),
				SectionIndex: &sectionIndex,
				SectionPath:  beforeTable.SectionPath,
				TableIndex:   &tableIndex,
				Cell:         &cell,
				Before:       beforeCell.Text,
			})
			continue
		}

		beforeCell := beforeCells[index]
		afterCell := afterCells[index]
		if beforeCell == afterCell {
			continue
		}

		sectionIndex := beforeTable.SectionIndex
		tableIndex := beforeTable.TableIndex
		cell := AnalysisCell{Row: beforeCell.Row, Col: beforeCell.Col}
		issues = append(issues, RoundtripIssue{
			Code:         "cell-changed",
			Severity:     "error",
			Message:      fmt.Sprintf("cell changed in table %d at row=%d col=%d", beforeTable.TableIndex, beforeCell.Row, beforeCell.Col),
			SectionIndex: &sectionIndex,
			SectionPath:  beforeTable.SectionPath,
			TableIndex:   &tableIndex,
			Cell:         &cell,
			Before:       beforeCell.Text,
			After:        afterCell.Text,
		})
	}
	return issues
}
