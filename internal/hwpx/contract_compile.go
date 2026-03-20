package hwpx

import (
	"fmt"
	"strings"
)

func CompileTemplateContract(contract TemplateContract, payload map[string]any) ([]FillTemplateReplacement, error) {
	replacements, _, err := CompileTemplateContractWithResolution(contract, payload)
	return replacements, err
}

func CompileTemplateContractWithResolution(contract TemplateContract, payload map[string]any) ([]FillTemplateReplacement, FillTemplateResolutionReport, error) {
	if err := ValidateTemplateContract(contract); err != nil {
		return nil, FillTemplateResolutionReport{}, err
	}

	var replacements []FillTemplateReplacement
	report := FillTemplateResolutionReport{
		InputKind: "contract",
		Entries:   []FillTemplateResolutionEntry{},
	}

	for _, field := range contract.Fields {
		replacement, entry, ok, err := compileTemplateContractField(field, payload)
		if err != nil {
			return nil, FillTemplateResolutionReport{}, err
		}
		entry.Index = len(report.Entries)
		if ok {
			resolutionIndex := entry.Index
			replacement.SourceIndex = &resolutionIndex
		}
		report.Entries = append(report.Entries, entry)
		if ok {
			replacements = append(replacements, replacement)
			report.ResolvedCount++
		} else {
			report.SkippedCount++
		}
	}

	for _, table := range contract.Tables {
		replacement, entry, ok, err := compileTemplateContractTable(table, payload)
		if err != nil {
			return nil, FillTemplateResolutionReport{}, err
		}
		entry.Index = len(report.Entries)
		if ok {
			resolutionIndex := entry.Index
			replacement.SourceIndex = &resolutionIndex
		}
		report.Entries = append(report.Entries, entry)
		if ok {
			replacements = append(replacements, replacement)
			report.ResolvedCount++
		} else {
			report.SkippedCount++
		}
	}

	report.EntryCount = len(report.Entries)
	return replacements, report, nil
}

func compileTemplateContractField(field TemplateContractField, payload map[string]any) (FillTemplateReplacement, FillTemplateResolutionEntry, bool, error) {
	replacement := FillTemplateReplacement{
		Mode:          normalizeContractCompilerMode(field.Mode, normalizeContractSelectorMode(field.Selector.Type)),
		MatchMode:     strings.ToLower(strings.TrimSpace(field.Selector.MatchMode)),
		TableLabel:    strings.TrimSpace(field.Selector.TableLabel),
		Required:      field.Required,
		FallbackValue: field.FallbackValue,
	}
	applyContractSelector(&replacement, field.Selector)

	value, ok := resolvePayloadPath(payload, field.Key)
	usedFallback := false
	if !ok {
		if strings.TrimSpace(field.FallbackValue) != "" {
			value = field.FallbackValue
			ok = true
			usedFallback = true
		} else if field.Required {
			return FillTemplateReplacement{}, FillTemplateResolutionEntry{}, false, fmt.Errorf("payload is missing required field %q", field.Key)
		} else {
			entry := buildFillTemplateResolutionEntry("field", replacement)
			entry.Key = field.Key
			entry.PayloadPath = field.Key
			entry.ValueKind = ""
			entry.ValuePreview = ""
			entry.ValueCount = 0
			entry.RecordCount = 0
			entry.Fields = nil
			entry.Skipped = true
			entry.SkipReason = "payload missing optional field"
			return FillTemplateReplacement{}, entry, false, nil
		}
	}

	text, err := stringifyPayloadScalar(value)
	if err != nil {
		return FillTemplateReplacement{}, FillTemplateResolutionEntry{}, false, fmt.Errorf("field %q: %w", field.Key, err)
	}

	replacement.Value = text
	entry := buildFillTemplateResolutionEntry("field", replacement)
	entry.Key = field.Key
	entry.PayloadPath = field.Key
	entry.UsedFallback = usedFallback
	return replacement, entry, true, nil
}

func compileTemplateContractTable(table TemplateContractTable, payload map[string]any) (FillTemplateReplacement, FillTemplateResolutionEntry, bool, error) {
	replacement := FillTemplateReplacement{
		Mode:       normalizeContractCompilerMode(table.Mode, defaultContractTableMode(nil, len(table.Columns) > 0)),
		MatchMode:  strings.ToLower(strings.TrimSpace(table.Selector.MatchMode)),
		TableLabel: strings.TrimSpace(table.Selector.TableLabel),
		Required:   table.Required,
		Expand:     table.Expand,
	}
	applyContractSelector(&replacement, table.Selector)

	value, ok := resolvePayloadPath(payload, table.Key)
	if !ok {
		if table.Required {
			return FillTemplateReplacement{}, FillTemplateResolutionEntry{}, false, fmt.Errorf("payload is missing required table %q", table.Key)
		}
		entry := buildFillTemplateResolutionEntry("table", replacement)
		entry.Key = table.Key
		entry.PayloadPath = table.Key
		entry.ValueKind = ""
		entry.ValuePreview = ""
		entry.ValueCount = 0
		entry.RecordCount = 0
		entry.Fields = nil
		entry.Skipped = true
		entry.SkipReason = "payload missing optional table"
		return FillTemplateReplacement{}, entry, false, nil
	}

	replacement.Mode = normalizeContractCompilerMode(table.Mode, defaultContractTableMode(value, len(table.Columns) > 0))

	switch {
	case len(table.Columns) > 0:
		records, fields, err := compileTemplateContractRecords(table, value)
		if err != nil {
			return FillTemplateReplacement{}, FillTemplateResolutionEntry{}, false, err
		}
		replacement.Mode = normalizeContractCompilerMode(replacement.Mode, "table-down-records")
		replacement.Fields = fields
		replacement.Records = records
		replacement.Expand = true
	case isPayloadGrid(value):
		grid, err := stringifyPayloadGrid(value)
		if err != nil {
			return FillTemplateReplacement{}, FillTemplateResolutionEntry{}, false, fmt.Errorf("table %q: %w", table.Key, err)
		}
		replacement.Grid = grid
	case isPayloadList(value):
		values, err := stringifyPayloadList(value)
		if err != nil {
			return FillTemplateReplacement{}, FillTemplateResolutionEntry{}, false, fmt.Errorf("table %q: %w", table.Key, err)
		}
		replacement.Values = values
	default:
		text, err := stringifyPayloadScalar(value)
		if err != nil {
			return FillTemplateReplacement{}, FillTemplateResolutionEntry{}, false, fmt.Errorf("table %q: %w", table.Key, err)
		}
		replacement.Value = text
	}

	entry := buildFillTemplateResolutionEntry("table", replacement)
	entry.Key = table.Key
	entry.PayloadPath = table.Key
	return replacement, entry, true, nil
}

