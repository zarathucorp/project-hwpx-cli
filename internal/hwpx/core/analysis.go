package core

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

var placeholderPattern = regexp.MustCompile(`\{\{[^{}]+\}\}|_{3,}|□{1,}|▢{1,}`)

func AnalyzeTemplate(targetPath string) (TemplateAnalysis, error) {
	entries, err := readEntriesFromTarget(targetPath)
	if err != nil {
		return TemplateAnalysis{}, err
	}
	return analyzeTemplateEntries(entries)
}

func readEntriesFromTarget(targetPath string) (map[string][]byte, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return readEntriesFromDir(targetPath)
	}
	return readEntriesFromArchive(targetPath)
}

func analyzeTemplateEntries(entries map[string][]byte) (TemplateAnalysis, error) {
	report, err := inspectEntries(entries)
	if err != nil {
		return TemplateAnalysis{}, err
	}
	if !report.Valid {
		return TemplateAnalysis{}, fmt.Errorf("cannot analyze invalid HWPX package: %s", strings.Join(report.Errors, "; "))
	}

	analysis := TemplateAnalysis{
		Sections:     []TemplateSection{},
		Tables:       []TemplateTable{},
		Paragraphs:   []TemplateParagraph{},
		Placeholders: []TemplateTextCandidate{},
		Guides:       []TemplateTextCandidate{},
	}

	styleByID := map[string]exportStyleRef{}
	if header := entries["Contents/header.xml"]; len(header) > 0 {
		parsedStyles, err := parseStyles(header)
		if err != nil {
			return TemplateAnalysis{}, fmt.Errorf("parse styles: %w", err)
		}
		styleByID = parsedStyles
	}

	for sectionIndex, sectionPath := range report.Summary.SectionPath {
		sectionAnalysis, tableResults, paragraphResults, placeholders, guides, err := analyzeSection(sectionIndex, sectionPath, entries[sectionPath], styleByID)
		if err != nil {
			return TemplateAnalysis{}, fmt.Errorf("analyze section %s: %w", sectionPath, err)
		}
		analysis.Sections = append(analysis.Sections, sectionAnalysis)
		analysis.Tables = append(analysis.Tables, tableResults...)
		analysis.Paragraphs = append(analysis.Paragraphs, paragraphResults...)
		analysis.Placeholders = append(analysis.Placeholders, placeholders...)
		analysis.Guides = append(analysis.Guides, guides...)
	}

	analysis.SectionCount = len(analysis.Sections)
	analysis.TableCount = len(analysis.Tables)
	analysis.ParagraphCount = len(analysis.Paragraphs)
	analysis.PlaceholderCount = len(analysis.Placeholders)
	analysis.GuideCount = len(analysis.Guides)
	analysis.Fingerprint = buildTemplateFingerprint(report, analysis)
	return analysis, nil
}

