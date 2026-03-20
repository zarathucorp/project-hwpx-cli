package hwpx

import "testing"

func TestCompileTemplateContract(t *testing.T) {
	contract := TemplateContract{
		TemplateID:      "project_form_v1",
		TemplateVersion: "1.0.0",
		Fingerprint: TemplateFingerprint{
			SectionCount: 1,
		},
		Fields: []TemplateContractField{
			{
				Key: "project.title",
				Selector: TemplateContractSelector{
					Type:  "placeholder",
					Value: "{{PROJECT_TITLE}}",
				},
			},
			{
				Key: "project.org",
				Selector: TemplateContractSelector{
					Type:       "anchor",
					Value:      "주관기관",
					TableLabel: "사업비 총괄표",
				},
			},
		},
		Tables: []TemplateContractTable{
			{
				Key: "participants",
				Selector: TemplateContractSelector{
					Type:       "anchor",
					Value:      "참여기관",
					TableLabel: "참여기관 표",
				},
				Columns: []TemplateContractColumn{
					{Key: "name", Source: "name"},
				},
				Expand: true,
			},
		},
	}
	payload := map[string]any{
		"project": map[string]any{
			"title": "프로젝트 Z",
			"org":   "예시 기관",
		},
		"participants": []any{
			map[string]any{"name": "기관1"},
			map[string]any{"name": "기관2"},
		},
	}

	replacements, err := CompileTemplateContract(contract, payload)
	if err != nil {
		t.Fatalf("compile contract: %v", err)
	}
	if len(replacements) != 3 {
		t.Fatalf("expected three replacements, got %+v", replacements)
	}
	if replacements[0].Placeholder != "{{PROJECT_TITLE}}" || replacements[0].Value != "프로젝트 Z" {
		t.Fatalf("unexpected placeholder replacement: %+v", replacements[0])
	}
	if replacements[1].Anchor != "주관기관" || replacements[1].TableLabel != "사업비 총괄표" || replacements[1].Value != "예시 기관" {
		t.Fatalf("unexpected anchor replacement: %+v", replacements[1])
	}
	if replacements[2].Mode != "table-down-records" || !replacements[2].Expand || len(replacements[2].Records) != 2 {
		t.Fatalf("unexpected table replacement: %+v", replacements[2])
	}
}

func TestCompileTemplateContractWithResolution(t *testing.T) {
	contract := TemplateContract{
		TemplateID:      "project_form_v1",
		TemplateVersion: "1.0.0",
		Fingerprint: TemplateFingerprint{
			SectionCount: 1,
		},
		Fields: []TemplateContractField{
			{
				Key: "project.title",
				Selector: TemplateContractSelector{
					Type:  "placeholder",
					Value: "{{PROJECT_TITLE}}",
				},
			},
			{
				Key:           "project.subtitle",
				FallbackValue: "기본 부제",
				Selector: TemplateContractSelector{
					Type:  "placeholder",
					Value: "{{PROJECT_SUBTITLE}}",
				},
			},
			{
				Key: "project.optional_note",
				Selector: TemplateContractSelector{
					Type:  "near-text",
					Value: "비고",
				},
			},
		},
	}
	payload := map[string]any{
		"project": map[string]any{
			"title": "프로젝트 Z",
		},
	}

	replacements, resolution, err := CompileTemplateContractWithResolution(contract, payload)
	if err != nil {
		t.Fatalf("compile contract with resolution: %v", err)
	}
	if len(replacements) != 2 {
		t.Fatalf("expected two replacements, got %+v", replacements)
	}
	if resolution.InputKind != "contract" || resolution.EntryCount != 3 || resolution.ResolvedCount != 2 || resolution.SkippedCount != 1 {
		t.Fatalf("unexpected resolution summary: %+v", resolution)
	}
	if resolution.Entries[1].UsedFallback != true || resolution.Entries[1].PayloadPath != "project.subtitle" {
		t.Fatalf("expected fallback resolution entry: %+v", resolution.Entries[1])
	}
	if !resolution.Entries[2].Skipped || resolution.Entries[2].SkipReason == "" {
		t.Fatalf("expected skipped optional entry: %+v", resolution.Entries[2])
	}
}