func compileTemplateContractRecords(table TemplateContractTable, value any) ([]map[string]string, []string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, nil, fmt.Errorf("table %q expects an array payload", table.Key)
	}

	fields := make([]string, 0, len(table.Columns))
	for _, column := range table.Columns {
		fields = append(fields, column.Key)
	}

	records := make([]map[string]string, 0, len(items))
	for recordIndex, item := range items {
		record := make(map[string]string, len(table.Columns))
		for _, column := range table.Columns {
			resolved, ok := resolvePayloadPathValue(item, column.Source)
			if !ok {
				return nil, nil, fmt.Errorf("table %q record[%d] is missing source %q", table.Key, recordIndex, column.Source)
			}
			text, err := stringifyPayloadScalar(resolved)
			if err != nil {
				return nil, nil, fmt.Errorf("table %q record[%d] column %q: %w", table.Key, recordIndex, column.Key, err)
			}
			record[column.Key] = text
		}
		records = append(records, record)
	}

	return records, fields, nil
}

func applyContractSelector(replacement *FillTemplateReplacement, selector TemplateContractSelector) {
	switch normalizeContractSelectorType(selector.Type) {
	case "placeholder":
		replacement.Placeholder = selector.Value
	case "near-text":
		replacement.NearText = selector.Value
	default:
		replacement.Anchor = selector.Value
	}
}

func resolvePayloadPath(payload map[string]any, path string) (any, bool) {
	return resolvePayloadPathValue(payload, path)
}

func resolvePayloadPathValue(value any, path string) (any, bool) {
	current := value
	for _, part := range strings.Split(strings.TrimSpace(path), ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := object[part]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func stringifyPayloadScalar(value any) (string, error) {
	switch typed := value.(type) {
	case nil:
		return "", fmt.Errorf("value is null")
	case string:
		return typed, nil
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(typed), nil
	default:
		return "", fmt.Errorf("value must be scalar")
	}
}

func stringifyPayloadList(value any) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("value must be an array")
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		text, err := stringifyPayloadScalar(item)
		if err != nil {
			return nil, err
		}
		values = append(values, text)
	}
	return values, nil
}

func stringifyPayloadGrid(value any) ([][]string, error) {
	rows, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("value must be a 2D array")
	}
	grid := make([][]string, 0, len(rows))
	for _, rowValue := range rows {
		rowItems, ok := rowValue.([]any)
		if !ok {
			return nil, fmt.Errorf("grid rows must be arrays")
		}
		row := make([]string, 0, len(rowItems))
		for _, item := range rowItems {
			text, err := stringifyPayloadScalar(item)
			if err != nil {
				return nil, err
			}
			row = append(row, text)
		}
		grid = append(grid, row)
	}
	return grid, nil
}

func isPayloadList(value any) bool {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return false
	}
	for _, item := range items {
		if _, ok := item.([]any); ok {
			return false
		}
	}
	return true
}

func isPayloadGrid(value any) bool {
	rows, ok := value.([]any)
	if !ok || len(rows) == 0 {
		return false
	}
	for _, row := range rows {
		if _, ok := row.([]any); !ok {
			return false
		}
	}
	return true
}

func normalizeContractSelectorType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "near-text", "near_text":
		return "near-text"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeContractSelectorMode(selectorType string) string {
	switch normalizeContractSelectorType(selectorType) {
	case "placeholder":
		return "replace"
	case "near-text":
		return "paragraph-next"
	default:
		return "table-right"
	}
}

func defaultContractTableMode(value any, hasColumns bool) string {
	if hasColumns {
		return "table-down-records"
	}
	if isPayloadGrid(value) {
		return "table-right-grid"
	}
	if isPayloadList(value) {
		return "table-down-repeat"
	}
	return "table-right"
}

func normalizeContractCompilerMode(value string, fallback string) string {
	mode := strings.ToLower(strings.TrimSpace(value))
	switch mode {
	case "":
		return fallback
	case "paragraph_next":
		return "paragraph-next"
	case "table_right":
		return "table-right"
	case "table_down":
		return "table-down"
	case "table_left":
		return "table-left"
	case "table_up":
		return "table-up"
	case "table_down_repeat":
		return "table-down-repeat"
	case "table_right_grid":
		return "table-right-grid"
	case "table_down_records":
		return "table-down-records"
	default:
		return mode
	}
}