func analyzeSection(sectionIndex int, sectionPath string, content []byte, styleByID map[string]exportStyleRef) (TemplateSection, []TemplateTable, []TemplateParagraph, []TemplateTextCandidate, []TemplateTextCandidate, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(content); err != nil {
		return TemplateSection{}, nil, nil, nil, nil, err
	}

	root := doc.Root()
	if root == nil {
		return TemplateSection{}, nil, nil, nil, nil, fmt.Errorf("section xml has no root")
	}

	allTables := findElementsByTag(root, "hp:tbl")
	tableIndexByElement := map[*etree.Element]int{}
	tableLabelByElement := deriveTableLabels(root)
	tableResults := make([]TemplateTable, 0, len(allTables))
	for tableIndex, table := range allTables {
		tableIndexByElement[table] = tableIndex
		parentTableIndex, nestedDepth := locateTableHierarchy(table, tableIndexByElement)
		tableResults = append(tableResults, TemplateTable{
			SectionIndex:     sectionIndex,
			SectionPath:      sectionPath,
			TableIndex:       tableIndex,
			ParentTableIndex: parentTableIndex,
			NestedDepth:      nestedDepth,
			Rows:             parseIntOrDefault(table.SelectAttrValue("rowCnt", ""), len(childElementsByTag(table, "hp:tr"))),
			Cols:             parseIntOrDefault(table.SelectAttrValue("colCnt", ""), 0),
			MergedCellCount:  countMergedCellsInTable(table),
			ParagraphCount:   len(findElementsByTag(table, "hp:p")),
			LabelText:        truncateText(strings.TrimSpace(tableLabelByElement[table]), 120),
			TextPreview:      truncateText(strings.TrimSpace(analyzeElementPlainText(table)), 120),
			Cells:            analyzeTableCells(table),
		})
	}

	bodyParagraphs := childElementsByTag(root, "hp:p")
	paragraphCount := 0
	textPreview := ""
	for _, paragraph := range bodyParagraphs {
		if paragraph.FindElement(".//hp:secPr") != nil {
			continue
		}
		paragraphCount++
		if textPreview == "" {
			textPreview = truncateText(strings.TrimSpace(analyzeElementPlainText(paragraph)), 120)
		}
	}

	paragraphResults := []TemplateParagraph{}
	placeholders := []TemplateTextCandidate{}
	guides := []TemplateTextCandidate{}
	allParagraphs := findElementsByTag(root, "hp:p")
	for paragraphIndex, paragraph := range allParagraphs {
		text := strings.TrimSpace(analyzeElementPlainText(paragraph))
		if text == "" {
			continue
		}

		tableIndex, cell := locateParagraphContext(paragraph, tableIndexByElement)
		styleIDRef := strings.TrimSpace(paragraph.SelectAttrValue("styleIDRef", ""))
		styleName, styleSummary := resolveParagraphStyle(styleByID, styleIDRef)
		paragraphResults = append(paragraphResults, TemplateParagraph{
			SectionIndex:   sectionIndex,
			SectionPath:    sectionPath,
			ParagraphIndex: paragraphIndex,
			TableIndex:     tableIndex,
			Cell:           cell,
			StyleIDRef:     styleIDRef,
			StyleName:      styleName,
			StyleSummary:   styleSummary,
			Text:           text,
		})
		if reason, ok := detectPlaceholderReason(text); ok {
			placeholders = append(placeholders, TemplateTextCandidate{
				SectionIndex:   sectionIndex,
				SectionPath:    sectionPath,
				ParagraphIndex: paragraphIndex,
				TableIndex:     tableIndex,
				Cell:           cell,
				StyleSummary:   styleSummary,
				Text:           text,
				Reason:         reason,
			})
		}
		if reason, ok := detectGuideReason(text); ok {
			guides = append(guides, TemplateTextCandidate{
				SectionIndex:   sectionIndex,
				SectionPath:    sectionPath,
				ParagraphIndex: paragraphIndex,
				TableIndex:     tableIndex,
				Cell:           cell,
				StyleSummary:   styleSummary,
				Text:           text,
				Reason:         reason,
			})
		}
	}

	sectionAnalysis := TemplateSection{
		SectionIndex:    sectionIndex,
		SectionPath:     sectionPath,
		ParagraphCount:  paragraphCount,
		TableCount:      len(allTables),
		MergedCellCount: countMergedCellsInSection(root),
		HasHeader:       len(findElementsByTag(root, "hp:header")) > 0,
		HasFooter:       len(findElementsByTag(root, "hp:footer")) > 0,
		HasPageNumber:   len(findElementsByTag(root, "hp:pageNum")) > 0,
		TextPreview:     textPreview,
	}

	return sectionAnalysis, tableResults, paragraphResults, placeholders, guides, nil
}

func locateParagraphContext(paragraph *etree.Element, tableIndexByElement map[*etree.Element]int) (*int, *AnalysisCell) {
	for current := paragraph.Parent(); current != nil; current = current.Parent() {
		if tagMatches(current.Tag, "hp:tc") {
			cell := &AnalysisCell{}
			if cellAddr := firstChildByTag(current, "hp:cellAddr"); cellAddr != nil {
				cell.Row = parseIntOrDefault(cellAddr.SelectAttrValue("rowAddr", ""), 0)
				cell.Col = parseIntOrDefault(cellAddr.SelectAttrValue("colAddr", ""), 0)
			}
			for ancestor := current.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
				if !tagMatches(ancestor.Tag, "hp:tbl") {
					continue
				}
				if index, ok := tableIndexByElement[ancestor]; ok {
					tableIndex := index
					return &tableIndex, cell
				}
			}
			return nil, cell
		}
		if tagMatches(current.Tag, "hp:tbl") {
			if index, ok := tableIndexByElement[current]; ok {
				tableIndex := index
				return &tableIndex, nil
			}
		}
	}
	return nil, nil
}

func detectPlaceholderReason(text string) (string, bool) {
	switch {
	case strings.Contains(text, "{{") && strings.Contains(text, "}}"):
		return "mustache-pattern", true
	case placeholderPattern.MatchString(text):
		return "placeholder-pattern", true
	default:
		return "", false
	}
}

func detectGuideReason(text string) (string, bool) {
	normalized := strings.TrimSpace(text)
	lower := strings.ToLower(normalized)

	switch {
	case strings.HasPrefix(normalized, "※") && (strings.Contains(lower, "작성") || strings.Contains(lower, "기재") || strings.Contains(lower, "참고")):
		return "guide-marker", true
	case strings.Contains(lower, "작성요령"), strings.Contains(lower, "작성 요령"):
		return "guide-text", true
	case strings.Contains(lower, "작성예시"), strings.Contains(lower, "작성 예시"):
		return "guide-text", true
	case strings.Contains(lower, "기재요령"), strings.Contains(lower, "기재 요령"):
		return "guide-text", true
	case strings.Contains(lower, "유의사항"):
		return "guide-text", true
	default:
		return "", false
	}
}

func buildTemplateFingerprint(report Report, analysis TemplateAnalysis) TemplateFingerprint {
	placeholderTexts := collectTemplateFingerprintPlaceholderTexts(analysis.Placeholders)
	return TemplateFingerprint{
		SectionCount:      len(report.Summary.SectionPath),
		SectionPaths:      append([]string{}, report.Summary.SectionPath...),
		TableLabels:       collectTemplateFingerprintTableLabels(analysis.Tables),
		PlaceholderTexts:  placeholderTexts,
		PlaceholderDigest: hashStrings(placeholderTexts),
	}
}

