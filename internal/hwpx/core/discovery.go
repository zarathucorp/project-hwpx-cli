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

	if value := normalizeTargetQuery(query.Anchor); value != "" {
		addAnchorMatches(&matches, seen, analysis, value, "anchor")
	}
	if value := normalizeTargetQuery(query.NearText); value != "" {
		addAnchorMatches(&matches, seen, analysis, value, "near-text")
	}
	if value := normalizeTargetQuery(query.TableLabel); value != "" {
		addTableLabelMatches(&matches, seen, analysis, value)
	}
	if value := normalizeTargetQuery(query.Placeholder); value != "" {
		addPlaceholderMatches(&matches, seen, analysis, value)
	}

	return matches, nil
}

func normalizeTargetQuery(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func addAnchorMatches(matches *[]TemplateTargetMatch, seen map[string]struct{}, analysis TemplateAnalysis, query string, queryType string) {
	for _, paragraph := range analysis.Paragraphs {
		if paragraph.TableIndex != nil {
			continue
		}
		if !containsNormalized(paragraph.Text, query) {
			continue
		}
		paragraphIndex := paragraph.ParagraphIndex
		appendTargetMatch(matches, seen, queryType, TemplateTargetMatch{
			Kind:           "paragraph",
			QueryType:      queryType,
			SectionIndex:   paragraph.SectionIndex,
			SectionPath:    paragraph.SectionPath,
			ParagraphIndex: &paragraphIndex,
			StyleSummary:   paragraph.StyleSummary,
			Text:           paragraph.Text,
		})
	}

	for _, table := range analysis.Tables {
		if containsNormalized(table.LabelText, query) {
			tableIndex := table.TableIndex
			appendTargetMatch(matches, seen, queryType, TemplateTargetMatch{
				Kind:         "table",
				QueryType:    queryType,
				SectionIndex: table.SectionIndex,
				SectionPath:  table.SectionPath,
				TableIndex:   &tableIndex,
				LabelText:    table.LabelText,
				Text:         table.TextPreview,
			})
		}
		for _, cell := range table.Cells {
			if !containsNormalized(cell.Text, query) {
				continue
			}
			tableIndex := table.TableIndex
			appendTargetMatch(matches, seen, queryType, TemplateTargetMatch{
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
			})
		}
	}
}

func addTableLabelMatches(matches *[]TemplateTargetMatch, seen map[string]struct{}, analysis TemplateAnalysis, query string) {
	for _, table := range analysis.Tables {
		if !containsNormalized(table.LabelText, query) && !containsNormalized(table.TextPreview, query) {
			continue
		}
		tableIndex := table.TableIndex
		appendTargetMatch(matches, seen, "table-label", TemplateTargetMatch{
			Kind:         "table",
			QueryType:    "table-label",
			SectionIndex: table.SectionIndex,
			SectionPath:  table.SectionPath,
			TableIndex:   &tableIndex,
			LabelText:    table.LabelText,
			Text:         table.TextPreview,
		})
	}
}

func addPlaceholderMatches(matches *[]TemplateTargetMatch, seen map[string]struct{}, analysis TemplateAnalysis, query string) {
	for _, candidate := range analysis.Placeholders {
		if !containsNormalized(candidate.Text, query) {
			continue
		}
		paragraphIndex := candidate.ParagraphIndex
		appendTargetMatch(matches, seen, "placeholder", TemplateTargetMatch{
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
		})
	}
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
