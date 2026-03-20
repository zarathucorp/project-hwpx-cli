package core

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

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
	analysis.Anchors = collectTemplateAnchorCandidates(analysis)
	analysis.AnchorCount = len(analysis.Anchors)
	applyTemplateAnalysisAnchorSummary(&analysis)
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

	tablePlaceholderCounts := map[int]int{}
	for _, candidate := range placeholders {
		if candidate.TableIndex == nil {
			continue
		}
		tablePlaceholderCounts[*candidate.TableIndex]++
	}
	tableGuideCounts := map[int]int{}
	for _, candidate := range guides {
		if candidate.TableIndex == nil {
			continue
		}
		tableGuideCounts[*candidate.TableIndex]++
	}

	topLevelTableCount := 0
	nestedTableCount := 0
	for index := range tableResults {
		if tableResults[index].ParentTableIndex == nil {
			topLevelTableCount++
		} else {
			nestedTableCount++
		}
		tableResults[index].PlaceholderCount = tablePlaceholderCounts[tableResults[index].TableIndex]
		tableResults[index].GuideCount = tableGuideCounts[tableResults[index].TableIndex]
		tableResults[index].AnchorHints = collectTableAnchorHints(tableResults[index])
		tableResults[index].AnchorCount = len(tableResults[index].AnchorHints)
		tableResults[index].Role = inferTemplateTableRole(tableResults[index])
		tableResults[index].RoleHints = collectTableRoleHints(tableResults[index])
	}

	sectionAnalysis := TemplateSection{
		SectionIndex:       sectionIndex,
		SectionPath:        sectionPath,
		ParagraphCount:     paragraphCount,
		TableCount:         len(allTables),
		TopLevelTableCount: topLevelTableCount,
		NestedTableCount:   nestedTableCount,
		MergedCellCount:    countMergedCellsInSection(root),
		PlaceholderCount:   len(placeholders),
		GuideCount:         len(guides),
		HasHeader:          len(findElementsByTag(root, "hp:header")) > 0,
		HasFooter:          len(findElementsByTag(root, "hp:footer")) > 0,
		HasPageNumber:      len(findElementsByTag(root, "hp:pageNum")) > 0,
		TextPreview:        textPreview,
	}
	sectionAnalysis.Role = inferTemplateSectionRole(sectionAnalysis)
	sectionAnalysis.RoleHints = collectSectionRoleHints(sectionAnalysis)

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

func applyTemplateAnalysisAnchorSummary(analysis *TemplateAnalysis) {
	if analysis == nil || len(analysis.Anchors) == 0 {
		return
	}

	sectionAnchorCounts := map[string]int{}
	tableAnchorCounts := map[string]int{}
	for _, candidate := range analysis.Anchors {
		sectionAnchorCounts[candidate.SectionPath]++
		if candidate.TableIndex != nil {
			tableAnchorCounts[tableContextKey(candidate.SectionPath, *candidate.TableIndex)]++
		}
	}

	for index := range analysis.Sections {
		section := &analysis.Sections[index]
		section.AnchorCount = sectionAnchorCounts[section.SectionPath]
		section.Role = inferTemplateSectionRole(*section)
		section.RoleHints = collectSectionRoleHints(*section)
	}
	for index := range analysis.Tables {
		table := &analysis.Tables[index]
		table.AnchorCount = tableAnchorCounts[tableContextKey(table.SectionPath, table.TableIndex)]
		table.Role = inferTemplateTableRole(*table)
		table.RoleHints = collectTableRoleHints(*table)
	}
}

