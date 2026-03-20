package hwpx

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	contractScaffoldSplitPattern       = regexp.MustCompile(`[^\p{L}\p{N}]+`)
	contractScaffoldZeroRunPattern     = regexp.MustCompile(`0{2,}`)
	contractScaffoldCorpMarkPattern    = regexp.MustCompile(`\(\s*주\s*\)|（\s*주\s*）`)
	scaffoldTemplatePlaceholderPattern = regexp.MustCompile(`\{\{[^{}]+\}\}|_{3,}|□{1,}|▢{1,}`)
)

func ScaffoldTemplateContract(targetPath, templateID, templateVersion string, strict bool) (TemplateContract, error) {
	analysis, err := AnalyzeTemplate(targetPath)
	if err != nil {
		return TemplateContract{}, err
	}

	fields := scaffoldTemplateContractFields(analysis.Placeholders)
	if len(fields) == 0 {
		return TemplateContract{}, fmt.Errorf("cannot scaffold template contract: no placeholder candidates found")
	}

	resolvedTemplateID := strings.TrimSpace(templateID)
	if resolvedTemplateID == "" {
		resolvedTemplateID = scaffoldTemplateContractTemplateID(targetPath)
	}
	resolvedTemplateVersion := strings.TrimSpace(templateVersion)
	if resolvedTemplateVersion == "" {
		resolvedTemplateVersion = "1.0.0"
	}

	contract := TemplateContract{
		TemplateID:      resolvedTemplateID,
		TemplateVersion: resolvedTemplateVersion,
		Strict:          strict,
		Fingerprint:     scaffoldTemplateFingerprint(analysis),
		Fields:          fields,
	}
	if err := ValidateTemplateContract(contract); err != nil {
		return TemplateContract{}, err
	}
	return contract, nil
}

func ScaffoldTemplatePayload(contract TemplateContract) (map[string]any, error) {
	if err := ValidateTemplateContract(contract); err != nil {
		return nil, err
	}

	payload := map[string]any{}
	for _, field := range contract.Fields {
		value := any("")
		if strings.TrimSpace(field.FallbackValue) != "" {
			value = field.FallbackValue
		}
		if err := assignScaffoldTemplatePayloadValue(payload, field.Key, value); err != nil {
			return nil, fmt.Errorf("field %q: %w", field.Key, err)
		}
	}
	for _, table := range contract.Tables {
		if err := assignScaffoldTemplatePayloadValue(payload, table.Key, scaffoldTemplatePayloadTableValue(table)); err != nil {
			return nil, fmt.Errorf("table %q: %w", table.Key, err)
		}
	}
	return payload, nil
}

func scaffoldTemplateFingerprint(analysis TemplateAnalysis) TemplateFingerprint {
	fingerprint := analysis.Fingerprint
	fingerprint.TableLabels = curateScaffoldTemplateFingerprintTableLabels(analysis.Tables)
	return fingerprint
}

func scaffoldTemplateContractFields(candidates []TemplateTextCandidate) []TemplateContractField {
	fields := make([]TemplateContractField, 0, len(candidates))
	seenSelectors := map[string]struct{}{}
	seenKeys := map[string]int{}
	for _, candidate := range candidates {
		selector := strings.TrimSpace(candidate.Text)
		if selector == "" {
			continue
		}
		if _, ok := seenSelectors[selector]; ok {
			continue
		}
		seenSelectors[selector] = struct{}{}
		key := scaffoldTemplateContractFieldKey(selector, len(fields)+1)
		key = dedupeScaffoldTemplateContractFieldKey(key, seenKeys)
		fields = append(fields, TemplateContractField{
			Key: key,
			Selector: TemplateContractSelector{
				Type:  "placeholder",
				Value: selector,
			},
		})
	}
	return fields
}

func scaffoldTemplateContractTemplateID(targetPath string) string {
	name := strings.TrimSpace(strings.TrimSuffix(filepath.Base(targetPath), filepath.Ext(targetPath)))
	key := scaffoldTemplateContractFieldKey(name, 0)
	if key == "" {
		return "template_contract"
	}
	key = strings.ReplaceAll(key, ".", "_")
	if startsWithDigit(key) {
		return "template_" + key
	}
	return key
}

func scaffoldTemplateContractFieldKey(value string, fallbackIndex int) string {
	normalized := scaffoldTemplateContractKeySource(value)
	parts := contractScaffoldSplitPattern.Split(normalized, -1)
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	switch len(filtered) {
	case 0:
		if fallbackIndex <= 0 {
			return ""
		}
		return fmt.Sprintf("field_%d", fallbackIndex)
	case 1:
		return filtered[0]
	case 2:
		return filtered[0] + "." + filtered[1]
	default:
		return filtered[0] + "." + strings.Join(filtered[1:], "_")
	}
}

func curateScaffoldTemplateFingerprintTableLabels(tables []TemplateTable) []string {
	values := make([]string, 0, len(tables))
	seen := map[string]struct{}{}
	fallbacks := make([]string, 0, len(tables))
	for _, table := range tables {
		label := sanitizeScaffoldTemplateTableLabel(table.LabelText)
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		fallbacks = append(fallbacks, label)
		if shouldKeepScaffoldTemplateTableLabel(label) {
			values = append(values, label)
		}
	}
	if len(values) == 0 {
		values = fallbacks
	}
	sort.Strings(values)
	if len(values) > 12 {
		values = values[:12]
	}
	return values
}

