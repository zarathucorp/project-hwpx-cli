package hwpx

import (
	"fmt"
	"strings"
)

type FillTemplateResolutionReport struct {
	InputKind     string                        `json:"inputKind"`
	EntryCount    int                           `json:"entryCount"`
	ResolvedCount int                           `json:"resolvedCount"`
	SkippedCount  int                           `json:"skippedCount,omitempty"`
	Entries       []FillTemplateResolutionEntry `json:"entries"`
}

type FillTemplateResolutionEntry struct {
	Index         int      `json:"index"`
	Source        string   `json:"source"`
	Key           string   `json:"key,omitempty"`
	Note          string   `json:"note,omitempty"`
	PayloadPath   string   `json:"payloadPath,omitempty"`
	SelectorType  string   `json:"selectorType"`
	Selector      string   `json:"selector"`
	TableLabel    string   `json:"tableLabel,omitempty"`
	TableIndex    *int     `json:"tableIndex,omitempty"`
	Occurrence    *int     `json:"occurrence,omitempty"`
	MatchMode     string   `json:"matchMode,omitempty"`
	Mode          string   `json:"mode"`
	ValueKind     string   `json:"valueKind,omitempty"`
	ValuePreview  string   `json:"valuePreview,omitempty"`
	ValueCount    int      `json:"valueCount,omitempty"`
	RecordCount   int      `json:"recordCount,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	Required      bool     `json:"required,omitempty"`
	Unique        bool     `json:"unique,omitempty"`
	Expand        bool     `json:"expand,omitempty"`
	UsedFallback  bool     `json:"usedFallback,omitempty"`
	Skipped       bool     `json:"skipped,omitempty"`
	SkipReason    string   `json:"skipReason,omitempty"`
	ChangeCount   int      `json:"changeCount,omitempty"`
	MissCount     int      `json:"missCount,omitempty"`
	ChangeIndexes []int    `json:"changeIndexes,omitempty"`
	MissIndexes   []int    `json:"missIndexes,omitempty"`
}

func BuildMappingFillTemplateResolutionReport(replacements []FillTemplateReplacement) FillTemplateResolutionReport {
	report := FillTemplateResolutionReport{
		InputKind:     "mapping",
		EntryCount:    len(replacements),
		ResolvedCount: len(replacements),
		Entries:       make([]FillTemplateResolutionEntry, 0, len(replacements)),
	}
	for index, replacement := range replacements {
		entry := buildFillTemplateResolutionEntry("mapping", replacement)
		entry.Index = index
		report.Entries = append(report.Entries, entry)
	}
	return report
}

func buildFillTemplateResolutionEntry(source string, replacement FillTemplateReplacement) FillTemplateResolutionEntry {
	valueKind, valuePreview, valueCount, recordCount := summarizeFillTemplateReplacementValue(replacement)
	selectorType, selector := summarizeFillTemplateReplacementSelector(replacement)
	entry := FillTemplateResolutionEntry{
		Source:       source,
		Key:          strings.TrimSpace(replacement.Key),
		Note:         strings.TrimSpace(replacement.Note),
		SelectorType: selectorType,
		Selector:     selector,
		TableLabel:   strings.TrimSpace(replacement.TableLabel),
		TableIndex:   replacement.TableIndex,
		Occurrence:   replacement.Occurrence,
		MatchMode:    strings.ToLower(strings.TrimSpace(replacement.MatchMode)),
		Mode:         resolveFillTemplateResolutionMode(replacement),
		ValueKind:    valueKind,
		ValuePreview: valuePreview,
		ValueCount:   valueCount,
		RecordCount:  recordCount,
		Fields:       append([]string{}, replacement.Fields...),
		Required:     replacement.Required,
		Unique:       replacement.Unique,
		Expand:       replacement.Expand,
	}
	return entry
}

func summarizeFillTemplateReplacementSelector(replacement FillTemplateReplacement) (string, string) {
	switch {
	case strings.TrimSpace(replacement.Placeholder) != "":
		return "placeholder", replacement.Placeholder
	case strings.TrimSpace(replacement.NearText) != "":
		return "near-text", replacement.NearText
	default:
		return "anchor", replacement.Anchor
	}
}

func summarizeFillTemplateReplacementValue(replacement FillTemplateReplacement) (string, string, int, int) {
	switch {
	case len(replacement.Records) > 0:
		return "records", summarizeFillTemplateRecordPreview(replacement), 0, len(replacement.Records)
	case len(replacement.Grid) > 0:
		return "grid", summarizeFillTemplateGridPreview(replacement.Grid), len(replacement.Grid), 0
	case len(replacement.Values) > 0:
		return "values", summarizeFillTemplateValuesPreview(replacement.Values), len(replacement.Values), 0
	default:
		return "value", truncateFillTemplateResolutionText(strings.TrimSpace(replacement.Value), 120), 1, 0
	}
}

func summarizeFillTemplateRecordPreview(replacement FillTemplateReplacement) string {
	if len(replacement.Records) == 0 {
		return ""
	}
	first := replacement.Records[0]
	parts := make([]string, 0, len(replacement.Fields))
	for _, field := range replacement.Fields {
		parts = append(parts, fmt.Sprintf("%s=%s", field, first[field]))
	}
	if len(parts) == 0 {
		for key, value := range first {
			parts = append(parts, fmt.Sprintf("%s=%s", key, value))
		}
	}
	return truncateFillTemplateResolutionText(strings.Join(parts, ", "), 120)
}

func summarizeFillTemplateGridPreview(grid [][]string) string {
	if len(grid) == 0 {
		return ""
	}
	return truncateFillTemplateResolutionText(strings.Join(grid[0], " | "), 120)
}

func summarizeFillTemplateValuesPreview(values []string) string {
	if len(values) == 0 {
		return ""
	}
	preview := values
	if len(preview) > 3 {
		preview = preview[:3]
	}
	return truncateFillTemplateResolutionText(strings.Join(preview, ", "), 120)
}

func truncateFillTemplateResolutionText(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit]) + "..."
}

func resolveFillTemplateResolutionMode(replacement FillTemplateReplacement) string {
	mode := strings.ToLower(strings.TrimSpace(replacement.Mode))
	if mode != "" {
		return mode
	}
	switch {
	case strings.TrimSpace(replacement.Placeholder) != "":
		return "replace"
	case strings.TrimSpace(replacement.NearText) != "":
		if len(replacement.Values) > 0 {
			return "paragraph-next-repeat"
		}
		return "paragraph-next"
	case len(replacement.Records) > 0:
		return "table-down-records"
	case len(replacement.Grid) > 0:
		return "table-right-grid"
	case len(replacement.Values) > 0:
		return "table-down-repeat"
	default:
		return "table-right"
	}
}

func CorrelateFillTemplateResolution(report *FillTemplateResolutionReport, changes []FillTemplateChange, misses []FillTemplateMiss) {
	if report == nil || len(report.Entries) == 0 {
		return
	}
	for index := range report.Entries {
		report.Entries[index].ChangeCount = 0
		report.Entries[index].MissCount = 0
		report.Entries[index].ChangeIndexes = nil
		report.Entries[index].MissIndexes = nil
	}
	for index, change := range changes {
		if change.ResolutionIndex == nil {
			continue
		}
		resolutionIndex := *change.ResolutionIndex
		if resolutionIndex < 0 || resolutionIndex >= len(report.Entries) {
			continue
		}
		report.Entries[resolutionIndex].ChangeCount++
		report.Entries[resolutionIndex].ChangeIndexes = append(report.Entries[resolutionIndex].ChangeIndexes, index)
	}
	for index, miss := range misses {
		if miss.ResolutionIndex == nil {
			continue
		}
		resolutionIndex := *miss.ResolutionIndex
		if resolutionIndex < 0 || resolutionIndex >= len(report.Entries) {
			continue
		}
		report.Entries[resolutionIndex].MissCount++
		report.Entries[resolutionIndex].MissIndexes = append(report.Entries[resolutionIndex].MissIndexes, index)
	}
}
