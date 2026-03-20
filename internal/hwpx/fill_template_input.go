package hwpx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const fillTemplateMappingSchemaVersion = "hwpxctl/fill-template-mapping/v1"

type ResolvedFillTemplateInput struct {
	InputKind    string
	MappingPath  string
	TemplatePath string
	PayloadPath  string
	Replacements []FillTemplateReplacement
	Resolution   FillTemplateResolutionReport
}

type FillTemplateInputError struct {
	Kind string
	Err  error
}

func (e *FillTemplateInputError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *FillTemplateInputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type fillTemplateMappingFile struct {
	SchemaVersion string                    `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`
	Entries       []FillTemplateReplacement `json:"entries,omitempty" yaml:"entries,omitempty"`
	Replacements  []FillTemplateReplacement `json:"replacements,omitempty" yaml:"replacements,omitempty"`
}

func ResolveFillTemplateInput(targetPath, mappingPath, templatePath, payloadPath string) (ResolvedFillTemplateInput, error) {
	mappingPath = strings.TrimSpace(mappingPath)
	templatePath = strings.TrimSpace(templatePath)
	payloadPath = strings.TrimSpace(payloadPath)

	if (mappingPath == "" && (templatePath == "" || payloadPath == "")) ||
		(mappingPath != "" && (templatePath != "" || payloadPath != "")) {
		return ResolvedFillTemplateInput{}, newFillTemplateInputError("invalid_arguments", "fill-template input requires either --mapping or both --template and --payload")
	}

	if mappingPath != "" {
		replacements, err := readFillTemplateMapping(mappingPath)
		if err != nil {
			return ResolvedFillTemplateInput{}, err
		}
		replacements = assignFillTemplateSourceIndexes(replacements)
		resolution := BuildMappingFillTemplateResolutionReport(replacements)
		return ResolvedFillTemplateInput{
			InputKind:    "mapping",
			MappingPath:  absoluteFilePath(mappingPath),
			Replacements: replacements,
			Resolution:   resolution,
		}, nil
	}

	contract, err := LoadTemplateContract(templatePath)
	if err != nil {
		return ResolvedFillTemplateInput{}, newFillTemplateInputError("invalid_arguments", err.Error())
	}
	analysis, err := AnalyzeTemplate(targetPath)
	if err != nil {
		return ResolvedFillTemplateInput{}, err
	}
	if err := VerifyTemplateContractFingerprint(contract, analysis); err != nil {
		return ResolvedFillTemplateInput{}, newFillTemplateInputError("template_contract_mismatch", err.Error())
	}
	payload, err := readTemplatePayload(payloadPath)
	if err != nil {
		return ResolvedFillTemplateInput{}, err
	}
	replacements, resolution, err := CompileTemplateContractWithResolution(contract, payload)
	if err != nil {
		return ResolvedFillTemplateInput{}, newFillTemplateInputError("invalid_arguments", err.Error())
	}
	replacements, err = validateFillTemplateReplacements(replacements)
	if err != nil {
		return ResolvedFillTemplateInput{}, err
	}

	return ResolvedFillTemplateInput{
		InputKind:    "contract",
		TemplatePath: absoluteFilePath(templatePath),
		PayloadPath:  absoluteFilePath(payloadPath),
		Replacements: replacements,
		Resolution:   resolution,
	}, nil
}

func readFillTemplateMapping(path string) ([]FillTemplateReplacement, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapped fillTemplateMappingFile
	if err := json.Unmarshal(content, &wrapped); err == nil {
		replacements, ok, err := normalizeFillTemplateMappingFile(wrapped)
		if err != nil {
			return nil, err
		}
		if ok {
			return validateFillTemplateReplacements(replacements)
		}
	}

	var direct []FillTemplateReplacement
	if err := json.Unmarshal(content, &direct); err == nil {
		return validateFillTemplateReplacements(direct)
	}

	if err := yaml.Unmarshal(content, &wrapped); err == nil {
		replacements, ok, err := normalizeFillTemplateMappingFile(wrapped)
		if err != nil {
			return nil, err
		}
		if ok {
			return validateFillTemplateReplacements(replacements)
		}
	}
	if err := yaml.Unmarshal(content, &direct); err == nil {
		return validateFillTemplateReplacements(direct)
	}

	return nil, newFillTemplateInputError("invalid_arguments", "invalid mapping file: expected JSON or YAML replacements")
}

func normalizeFillTemplateMappingFile(document fillTemplateMappingFile) ([]FillTemplateReplacement, bool, error) {
	schemaVersion := strings.TrimSpace(document.SchemaVersion)
	if schemaVersion != "" && schemaVersion != fillTemplateMappingSchemaVersion {
		return nil, false, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("invalid mapping schemaVersion %q", schemaVersion))
	}

	switch {
	case len(document.Entries) > 0 && len(document.Replacements) > 0:
		return nil, false, newFillTemplateInputError("invalid_arguments", "mapping file must define either entries or replacements")
	case len(document.Entries) > 0:
		return document.Entries, true, nil
	case len(document.Replacements) > 0:
		return document.Replacements, true, nil
	case schemaVersion != "":
		return nil, false, newFillTemplateInputError("invalid_arguments", "mapping file entries must not be empty")
	default:
		return nil, false, nil
	}
}

func readTemplatePayload(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err == nil && len(payload) > 0 {
		return payload, nil
	}
	if err := yaml.Unmarshal(content, &payload); err == nil && len(payload) > 0 {
		return payload, nil
	}

	return nil, newFillTemplateInputError("invalid_arguments", "invalid payload file: expected JSON or YAML object")
}

