package core

import (
	"strconv"
	"strings"
)

func FindTargets(targetPath string, query TargetQuery) ([]TemplateTargetMatch, error) {
	analysis, err := AnalyzeTemplate(targetPath)
	if err != nil {
		return nil, err
	}

	var matches []TemplateTargetMatch
	seen := map[string]struct{}{}
	contextIndex := buildTargetContextIndex(analysis)

	if value := normalizeTargetQuery(query.Anchor); value != "" {
		addAnchorMatches(&matches, seen, contextIndex, analysis, value, "anchor")
	}
	if value := normalizeTargetQuery(query.NearText); value != "" {
		addAnchorMatches(&matches, seen, contextIndex, analysis, value, "near-text")
	}
	if value := normalizeTargetQuery(query.TableLabel); value != "" {
		addTableLabelMatches(&matches, seen, contextIndex, analysis, value)
	}
	if value := normalizeTargetQuery(query.Placeholder); value != "" {
		addPlaceholderMatches(&matches, seen, contextIndex, analysis, value)
	}

	return matches, nil
}

type targetContextIndex struct {
	sections      map[string]TemplateSection
	tables        map[string]TemplateTable
	paragraphs    map[string]TemplateParagraph
	cellParagraph map[string][]TemplateParagraph
}

func normalizeTargetQuery(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func buildTargetContextIndex(analysis TemplateAnalysis) targetContextIndex {
	index := targetContextIndex{
		sections:      map[string]TemplateSection{},
		tables:        map[string]TemplateTable{},
		paragraphs:    map[string]TemplateParagraph{},
		cellParagraph: map[string][]TemplateParagraph{},
	}
	for _, section := range analysis.Sections {
		index.sections[section.SectionPath] = section
	}
	for _, table := range analysis.Tables {
		index.tables[tableContextKey(table.SectionPath, table.TableIndex)] = table
	}
	for _, paragraph := range analysis.Paragraphs {
		index.paragraphs[paragraphContextKey(paragraph.SectionPath, paragraph.ParagraphIndex)] = paragraph
		if paragraph.TableIndex != nil && paragraph.Cell != nil {
			key := cellContextKey(paragraph.SectionPath, *paragraph.TableIndex, paragraph.Cell.Row, paragraph.Cell.Col)
			index.cellParagraph[key] = append(index.cellParagraph[key], paragraph)
		}
	}
	return index
}

func addAnchorMatches(matches *[]TemplateTargetMatch, seen map[string]struct{}, contextIndex targetContextIndex, analysis TemplateAnalysis, query string, queryType string) {
	for _, paragraph := range analysis.Paragraphs {
		if paragraph.TableIndex != nil {
			continue
		}
		if !containsNormalized(paragraph.Text, query) {
			continue
		}
		paragraphIndex := paragraph.ParagraphIndex
		appendTargetMatch(matches, seen, queryType, withTargetContext(contextIndex, query, TemplateTargetMatch{
			Kind:           "paragraph",
			QueryType:      queryType,
			SectionIndex:   paragraph.SectionIndex,
			SectionPath:    paragraph.SectionPath,
			ParagraphIndex: &paragraphIndex,
			StyleSummary:   paragraph.StyleSummary,
			Text:           paragraph.Text,
		}))
	}

	for _, table := range analysis.Tables {
		if containsNormalized(table.LabelText, query) {
			tableIndex := table.TableIndex
			appendTargetMatch(matches, seen, queryType, withTargetContext(contextIndex, query, TemplateTargetMatch{
				Kind:         "table",
				QueryType:    queryType,
				SectionIndex: table.SectionIndex,
				SectionPath:  table.SectionPath,
				TableIndex:   &tableIndex,
				LabelText:    table.LabelText,
				Text:         table.TextPreview,
			}))
		}
		for _, cell := range table.Cells {
			if !containsNormalized(cell.Text, query) {
				continue
			}
			tableIndex := table.TableIndex
			appendTargetMatch(matches, seen, queryType, withTargetContext(contextIndex, query, TemplateTargetMatch{
				Kind:         "cell",
				QueryType:    queryType,
				SectionIndex: table.SectionIndex,
				SectionPath:  table.SectionPath,
				TableIndex:   &tableIndex,
				Cell: &AnalysisCell{
					Row: cell.Row,
					Col: cell.Col,
				},
				LabelText: table.LabelText,
				Text:      cell.Text,
				RowSpan:   cell.RowSpan,
				ColSpan:   cell.ColSpan,
			}))
		}
	}
}

func addTableLabelMatches(matches *[]TemplateTargetMatch, seen map[string]struct{}, contextIndex targetContextIndex, analysis TemplateAnalysis, query string) {
	for _, table := range analysis.Tables {
		if !containsNormalized(table.LabelText, query) && !containsNormalized(table.TextPreview, query) {
			continue
		}
		tableIndex := table.TableIndex
		appendTargetMatch(matches, seen, "table-label", withTargetContext(contextIndex, query, TemplateTargetMatch{
			Kind:         "table",
			QueryType:    "table-label",
			SectionIndex: table.SectionIndex,
			SectionPath:  table.SectionPath,
			TableIndex:   &tableIndex,
			LabelText:    table.LabelText,
			Text:         table.TextPreview,
		}))
	}
}

func addPlaceholderMatches(matches *[]TemplateTargetMatch, seen map[string]struct{}, contextIndex targetContextIndex, analysis TemplateAnalysis, query string) {
	for _, candidate := range analysis.Placeholders {
		if !containsNormalized(candidate.Text, query) {
			continue
		}
		paragraphIndex := candidate.ParagraphIndex
		appendTargetMatch(matches, seen, "placeholder", withTargetContext(contextIndex, query, TemplateTargetMatch{
			Kind:           "placeholder",
			QueryType:      "placeholder",
			SectionIndex:   candidate.SectionIndex,
			SectionPath:    candidate.SectionPath,
			ParagraphIndex: &paragraphIndex,
			TableIndex:     candidate.TableIndex,
			Cell:           candidate.Cell,
			StyleSummary:   candidate.StyleSummary,
			Text:           candidate.Text,
			Reason:         candidate.Reason,
		}))
	}
}

func withTargetContext(contextIndex targetContextIndex, query string, match TemplateTargetMatch) TemplateTargetMatch {
	context := &TemplateTargetContext{}
	if section, ok := contextIndex.sections[match.SectionPath]; ok {
		context.Section = &TemplateTargetSectionContext{
			ParagraphCount:   section.ParagraphCount,
			TableCount:       section.TableCount,
			MergedCellCount:  section.MergedCellCount,
			PlaceholderCount: section.PlaceholderCount,
			GuideCount:       section.GuideCount,
			AnchorCount:      section.AnchorCount,
			HasHeader:        section.HasHeader,
			HasFooter:        section.HasFooter,
			HasPageNumber:    section.HasPageNumber,
			TextPreview:      section.TextPreview,
			Role:             section.Role,
			RoleHints:        append([]string{}, section.RoleHints...),
		}
	}
	if match.TableIndex != nil {
		if table, ok := contextIndex.tables[tableContextKey(match.SectionPath, *match.TableIndex)]; ok {
			context.Table = &TemplateTargetTableContext{
				Rows:             table.Rows,
				Cols:             table.Cols,
				MergedCellCount:  table.MergedCellCount,
				ParagraphCount:   table.ParagraphCount,
				PlaceholderCount: table.PlaceholderCount,
				GuideCount:       table.GuideCount,
				AnchorCount:      table.AnchorCount,
				NestedDepth:      table.NestedDepth,
				LabelText:        table.LabelText,
				TextPreview:      table.TextPreview,
				AnchorHints:      append([]string{}, table.AnchorHints...),
				Role:             table.Role,
				RoleHints:        append([]string{}, table.RoleHints...),
			}
		}
	}
	if paragraph := resolveTargetParagraphContext(contextIndex, match, query); paragraph != nil {
		context.Paragraph = &TemplateTargetParagraphContext{
			StyleSummary: paragraph.StyleSummary,
			TextPreview:  truncateText(strings.TrimSpace(paragraph.Text), 120),
		}
	}
	if context.Section == nil && context.Table == nil && context.Paragraph == nil {
		return match
	}
	match.Context = context
	return match
}

func resolveTargetParagraphContext(contextIndex targetContextIndex, match TemplateTargetMatch, query string) *TemplateParagraph {
	if match.ParagraphIndex != nil {
		if paragraph, ok := contextIndex.paragraphs[paragraphContextKey(match.SectionPath, *match.ParagraphIndex)]; ok {
			return &paragraph
		}
	}
	if match.TableIndex == nil || match.Cell == nil {
		return nil
	}
	paragraphs := contextIndex.cellParagraph[cellContextKey(match.SectionPath, *match.TableIndex, match.Cell.Row, match.Cell.Col)]
	if len(paragraphs) == 0 {
		return nil
	}
	for _, paragraph := range paragraphs {
		if containsNormalized(paragraph.Text, query) {
			return &paragraph
		}
	}
	return &paragraphs[0]
}

func paragraphContextKey(sectionPath string, paragraphIndex int) string {
	return sectionPath + "|p:" + strconv.Itoa(paragraphIndex)
}

func tableContextKey(sectionPath string, tableIndex int) string {
	return sectionPath + "|t:" + strconv.Itoa(tableIndex)
}

func cellContextKey(sectionPath string, tableIndex int, row int, col int) string {
	return sectionPath + "|t:" + strconv.Itoa(tableIndex) + "|c:" + strconv.Itoa(row) + "," + strconv.Itoa(col)
}

func containsNormalized(value string, query string) bool {
	if query == "" {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(value)), query)
}

func appendTargetMatch(matches *[]TemplateTargetMatch, seen map[string]struct{}, queryType string, match TemplateTargetMatch) {
	key := targetMatchKey(queryType, match)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*matches = append(*matches, match)
}

func targetMatchKey(queryType string, match TemplateTargetMatch) string {
	var builder strings.Builder
	builder.WriteString(queryType)
	builder.WriteString("|")
	builder.WriteString(match.Kind)
	builder.WriteString("|")
	builder.WriteString(match.SectionPath)
	builder.WriteString("|")
	if match.ParagraphIndex != nil {
		builder.WriteString("p:")
		builder.WriteString(strconv.Itoa(*match.ParagraphIndex))
	}
	builder.WriteString("|")
	if match.TableIndex != nil {
		builder.WriteString("t:")
		builder.WriteString(strconv.Itoa(*match.TableIndex))
	}
	builder.WriteString("|")
	if match.Cell != nil {
		builder.WriteString("c:")
		builder.WriteString(strconv.Itoa(match.Cell.Row))
		builder.WriteString(",")
		builder.WriteString(strconv.Itoa(match.Cell.Col))
	}
	builder.WriteString("|")
	builder.WriteString(match.Reason)
	return builder.String()
}
