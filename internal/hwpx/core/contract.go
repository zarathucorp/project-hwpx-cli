package core

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type TemplateContract struct {
	TemplateID      string                  `json:"templateId" yaml:"template_id"`
	TemplateVersion string                  `json:"templateVersion" yaml:"template_version"`
	Strict          bool                    `json:"strict,omitempty" yaml:"strict,omitempty"`
	Fingerprint     TemplateFingerprint     `json:"fingerprint" yaml:"fingerprint"`
	Fields          []TemplateContractField `json:"fields,omitempty" yaml:"fields,omitempty"`
	Tables          []TemplateContractTable `json:"tables,omitempty" yaml:"tables,omitempty"`
}

type TemplateContractField struct {
	Key           string                   `json:"key" yaml:"key"`
	Selector      TemplateContractSelector `json:"selector" yaml:"selector"`
	Mode          string                   `json:"mode,omitempty" yaml:"mode,omitempty"`
	Required      bool                     `json:"required,omitempty" yaml:"required,omitempty"`
	FallbackValue string                   `json:"fallbackValue,omitempty" yaml:"fallback_value,omitempty"`
}

type TemplateContractTable struct {
	Key      string                   `json:"key" yaml:"key"`
	Selector TemplateContractSelector `json:"selector" yaml:"selector"`
	Mode     string                   `json:"mode,omitempty" yaml:"mode,omitempty"`
	Required bool                     `json:"required,omitempty" yaml:"required,omitempty"`
	Expand   bool                     `json:"expand,omitempty" yaml:"expand,omitempty"`
	Columns  []TemplateContractColumn `json:"columns,omitempty" yaml:"columns,omitempty"`
}

type TemplateContractColumn struct {
	Key    string `json:"key" yaml:"key"`
	Source string `json:"source" yaml:"source"`
}

type TemplateContractSelector struct {
	Type       string `json:"type" yaml:"type"`
	Value      string `json:"value" yaml:"value"`
	TableLabel string `json:"tableLabel,omitempty" yaml:"table_label,omitempty"`
	MatchMode  string `json:"matchMode,omitempty" yaml:"match_mode,omitempty"`
}

func LoadTemplateContract(path string) (TemplateContract, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return TemplateContract{}, err
	}

	contract, err := parseTemplateContract(content)
	if err != nil {
		return TemplateContract{}, err
	}
	if err := ValidateTemplateContract(contract); err != nil {
		return TemplateContract{}, err
	}
	return contract, nil
}

func parseTemplateContract(content []byte) (TemplateContract, error) {
	var contract TemplateContract
	if err := json.Unmarshal(content, &contract); err == nil && templateContractLooksDefined(contract) {
		return contract, nil
	}
	if err := yaml.Unmarshal(content, &contract); err == nil && templateContractLooksDefined(contract) {
		return contract, nil
	}
	return TemplateContract{}, fmt.Errorf("invalid template contract: expected JSON or YAML contract object")
}

