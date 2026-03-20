package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTemplateContractYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contract.yaml")
	content := `
template_id: project_form_v1
template_version: 1.0.0
strict: true
fingerprint:
  section_count: 1
  section_paths:
    - Contents/section0.xml
fields:
  - key: project.title
    selector:
      type: placeholder
      value: "{{PROJECT_TITLE}}"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	contract, err := LoadTemplateContract(path)
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}

	if contract.TemplateID != "project_form_v1" {
		t.Fatalf("unexpected template id: %+v", contract)
	}
	if contract.TemplateVersion != "1.0.0" {
		t.Fatalf("unexpected template version: %+v", contract)
	}
	if !contract.Strict {
		t.Fatalf("expected strict contract: %+v", contract)
	}
	if len(contract.Fields) != 1 || contract.Fields[0].Selector.Type != "placeholder" {
		t.Fatalf("unexpected fields: %+v", contract.Fields)
	}
}

func TestLoadTemplateContractRejectsUnsupportedSelector(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid-contract.yaml")
	content := `
template_id: invalid_v1
template_version: 1.0.0
fingerprint:
  section_count: 1
fields:
  - key: project.title
    selector:
      type: bookmark
      value: project_title
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write invalid contract: %v", err)
	}

	_, err := LoadTemplateContract(path)
	if err == nil || !strings.Contains(err.Error(), "selector.type is not supported") {
		t.Fatalf("expected selector validation error, got %v", err)
	}
}

func TestVerifyTemplateContractFingerprint(t *testing.T) {
	analysis, err := analyzeTemplateEntries(templateContractTestEntries())
	if err != nil {
		t.Fatalf("analyze template entries: %v", err)
	}

	contract := TemplateContract{
		TemplateID:      "project_form_v1",
		TemplateVersion: "1.0.0",
		Fingerprint: TemplateFingerprint{
			SectionCount:      analysis.Fingerprint.SectionCount,
			SectionPaths:      append([]string{}, analysis.Fingerprint.SectionPaths...),
			TableLabels:       append([]string{}, analysis.Fingerprint.TableLabels...),
			PlaceholderTexts:  append([]string{}, analysis.Fingerprint.PlaceholderTexts...),
			PlaceholderDigest: analysis.Fingerprint.PlaceholderDigest,
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

	if err := VerifyTemplateContractFingerprint(contract, analysis); err != nil {
		t.Fatalf("verify matching fingerprint: %v", err)
	}

	contract.Fingerprint.TableLabels = []string{"없는 표"}
	err = VerifyTemplateContractFingerprint(contract, analysis)
	if err == nil || !strings.Contains(err.Error(), "tableLabels missing") {
		t.Fatalf("expected table label mismatch, got %v", err)
	}
}

func templateContractTestEntries() map[string][]byte {
	return map[string][]byte{
		"mimetype":            []byte("application/hwp+zip"),
		"version.xml":         []byte(`<version appVersion="1" hwpxVersion="1"/>`),
		"Contents/header.xml": []byte(`<hh:head xmlns:hh="http://www.hancom.co.kr/hwpml/2011/head" secCnt="1"></hh:head>`),
		"Contents/content.hpf": []byte(
			`<package><metadata><title>Contract Fixture</title></metadata><manifest>` +
				`<item id="header" href="Contents/header.xml" media-type="application/xml"></item>` +
				`<item id="section0" href="Contents/section0.xml" media-type="application/xml"></item>` +
				`</manifest><spine><itemref idref="header"></itemref><itemref idref="section0"></itemref></spine></package>`,
		),
		"Contents/section0.xml": []byte(
			`<hs:sec xmlns:hs="http://www.hancom.co.kr/hwpml/2011/section" xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph">` +
				`<hp:p><hp:run><hp:t>{{PROJECT_TITLE}}</hp:t></hp:run></hp:p>` +
				`<hp:p><hp:run><hp:t>사업비 총괄표</hp:t></hp:run></hp:p>` +
				`<hp:p><hp:run><hp:tbl rowCnt="1" colCnt="2">` +
				`<hp:tr>` +
				`<hp:tc><hp:cellAddr rowAddr="0" colAddr="0"></hp:cellAddr><hp:subList><hp:p><hp:run><hp:t>과제명</hp:t></hp:run></hp:p></hp:subList></hp:tc>` +
				`<hp:tc><hp:cellAddr rowAddr="0" colAddr="1"></hp:cellAddr><hp:subList><hp:p><hp:run><hp:t>프로젝트X</hp:t></hp:run></hp:p></hp:subList></hp:tc>` +
				`</hp:tr>` +
				`</hp:tbl></hp:run></hp:p>` +
				`</hs:sec>`,
		),
	}
}