func assignFillTemplateSourceIndexes(replacements []FillTemplateReplacement) []FillTemplateReplacement {
	for index := range replacements {
		sourceIndex := index
		replacements[index].SourceIndex = &sourceIndex
	}
	return replacements
}

func validateFillTemplateReplacements(replacements []FillTemplateReplacement) ([]FillTemplateReplacement, error) {
	for index, replacement := range replacements {
		selectorCount := 0
		if strings.TrimSpace(replacement.Placeholder) != "" {
			selectorCount++
		}
		if strings.TrimSpace(replacement.Anchor) != "" {
			selectorCount++
		}
		if strings.TrimSpace(replacement.NearText) != "" {
			selectorCount++
		}
		if selectorCount != 1 {
			return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] must define exactly one of placeholder, anchor, or nearText", index))
		}

		hasValue := strings.TrimSpace(replacement.Value) != ""
		hasValues := len(replacement.Values) > 0
		hasGrid := len(replacement.Grid) > 0
		hasRecords := len(replacement.Records) > 0
		valueKinds := 0
		if hasValue {
			valueKinds++
		}
		if hasValues {
			valueKinds++
		}
		if hasGrid {
			valueKinds++
		}
		if hasRecords {
			valueKinds++
		}
		if valueKinds != 1 {
			return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] must define exactly one of value, values, grid, or records", index))
		}

		mode := strings.ToLower(strings.TrimSpace(replacement.Mode))
		if mode == "" {
			switch {
			case strings.TrimSpace(replacement.Placeholder) != "":
				mode = "replace"
			case strings.TrimSpace(replacement.NearText) != "":
				if hasValues {
					mode = "paragraph-next-repeat"
				} else {
					mode = "paragraph-next"
				}
			case hasRecords:
				mode = "table-down-records"
			case hasGrid:
				mode = "table-right-grid"
			case hasValues:
				mode = "table-down-repeat"
			default:
				mode = "table-right"
			}
		}
		switch mode {
		case "replace":
			if strings.TrimSpace(replacement.Placeholder) == "" {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode replace requires placeholder", index))
			}
		case "table-right", "table-down", "table-left", "table-up":
			if strings.TrimSpace(replacement.Anchor) == "" || !hasValue {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode %s requires anchor and value", index, mode))
			}
			if replacement.Expand && mode != "table-down" {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] expand is only supported with table-down, table-down-repeat, or table-down-grid", index))
			}
		case "table-right-repeat", "table-down-repeat", "table-left-repeat", "table-up-repeat":
			if strings.TrimSpace(replacement.Anchor) == "" || !hasValues {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode %s requires anchor and values", index, mode))
			}
			if replacement.Expand && mode != "table-down-repeat" {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] expand is only supported with table-down, table-down-repeat, or table-down-grid", index))
			}
		case "table-right-grid", "table-down-grid":
			if strings.TrimSpace(replacement.Anchor) == "" || !hasGrid {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode %s requires anchor and grid", index, mode))
			}
			if replacement.Expand && mode != "table-down-grid" {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] expand is only supported with table-down, table-down-repeat, or table-down-grid", index))
			}
		case "table-down-records":
			if strings.TrimSpace(replacement.Anchor) == "" || !hasRecords {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode %s requires anchor and records", index, mode))
			}
			if len(replacement.Fields) == 0 {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode %s requires fields", index, mode))
			}
			if !replacement.Expand {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode %s requires expand", index, mode))
			}
		case "paragraph-next", "paragraph-replace":
			if strings.TrimSpace(replacement.NearText) == "" || !hasValue {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode %s requires nearText and value", index, mode))
			}
		case "paragraph-next-repeat", "paragraph-replace-repeat":
			if strings.TrimSpace(replacement.NearText) == "" || !hasValues {
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] mode %s requires nearText and values", index, mode))
			}
		default:
			return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] has unsupported mode %q", index, replacement.Mode))
		}

		if strings.TrimSpace(replacement.TableLabel) != "" && strings.TrimSpace(replacement.Anchor) == "" {
			return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] tableLabel requires anchor", index))
		}
		if replacement.TableIndex != nil && strings.TrimSpace(replacement.Anchor) == "" {
			return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] tableIndex requires anchor", index))
		}
		if replacement.TableIndex != nil && *replacement.TableIndex < 0 {
			return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] tableIndex must be zero or greater", index))
		}
		if replacement.Occurrence != nil && *replacement.Occurrence <= 0 {
			return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] occurrence must be greater than zero", index))
		}
		if matchMode := strings.ToLower(strings.TrimSpace(replacement.MatchMode)); matchMode != "" && matchMode != "contains" && matchMode != "exact" {
			return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] matchMode must be contains or exact", index))
		}
		if replacement.Unique {
			switch mode {
			case "replace", "table-right", "table-down", "table-left", "table-up", "paragraph-next", "paragraph-replace":
			default:
				return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] unique is only supported with single-target modes", index))
			}
		}
		if hasRecords {
			for fieldIndex, field := range replacement.Fields {
				if strings.TrimSpace(field) == "" {
					return nil, newFillTemplateInputError("invalid_arguments", fmt.Sprintf("replacement[%d] fields[%d] must not be empty", index, fieldIndex))
				}
			}
		}
	}

	return replacements, nil
}

func newFillTemplateInputError(kind, message string) error {
	return &FillTemplateInputError{
		Kind: kind,
		Err:  fmt.Errorf("%s", message),
	}
}

func absoluteFilePath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return absolute
}