func ValidateTemplateContract(contract TemplateContract) error {
	var issues []string

	if strings.TrimSpace(contract.TemplateID) == "" {
		issues = append(issues, "templateId is required")
	}
	if strings.TrimSpace(contract.TemplateVersion) == "" {
		issues = append(issues, "templateVersion is required")
	}
	if contract.Fingerprint.SectionCount <= 0 {
		issues = append(issues, "fingerprint.sectionCount must be greater than zero")
	}
	if len(contract.Fields) == 0 && len(contract.Tables) == 0 {
		issues = append(issues, "at least one field or table binding is required")
	}

	fieldKeys := map[string]struct{}{}
	for index, field := range contract.Fields {
		prefix := fmt.Sprintf("fields[%d]", index)
		if strings.TrimSpace(field.Key) == "" {
			issues = append(issues, prefix+".key is required")
		} else if _, exists := fieldKeys[field.Key]; exists {
			issues = append(issues, prefix+".key must be unique")
		} else {
			fieldKeys[field.Key] = struct{}{}
		}
		issues = append(issues, validateTemplateContractSelector(prefix+".selector", field.Selector, false)...)
		mode := normalizeTemplateContractMode(field.Mode)
		if mode != "" && !slices.Contains([]string{
			"replace",
			"table-right",
			"table-down",
			"table-left",
			"table-up",
			"paragraph-next",
		}, mode) {
			issues = append(issues, prefix+".mode is not supported")
		}
	}

	tableKeys := map[string]struct{}{}
	for index, table := range contract.Tables {
		prefix := fmt.Sprintf("tables[%d]", index)
		if strings.TrimSpace(table.Key) == "" {
			issues = append(issues, prefix+".key is required")
		} else if _, exists := tableKeys[table.Key]; exists {
			issues = append(issues, prefix+".key must be unique")
		} else {
			tableKeys[table.Key] = struct{}{}
		}
		issues = append(issues, validateTemplateContractSelector(prefix+".selector", table.Selector, true)...)
		if normalizeTemplateContractSelectorType(table.Selector.Type) != "anchor" {
			issues = append(issues, prefix+".selector.type must be anchor")
		}
		mode := normalizeTemplateContractMode(table.Mode)
		if mode == "" {
			mode = "table-down-repeat"
		}
		if !slices.Contains([]string{
			"table-down-repeat",
			"table-right-grid",
			"table-down-records",
		}, mode) {
			issues = append(issues, prefix+".mode is not supported")
		}
		for columnIndex, column := range table.Columns {
			columnPrefix := fmt.Sprintf("%s.columns[%d]", prefix, columnIndex)
			if strings.TrimSpace(column.Key) == "" {
				issues = append(issues, columnPrefix+".key is required")
			}
			if strings.TrimSpace(column.Source) == "" {
				issues = append(issues, columnPrefix+".source is required")
			}
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("invalid template contract: %s", strings.Join(issues, "; "))
	}
	return nil
}

func VerifyTemplateContractFingerprint(contract TemplateContract, analysis TemplateAnalysis) error {
	expected := contract.Fingerprint
	actual := analysis.Fingerprint
	var issues []string

	if expected.SectionCount > 0 && expected.SectionCount != actual.SectionCount {
		issues = append(issues, fmt.Sprintf("fingerprint.sectionCount mismatch: expected %d got %d", expected.SectionCount, actual.SectionCount))
	}
	if len(expected.SectionPaths) > 0 && !slices.Equal(expected.SectionPaths, actual.SectionPaths) {
		issues = append(issues, "fingerprint.sectionPaths mismatch")
	}
	if expected.PlaceholderDigest != "" && expected.PlaceholderDigest != actual.PlaceholderDigest {
		issues = append(issues, "fingerprint.placeholderDigest mismatch")
	}
	for _, label := range expected.TableLabels {
		if !slices.Contains(actual.TableLabels, label) {
			issues = append(issues, fmt.Sprintf("fingerprint.tableLabels missing %q", label))
		}
	}
	for _, text := range expected.PlaceholderTexts {
		if !slices.Contains(actual.PlaceholderTexts, text) {
			issues = append(issues, fmt.Sprintf("fingerprint.placeholderTexts missing %q", text))
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf("template fingerprint mismatch: %s", strings.Join(issues, "; "))
	}
	return nil
}

func templateContractLooksDefined(contract TemplateContract) bool {
	return strings.TrimSpace(contract.TemplateID) != "" ||
		strings.TrimSpace(contract.TemplateVersion) != "" ||
		len(contract.Fields) > 0 ||
		len(contract.Tables) > 0 ||
		contract.Fingerprint.SectionCount > 0
}

func validateTemplateContractSelector(prefix string, selector TemplateContractSelector, allowTableContext bool) []string {
	var issues []string

	selectorType := normalizeTemplateContractSelectorType(selector.Type)
	if selectorType == "" {
		issues = append(issues, prefix+".type is required")
	} else if !slices.Contains([]string{"placeholder", "anchor", "near-text"}, selectorType) {
		issues = append(issues, prefix+".type is not supported")
	}
	if strings.TrimSpace(selector.Value) == "" {
		issues = append(issues, prefix+".value is required")
	}
	if !allowTableContext && strings.TrimSpace(selector.TableLabel) != "" && selectorType == "placeholder" {
		issues = append(issues, prefix+".tableLabel is not supported for placeholder selectors")
	}
	matchMode := normalizeTemplateContractMatchMode(selector.MatchMode)
	if matchMode != "" && !slices.Contains([]string{"contains", "exact"}, matchMode) {
		issues = append(issues, prefix+".matchMode is not supported")
	}

	return issues
}

func normalizeTemplateContractSelectorType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "near-text", "near_text":
		return "near-text"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeTemplateContractMatchMode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeTemplateContractMode(value string) string {
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