func collectTemplateAnchorCandidates(analysis TemplateAnalysis) []TemplateAnchorCandidate {
	candidates := make([]TemplateAnchorCandidate, 0, len(analysis.Tables)*4)
	seen := map[string]struct{}{}

	for _, table := range analysis.Tables {
		if label := strings.TrimSpace(table.LabelText); label != "" {
			appendTemplateAnchorCandidate(&candidates, seen, TemplateAnchorCandidate{
				Kind:         "table-label",
				Role:         "table-label",
				Score:        100,
				SectionIndex: table.SectionIndex,
				SectionPath:  table.SectionPath,
				TableIndex:   intPointer(table.TableIndex),
				TableLabel:   table.LabelText,
				Text:         label,
			})
		}

		for _, cell := range table.Cells {
			value := normalizeAnchorHintText(cell.Text)
			if !shouldKeepAnchorHint(value) {
				continue
			}
			role, score := classifyTableCellAnchorRole(cell)
			if role == "" {
				continue
			}
			appendTemplateAnchorCandidate(&candidates, seen, TemplateAnchorCandidate{
				Kind:         "table-cell",
				Role:         role,
				Score:        score,
				SectionIndex: table.SectionIndex,
				SectionPath:  table.SectionPath,
				TableIndex:   intPointer(table.TableIndex),
				Cell: &AnalysisCell{
					Row: cell.Row,
					Col: cell.Col,
				},
				TableLabel: table.LabelText,
				Text:       value,
			})
		}
	}

	for _, paragraph := range analysis.Paragraphs {
		if paragraph.TableIndex != nil {
			continue
		}
		value := normalizeAnchorHintText(paragraph.Text)
		if !shouldKeepAnchorHint(value) || !looksLikeParagraphAnchor(paragraph) {
			continue
		}
		role, score := classifyParagraphAnchorRole(paragraph)
		appendTemplateAnchorCandidate(&candidates, seen, TemplateAnchorCandidate{
			Kind:           "paragraph",
			Role:           role,
			Score:          score,
			SectionIndex:   paragraph.SectionIndex,
			SectionPath:    paragraph.SectionPath,
			ParagraphIndex: intPointer(paragraph.ParagraphIndex),
			StyleSummary:   paragraph.StyleSummary,
			Text:           value,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].SectionIndex != candidates[j].SectionIndex {
			return candidates[i].SectionIndex < candidates[j].SectionIndex
		}
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		if candidates[i].Kind != candidates[j].Kind {
			return candidates[i].Kind < candidates[j].Kind
		}
		return candidates[i].Text < candidates[j].Text
	})
	return candidates
}

func appendTemplateAnchorCandidate(target *[]TemplateAnchorCandidate, seen map[string]struct{}, candidate TemplateAnchorCandidate) {
	key := candidate.Kind + "|" + candidate.SectionPath + "|" + optionalIntKey(candidate.ParagraphIndex) + "|" + optionalIntKey(candidate.TableIndex) + "|" + optionalCellKey(candidate.Cell) + "|" + candidate.Text
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*target = append(*target, candidate)
}

func optionalIntKey(value *int) string {
	if value == nil {
		return "-"
	}
	return strconv.Itoa(*value)
}

func optionalCellKey(value *AnalysisCell) string {
	if value == nil {
		return "-"
	}
	return strconv.Itoa(value.Row) + "," + strconv.Itoa(value.Col)
}

func intPointer(value int) *int {
	resolved := value
	return &resolved
}

func inferTemplateSectionRole(section TemplateSection) string {
	switch {
	case section.PlaceholderCount > 0 || section.GuideCount > 0:
		return "template-form"
	case section.TopLevelTableCount >= 2 && section.ParagraphCount <= section.TopLevelTableCount*2+2:
		return "table-form"
	case section.TopLevelTableCount > 0 && section.ParagraphCount <= section.TopLevelTableCount+2:
		return "table-sheet"
	case section.TableCount == 0 && section.ParagraphCount >= 3:
		return "narrative"
	default:
		return "mixed"
	}
}