func scaffoldTemplateContractKeySource(value string) string {
	normalized := strings.TrimSpace(value)
	normalized = strings.ReplaceAll(normalized, "\n", " ")
	if index := strings.Index(normalized, "*"); index >= 0 {
		normalized = normalized[:index]
	}
	normalized = strings.TrimSpace(normalized)
	normalized = strings.NewReplacer(
		"{", "",
		"}", "",
		"[", "",
		"]", "",
		"<", "",
		">", "",
		"□", " ",
		"▢", " ",
		"※", " ",
	).Replace(normalized)
	normalized = contractScaffoldCorpMarkPattern.ReplaceAllString(normalized, " ")
	normalized = contractScaffoldZeroRunPattern.ReplaceAllString(normalized, " ")
	normalized = strings.TrimSpace(normalized)
	return strings.ToLower(normalized)
}

func sanitizeScaffoldTemplateTableLabel(value string) string {
	normalized := strings.ToValidUTF8(strings.TrimSpace(value), "")
	if normalized == "" {
		return ""
	}
	return strings.Join(strings.Fields(normalized), " ")
}

func shouldKeepScaffoldTemplateTableLabel(value string) bool {
	if value == "" {
		return false
	}
	if utf8.RuneCountInString(value) > 48 {
		return false
	}
	if strings.Contains(value, "�") {
		return false
	}
	if looksLikeScaffoldTemplatePlaceholderLabel(value) {
		return false
	}
	if looksLikeScaffoldTemplateGuideLabel(value) {
		return false
	}
	if looksLikeScaffoldTemplateDate(value) {
		return false
	}
	switch first := []rune(value)[0]; first {
	case '※', '*', '-', '<', '(', 'ㅇ':
		return false
	}
	return true
}

func looksLikeScaffoldTemplatePlaceholderLabel(value string) bool {
	if scaffoldTemplatePlaceholderPattern.MatchString(value) {
		return true
	}
	normalized := strings.TrimSpace(value)
	return strings.Contains(normalized, "{{") ||
		strings.Contains(normalized, "}}") ||
		strings.Contains(normalized, "□") ||
		contractScaffoldZeroRunPattern.MatchString(normalized)
}

func looksLikeScaffoldTemplateGuideLabel(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return strings.Contains(lower, "작성") ||
		strings.Contains(lower, "기재") ||
		strings.Contains(lower, "예시") ||
		strings.Contains(lower, "참고") ||
		strings.Contains(lower, "필요시") ||
		strings.Contains(lower, "유의")
}

func looksLikeScaffoldTemplateDate(value string) bool {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return false
	}
	hasDigit := false
	for _, r := range normalized {
		switch {
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsSpace(r):
		case r == '.', r == '-', r == '/', r == ':':
		default:
			return false
		}
	}
	return hasDigit
}

func dedupeScaffoldTemplateContractFieldKey(key string, seen map[string]int) string {
	count := seen[key]
	seen[key] = count + 1
	if count == 0 {
		return key
	}
	return fmt.Sprintf("%s_%d", key, count+1)
}

func startsWithDigit(value string) bool {
	for _, r := range value {
		return unicode.IsDigit(r)
	}
	return false
}

func assignScaffoldTemplatePayloadValue(root map[string]any, path string, value any) error {
	parts := strings.Split(strings.TrimSpace(path), ".")
	current := root
	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("invalid path")
		}
		if index == len(parts)-1 {
			if existing, ok := current[part]; ok {
				if !scaffoldTemplatePayloadValuesEqual(existing, value) {
					return fmt.Errorf("path conflicts with existing value")
				}
				return nil
			}
			current[part] = value
			return nil
		}
		next, ok := current[part]
		if !ok {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("path conflicts with scalar value")
		}
		current = child
	}
	return nil
}

func scaffoldTemplatePayloadTableValue(table TemplateContractTable) any {
	mode := normalizeScaffoldTemplateMode(table.Mode)
	if len(table.Columns) > 0 {
		record := map[string]any{}
		for _, column := range table.Columns {
			if err := assignScaffoldTemplatePayloadValue(record, column.Source, ""); err != nil {
				record[column.Source] = ""
			}
		}
		return []any{record}
	}
	if mode == "table-right-grid" {
		return []any{
			[]any{""},
		}
	}
	return []any{""}
}

func normalizeScaffoldTemplateMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
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
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func scaffoldTemplatePayloadValuesEqual(left any, right any) bool {
	switch leftTyped := left.(type) {
	case string:
		rightTyped, ok := right.(string)
		return ok && leftTyped == rightTyped
	case []any:
		rightTyped, ok := right.([]any)
		return ok && fmt.Sprint(leftTyped) == fmt.Sprint(rightTyped)
	case map[string]any:
		rightTyped, ok := right.(map[string]any)
		return ok && fmt.Sprint(leftTyped) == fmt.Sprint(rightTyped)
	default:
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
}