func collectTemplateFingerprintTableLabels(tables []TemplateTable) []string {
	values := make([]string, 0, len(tables))
	seen := map[string]struct{}{}
	for _, table := range tables {
		label := strings.TrimSpace(table.LabelText)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		values = append(values, label)
	}
	sort.Strings(values)
	return values
}

func collectTemplateFingerprintPlaceholderTexts(candidates []TemplateTextCandidate) []string {
	values := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		text := strings.TrimSpace(candidate.Text)
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		values = append(values, text)
	}
	sort.Strings(values)
	return values
}

func deriveTableLabels(root *etree.Element) map[*etree.Element]string {
	labels := map[*etree.Element]string{}
	lastText := ""

	for _, paragraph := range findElementsByTag(root, "hp:p") {
		if paragraphHasSectionProperty(paragraph) {
			continue
		}

		directText := strings.TrimSpace(paragraphDirectText(paragraph))
		labelText := directText
		if labelText == "" {
			labelText = lastText
		}

		for _, table := range paragraphTables(paragraph) {
			if strings.TrimSpace(labelText) == "" {
				continue
			}
			labels[table] = labelText
		}

		if directText != "" {
			lastText = directText
		}
	}

	return labels
}

func locateTableHierarchy(table *etree.Element, tableIndexByElement map[*etree.Element]int) (*int, int) {
	nestedDepth := 0
	for ancestor := table.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
		if !tagMatches(ancestor.Tag, "hp:tbl") {
			continue
		}
		nestedDepth++
		if index, ok := tableIndexByElement[ancestor]; ok {
			parentTableIndex := index
			return &parentTableIndex, nestedDepth
		}
	}
	return nil, 0
}

func paragraphTables(paragraph *etree.Element) []*etree.Element {
	var tables []*etree.Element
	for _, run := range childElementsByTag(paragraph, "hp:run") {
		for _, child := range run.ChildElements() {
			if tagMatches(child.Tag, "hp:tbl") {
				tables = append(tables, child)
			}
		}
	}
	return tables
}

func analyzeTableCells(table *etree.Element) []TemplateCell {
	rows := childElementsByTag(table, "hp:tr")
	cells := make([]TemplateCell, 0, len(rows)*2)
	for _, row := range rows {
		for _, cell := range childElementsByTag(row, "hp:tc") {
			summary := TemplateCell{
				ParagraphCount: len(findElementsByTag(cell, "hp:p")),
				Text:           truncateText(strings.TrimSpace(analyzeElementPlainText(cell)), 120),
			}
			if addr := firstChildByTag(cell, "hp:cellAddr"); addr != nil {
				summary.Row = parseIntOrDefault(addr.SelectAttrValue("rowAddr", ""), 0)
				summary.Col = parseIntOrDefault(addr.SelectAttrValue("colAddr", ""), 0)
			}
			if span := firstChildByTag(cell, "hp:cellSpan"); span != nil {
				summary.RowSpan = parseIntOrDefault(span.SelectAttrValue("rowSpan", ""), 1)
				summary.ColSpan = parseIntOrDefault(span.SelectAttrValue("colSpan", ""), 1)
			}
			cells = append(cells, summary)
		}
	}
	return cells
}

func resolveParagraphStyle(styleByID map[string]exportStyleRef, styleID string) (string, string) {
	if strings.TrimSpace(styleID) == "" {
		return "", ""
	}

	styleName := strings.TrimSpace(styleByID[styleID].Name)
	if styleName == "" {
		return "", "style#" + styleID
	}
	return styleName, fmt.Sprintf("%s (%s)", styleName, styleID)
}

func countMergedCellsInSection(root *etree.Element) int {
	return countMergedCellSpans(findElementsByTag(root, "hp:cellSpan"))
}

func countMergedCellsInTable(table *etree.Element) int {
	return countMergedCellSpans(findElementsByTag(table, "hp:cellSpan"))
}

func countMergedCellSpans(spans []*etree.Element) int {
	count := 0
	for _, span := range spans {
		rowSpan := parseIntOrDefault(span.SelectAttrValue("rowSpan", ""), 1)
		colSpan := parseIntOrDefault(span.SelectAttrValue("colSpan", ""), 1)
		if rowSpan > 1 || colSpan > 1 {
			count++
		}
	}
	return count
}

func analyzeElementPlainText(element *etree.Element) string {
	if element == nil {
		return ""
	}

	var builder strings.Builder
	var walk func(current *etree.Element)
	walk = func(current *etree.Element) {
		if tagMatches(current.Tag, "hp:t") {
			builder.WriteString(current.Text())
		}
		for _, child := range current.ChildElements() {
			walk(child)
		}
	}
	walk(element)
	return builder.String()
}

func parseIntOrDefault(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func truncateText(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit]) + "..."
}