func inferTemplateTableRole(table TemplateTable) string {
	switch {
	case table.NestedDepth > 0:
		return "nested"
	case table.Rows <= 2 && table.Cols <= 2:
		return "key-value"
	case table.Rows >= 2 && table.Cols >= 2:
		return "matrix"
	default:
		return "single-axis"
	}
}

func collectSectionRoleHints(section TemplateSection) []string {
	hints := make([]string, 0, 5)
	if section.Role != "" {
		hints = append(hints, section.Role)
	}
	if section.PlaceholderCount > 0 {
		hints = append(hints, "contains-placeholders")
	}
	if section.GuideCount > 0 {
		hints = append(hints, "contains-guides")
	}
	if section.TopLevelTableCount > 0 {
		hints = append(hints, "contains-tables")
	}
	if section.NestedTableCount > 0 {
		hints = append(hints, "contains-nested-tables")
	}
	if section.HasHeader || section.HasFooter || section.HasPageNumber {
		hints = append(hints, "page-decorated")
	}
	return hints
}

func collectTableRoleHints(table TemplateTable) []string {
	hints := make([]string, 0, 5)
	if table.Role != "" {
		hints = append(hints, table.Role)
	}
	if strings.TrimSpace(table.LabelText) != "" {
		hints = append(hints, "labeled-table")
	}
	if table.PlaceholderCount > 0 {
		hints = append(hints, "contains-placeholders")
	}
	if table.GuideCount > 0 {
		hints = append(hints, "contains-guides")
	}
	if table.NestedDepth > 0 {
		hints = append(hints, "nested-table")
	}
	if len(table.AnchorHints) > 0 {
		hints = append(hints, "anchor-rich")
	}
	return hints
}

func looksLikeParagraphAnchor(paragraph TemplateParagraph) bool {
	text := normalizeAnchorHintText(paragraph.Text)
	if text == "" {
		return false
	}
	if utf8RuneCount(text) > 24 {
		return false
	}
	if strings.Contains(text, " ") && utf8RuneCount(text) > 18 {
		return false
	}
	return true
}

func classifyParagraphAnchorRole(paragraph TemplateParagraph) (string, int) {
	lowerStyle := strings.ToLower(strings.TrimSpace(paragraph.StyleSummary))
	if strings.Contains(lowerStyle, "heading") || strings.Contains(lowerStyle, "title") {
		return "section-heading", 80
	}
	return "paragraph-label", 60
}

func classifyTableCellAnchorRole(cell TemplateCell) (string, int) {
	switch {
	case cell.Row == 0 && cell.Col == 0:
		return "header-cell", 95
	case cell.Row == 0:
		return "column-label", 90
	case cell.Col == 0:
		return "row-label", 88
	default:
		return "", 0
	}
}

func collectTableAnchorHints(table TemplateTable) []string {
	preferred := []string{}
	fallback := []string{}
	seen := map[string]struct{}{}

	for _, cell := range table.Cells {
		value := normalizeAnchorHintText(cell.Text)
		if !shouldKeepAnchorHint(value) {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		if cell.Row == 0 || cell.Col == 0 {
			preferred = append(preferred, value)
			continue
		}
		fallback = append(fallback, value)
	}

	values := append(preferred, fallback...)
	if len(values) > 8 {
		values = values[:8]
	}
	return values
}

func normalizeAnchorHintText(value string) string {
	normalized := strings.ToValidUTF8(strings.TrimSpace(value), "")
	if normalized == "" {
		return ""
	}
	return strings.Join(strings.Fields(normalized), " ")
}

func shouldKeepAnchorHint(value string) bool {
	if value == "" {
		return false
	}
	if utf8RuneCount(value) > 32 {
		return false
	}
	if _, ok := detectPlaceholderReason(value); ok {
		return false
	}
	if _, ok := detectGuideReason(value); ok {
		return false
	}
	hasLetter := false
	for _, r := range value {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	return hasLetter
}

func utf8RuneCount(value string) int {
	count := 0
	for range value {
		count++
	}
	return count
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
