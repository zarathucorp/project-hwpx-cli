package hwpx

import "testing"

func TestScaffoldTemplateContractFieldKey(t *testing.T) {
	tests := []struct {
		value    string
		fallback int
		want     string
	}{
		{value: "{{PROJECT_TITLE}}", fallback: 1, want: "project.title"},
		{value: "□ 주관기관_(주)00000", fallback: 2, want: "주관기관"},
		{value: "□ 참여기관_(주)00000  * 참여기관별로 작성", fallback: 3, want: "참여기관"},
		{value: "____", fallback: 4, want: "field_4"},
	}
	for _, tc := range tests {
		got := scaffoldTemplateContractFieldKey(tc.value, tc.fallback)
		if got != tc.want {
			t.Fatalf("scaffoldTemplateContractFieldKey(%q)=%q want %q", tc.value, got, tc.want)
		}
	}
}

func TestScaffoldTemplateContractTemplateID(t *testing.T) {
	got := scaffoldTemplateContractTemplateID("/tmp/붙임 3. 2026년 AI AGENT 양식.hwpx")
	if got != "붙임_3_2026년_ai_agent_양식" {
		t.Fatalf("unexpected scaffold template id: %q", got)
	}
}

func TestScaffoldTemplatePayload(t *testing.T) {
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
				Key:           "project.org",
				FallbackValue: "예시 기관",
				Selector: TemplateContractSelector{
					Type:  "placeholder",
					Value: "{{PROJECT_ORG}}",
				},
			},
		},
		Tables: []TemplateContractTable{
			{
				Key:  "participants",
				Mode: "table-down-records",
				Selector: TemplateContractSelector{
					Type:  "anchor",
					Value: "참여기관",
				},
				Columns: []TemplateContractColumn{
					{Key: "name", Source: "name"},
					{Key: "role", Source: "meta.role"},
				},
			},
		},
	}

	payload, err := ScaffoldTemplatePayload(contract)
	if err != nil {
		t.Fatalf("ScaffoldTemplatePayload returned error: %v", err)
	}

	project, ok := payload["project"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested project payload: %#v", payload)
	}
	if project["title"] != "" {
		t.Fatalf("expected empty title scaffold: %#v", project)
	}
	if project["org"] != "예시 기관" {
		t.Fatalf("expected fallback scaffold value: %#v", project)
	}

	participants, ok := payload["participants"].([]any)
	if !ok || len(participants) != 1 {
		t.Fatalf("expected participants record scaffold: %#v", payload["participants"])
	}
	record, ok := participants[0].(map[string]any)
	if !ok {
		t.Fatalf("expected participant record object: %#v", participants[0])
	}
	if record["name"] != "" {
		t.Fatalf("expected name scaffold value: %#v", record)
	}
	meta, ok := record["meta"].(map[string]any)
	if !ok || meta["role"] != "" {
		t.Fatalf("expected nested source scaffold value: %#v", record)
	}
}

func TestCurateScaffoldTemplateFingerprintTableLabels(t *testing.T) {
	labels := curateScaffoldTemplateFingerprintTableLabels([]TemplateTable{
		{LabelText: "사업비 소요명세"},
		{LabelText: "수행실적"},
		{LabelText: "□ 주관기관_(주)00000"},
		{LabelText: "1. 주관기관_(주)000"},
		{LabelText: "※ 하단의 표는 예시이며, 사업 성격에 따라 작성 서식을 변경하여 작성 가능"},
		{LabelText: "(단위 : 천원)"},
		{LabelText: "2026. 2. 27."},
		{LabelText: "ㅇ 2026년 정량·정성 목표"},
		{LabelText: "정보통신산업진흥원장 귀하"},
		{LabelText: "  사업비   소요명세  "},
	})

	if len(labels) != 3 {
		t.Fatalf("unexpected curated label count: %#v", labels)
	}
	if labels[0] != "사업비 소요명세" || labels[1] != "수행실적" || labels[2] != "정보통신산업진흥원장 귀하" {
		t.Fatalf("unexpected curated labels: %#v", labels)
	}
}
