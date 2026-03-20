package hwpx

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func fixtureTemplateDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "test", "fixtures", "minimal")
}

func TestResolveFillTemplateInputWithMapping(t *testing.T) {
	workDir := t.TempDir()
	mappingPath := filepath.Join(workDir, "mapping.yaml")
	mapping := map[string]any{
		"schemaVersion": fillTemplateMappingSchemaVersion,
		"entries": []map[string]any{
			{
				"key":         "project.title",
				"note":        "title placeholder",
				"placeholder": "{{PROJECT_TITLE}}",
				"value":       "프로젝트 Z",
			},
		},
	}
	content, err := yaml.Marshal(mapping)
	if err != nil {
		t.Fatalf("marshal mapping: %v", err)
	}
	if err := os.WriteFile(mappingPath, content, 0o644); err != nil {
		t.Fatalf("write mapping: %v", err)
	}

	resolved, err := ResolveFillTemplateInput("", mappingPath, "", "")
	if err != nil {
		t.Fatalf("resolve mapping input: %v", err)
	}
	if resolved.InputKind != "mapping" || resolved.MappingPath == "" {
		t.Fatalf("unexpected mapping resolution: %+v", resolved)
	}
	if len(resolved.Replacements) != 1 || resolved.Replacements[0].SourceIndex == nil || *resolved.Replacements[0].SourceIndex != 0 {
		t.Fatalf("expected source-indexed replacements: %+v", resolved.Replacements)
	}
	if resolved.Replacements[0].Key != "project.title" || resolved.Replacements[0].Note != "title placeholder" {
		t.Fatalf("expected normalized mapping metadata: %+v", resolved.Replacements[0])
	}
	if resolved.Resolution.EntryCount != 1 || resolved.Resolution.InputKind != "mapping" {
		t.Fatalf("unexpected mapping resolution report: %+v", resolved.Resolution)
	}
	if resolved.Resolution.Entries[0].Key != "project.title" || resolved.Resolution.Entries[0].Note != "title placeholder" {
		t.Fatalf("expected mapping resolution metadata: %+v", resolved.Resolution.Entries[0])
	}
}

func TestResolveFillTemplateInputWithContract(t *testing.T) {
	workDir := t.TempDir()
	contractPath := filepath.Join(workDir, "contract.yaml")
	payloadPath := filepath.Join(workDir, "payload.json")

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
		},
	}
	contractContent, err := yaml.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	if err := os.WriteFile(contractPath, contractContent, 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	payload := map[string]any{
		"project": map[string]any{
			"title": "프로젝트 Z",
		},
	}
	payloadContent, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := os.WriteFile(payloadPath, payloadContent, 0o644); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	resolved, err := ResolveFillTemplateInput(fixtureTemplateDir(t), "", contractPath, payloadPath)
	if err != nil {
		t.Fatalf("resolve contract input: %v", err)
	}
	if resolved.InputKind != "contract" || resolved.TemplatePath == "" || resolved.PayloadPath == "" {
		t.Fatalf("unexpected contract resolution: %+v", resolved)
	}
	if len(resolved.Replacements) != 1 || resolved.Replacements[0].Value != "프로젝트 Z" {
		t.Fatalf("unexpected contract replacements: %+v", resolved.Replacements)
	}
	if resolved.Resolution.InputKind != "contract" || resolved.Resolution.EntryCount != 1 || resolved.Resolution.ResolvedCount != 1 {
		t.Fatalf("unexpected contract resolution report: %+v", resolved.Resolution)
	}
}

func TestResolveFillTemplateInputFingerprintMismatch(t *testing.T) {
	workDir := t.TempDir()
	contractPath := filepath.Join(workDir, "contract.yaml")
	payloadPath := filepath.Join(workDir, "payload.yaml")

	contract := TemplateContract{
		TemplateID:      "project_form_v1",
		TemplateVersion: "1.0.0",
		Fingerprint: TemplateFingerprint{
			SectionCount: 2,
		},
		Fields: []TemplateContractField{
			{
				Key: "project.title",
				Selector: TemplateContractSelector{
					Type:  "placeholder",
					Value: "{{PROJECT_TITLE}}",
				},
			},
		},
	}
	contractContent, err := yaml.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal mismatched contract: %v", err)
	}
	if err := os.WriteFile(contractPath, contractContent, 0o644); err != nil {
		t.Fatalf("write mismatched contract: %v", err)
	}
	if err := os.WriteFile(payloadPath, []byte("project:\n  title: 프로젝트 Z\n"), 0o644); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	_, err = ResolveFillTemplateInput(fixtureTemplateDir(t), "", contractPath, payloadPath)
	if err == nil {
		t.Fatal("expected fingerprint mismatch error")
	}

	var inputErr *FillTemplateInputError
	if !errors.As(err, &inputErr) {
		t.Fatalf("expected fill template input error, got %T", err)
	}
	if inputErr.Kind != "template_contract_mismatch" {
		t.Fatalf("expected template_contract_mismatch, got %q", inputErr.Kind)
	}
}

func TestResolveFillTemplateInputInvalidMappingSchemaVersion(t *testing.T) {
	workDir := t.TempDir()
	mappingPath := filepath.Join(workDir, "mapping.yaml")
	if err := os.WriteFile(mappingPath, []byte("schemaVersion: invalid\nentries:\n  - placeholder: \"{{PROJECT_TITLE}}\"\n    value: \"프로젝트 Z\"\n"), 0o644); err != nil {
		t.Fatalf("write mapping: %v", err)
	}

	_, err := ResolveFillTemplateInput("", mappingPath, "", "")
	if err == nil {
		t.Fatal("expected invalid schema version error")
	}

	var inputErr *FillTemplateInputError
	if !errors.As(err, &inputErr) {
		t.Fatalf("expected fill template input error, got %T", err)
	}
	if inputErr.Kind != "invalid_arguments" {
		t.Fatalf("expected invalid_arguments, got %q", inputErr.Kind)
	}
}
