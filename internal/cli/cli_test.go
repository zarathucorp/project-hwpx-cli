package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "test", "fixtures", "minimal")
}

func fixtureArchive(t *testing.T) string {
	t.Helper()
	workDir := t.TempDir()
	archivePath := filepath.Join(workDir, "fixture.hwpx")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"pack", fixtureDir(t), "--output", archivePath}, stdout, stderr); err != nil {
		t.Fatalf("pack fixture: %v stderr=%s", err, stderr.String())
	}

	return archivePath
}

func TestTextJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"text", archivePath, "--format", "json"}, stdout, stderr)
	if err != nil {
		t.Fatalf("text json should succeed: %v stderr=%s", err, stderr.String())
	}

	var envelope struct {
		Command string `json:"command"`
		Success bool   `json:"success"`
		Data    struct {
			Text           string `json:"text"`
			LineCount      int    `json:"lineCount"`
			CharacterCount int    `json:"characterCount"`
		} `json:"data"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode envelope: %v", decodeErr)
	}

	if !envelope.Success || envelope.Command != "text" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if envelope.Data.Text != "Hello HWPX\nSecond paragraph" {
		t.Fatalf("unexpected text payload: %q", envelope.Data.Text)
	}
	if envelope.Data.LineCount != 2 {
		t.Fatalf("unexpected line count: %d", envelope.Data.LineCount)
	}
}

func TestValidateJSONFailure(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "mimetype"), []byte("application/hwp+zip"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"validate", tempDir, "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("validate should fail for invalid directory")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
		Data struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode envelope: %v", decodeErr)
	}

	if envelope.Success {
		t.Fatal("validate failure should have success=false")
	}
	if envelope.Error.Code != "validation_failed" {
		t.Fatalf("unexpected error code: %s", envelope.Error.Code)
	}
	if envelope.Data.Report.Valid {
		t.Fatal("invalid report should remain invalid")
	}
}

func TestSchemaCommandReturnsJSON(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"schema"}, stdout, stderr)
	if err != nil {
		t.Fatalf("schema should succeed: %v stderr=%s", err, stderr.String())
	}

	var document struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Environment []any  `json:"environment"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &document); decodeErr != nil {
		t.Fatalf("decode schema: %v", decodeErr)
	}

	if document.Name != "hwpxctl" {
		t.Fatalf("unexpected schema name: %s", document.Name)
	}
	if document.Version == "" || len(document.Environment) == 0 {
		t.Fatalf("schema should include version and environment: %+v", document)
	}
}

func TestSubcommandHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"inspect", "--help"}, stdout, stderr)
	if err != nil {
		t.Fatalf("inspect help should succeed: %v stderr=%s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Inspect HWPX metadata") {
		t.Fatalf("unexpected help output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--format") {
		t.Fatalf("help should include inherited format flag: %s", stdout.String())
	}
}

func TestUnknownCommandJSONFailure(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"missing-command", "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("unknown command should fail")
	}

	var envelope struct {
		Command string `json:"command"`
		Success bool   `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode envelope: %v", decodeErr)
	}

	if envelope.Command != "missing-command" {
		t.Fatalf("unexpected command: %s", envelope.Command)
	}
	if envelope.Success {
		t.Fatal("unknown command should have success=false")
	}
	if envelope.Error.Code != "unknown_command" {
		t.Fatalf("unexpected error code: %s", envelope.Error.Code)
	}
}

func TestUnpackJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr)
	if err != nil {
		t.Fatalf("unpack json should succeed: %v stderr=%s", err, stderr.String())
	}

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			OutputPath string `json:"outputPath"`
			Report     struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode unpack envelope: %v", decodeErr)
	}

	if !envelope.Success || !envelope.Data.Report.Valid {
		t.Fatalf("unexpected unpack envelope: %+v", envelope)
	}
	if envelope.Data.OutputPath == "" {
		t.Fatal("json output should include outputPath")
	}
}

func TestExportWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")
	markdownPath := filepath.Join(workDir, "result.md")
	htmlPath := filepath.Join(workDir, "result.html")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "제목\n본문 문단", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,값;이름,홍길동", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	markdownStdout := runCLI(t, "export-markdown", archivePath, "--output", markdownPath, "--format", "json")
	htmlStdout := runCLI(t, "export-html", archivePath, "--output", htmlPath, "--format", "json")

	var markdownEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			OutputPath string `json:"outputPath"`
			BlockCount int    `json:"blockCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(markdownStdout.Bytes(), &markdownEnvelope); err != nil {
		t.Fatalf("decode markdown response: %v", err)
	}
	if !markdownEnvelope.Success || markdownEnvelope.Data.OutputPath == "" || markdownEnvelope.Data.BlockCount < 3 {
		t.Fatalf("unexpected markdown response: %s", markdownStdout.String())
	}

	var htmlEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			OutputPath string `json:"outputPath"`
			BlockCount int    `json:"blockCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(htmlStdout.Bytes(), &htmlEnvelope); err != nil {
		t.Fatalf("decode html response: %v", err)
	}
	if !htmlEnvelope.Success || htmlEnvelope.Data.OutputPath == "" || htmlEnvelope.Data.BlockCount < 3 {
		t.Fatalf("unexpected html response: %s", htmlStdout.String())
	}

	markdownBytes, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown export: %v", err)
	}
	markdownText := string(markdownBytes)
	for _, needle := range []string{"제목", "본문 문단", "| 항목 | 값 |"} {
		if !strings.Contains(markdownText, needle) {
			t.Fatalf("expected %q in markdown export: %s", needle, markdownText)
		}
	}

	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read html export: %v", err)
	}
	htmlText := string(htmlBytes)
	for _, needle := range []string{"<table>", "<p>제목</p>", "<td>홍길동</td>"} {
		if !strings.Contains(htmlText, needle) {
			t.Fatalf("expected %q in html export: %s", needle, htmlText)
		}
	}
}

func TestCreateEditPackWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	imagePath := filepath.Join(workDir, "pixel.png")
	archivePath := filepath.Join(workDir, "result.hwpx")

	if err := os.WriteFile(imagePath, mustTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,내용;이름,홍길동", "--format", "json")
	runCLI(t, "set-table-cell", editableDir, "--table", "0", "--row", "1", "--col", "1", "--text", "김영희", "--format", "json")

	embedStdout := runCLI(t, "embed-image", editableDir, "--image", imagePath, "--format", "json")
	var embedEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			ItemID     string `json:"itemId"`
			BinaryPath string `json:"binaryPath"`
		} `json:"data"`
	}
	if err := json.Unmarshal(embedStdout.Bytes(), &embedEnvelope); err != nil {
		t.Fatalf("decode embed-image response: %v", err)
	}
	if !embedEnvelope.Success || embedEnvelope.Data.ItemID == "" || embedEnvelope.Data.BinaryPath == "" {
		t.Fatalf("unexpected embed response: %s", embedStdout.String())
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if want := "첫 문단\n둘째 문단\n항목\n내용\n이름\n김영희"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid   bool `json:"valid"`
				Summary struct {
					BinaryPath []string `json:"binaryPaths"`
				} `json:"summary"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
	if len(inspectEnvelope.Data.Report.Summary.BinaryPath) != 1 {
		t.Fatalf("expected one embedded binary path: %v", inspectEnvelope.Data.Report.Summary.BinaryPath)
	}
}

func TestParagraphEditWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단\n셋째 문단", "--format", "json")
	runCLI(t, "set-paragraph-text", editableDir, "--paragraph", "1", "--text", "수정된 둘째 문단", "--format", "json")
	runCLI(t, "delete-paragraph", editableDir, "--paragraph", "0", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Contains(sectionText, "첫 문단") {
		t.Fatalf("expected deleted paragraph to be removed: %s", sectionText)
	}
	if !strings.Contains(sectionText, "수정된 둘째 문단") {
		t.Fatalf("expected updated paragraph text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "수정된 둘째 문단\n셋째 문단"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestAddRunTextWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "add-run-text", editableDir, "--paragraph", "1", "--text", " (검토본)", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"둘째 문단", "(검토본)"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}
	if strings.Count(sectionText, "<hp:run") < 4 {
		t.Fatalf("expected inserted run in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "첫 문단\n둘째 문단 (검토본)"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestAppendTextTrackChangesWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "추적 문단", "--track-changes", "true", "--change-author", "tester", "--change-summary", "Added tracked paragraph", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	contentBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "content.hpf"))
	if err != nil {
		t.Fatalf("read content.hpf: %v", err)
	}
	contentText := string(contentBytes)
	if !strings.Contains(contentText, "Contents/history.xml") {
		t.Fatalf("expected history manifest item: %s", contentText)
	}

	historyBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "history.xml"))
	if err != nil {
		t.Fatalf("read history.xml: %v", err)
	}
	historyText := string(historyBytes)
	for _, needle := range []string{
		"command=\"append-text\"",
		"author=\"tester\"",
		"Added tracked paragraph",
	} {
		if !strings.Contains(historyText, needle) {
			t.Fatalf("expected %q in history xml: %s", needle, historyText)
		}
	}

	validateStdout := runCLI(t, "validate", archivePath, "--format", "json")
	var validateEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(validateStdout.Bytes(), &validateEnvelope); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if !validateEnvelope.Success || !validateEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected validate response: %s", validateStdout.String())
	}
}

func TestSetRunTextWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "add-run-text", editableDir, "--paragraph", "1", "--text", " (검토본)", "--format", "json")
	runCLI(t, "set-run-text", editableDir, "--paragraph", "1", "--run", "1", "--text", " (최종본)", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Contains(sectionText, "(검토본)") {
		t.Fatalf("expected previous run text to be replaced: %s", sectionText)
	}
	if !strings.Contains(sectionText, "(최종본)") {
		t.Fatalf("expected updated run text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "첫 문단\n둘째 문단 (최종본)"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestSetTextStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "set-text-style", editableDir, "--paragraph", "1", "--bold", "true", "--italic", "true", "--underline", "true", "--text-color", "#C00000", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	headerBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "header.xml"))
	if err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	headerText := string(headerBytes)
	for _, needle := range []string{
		"textColor=\"#C00000\"",
		"<hh:bold",
		"<hh:italic",
		"<hh:underline type=\"BOTTOM\"",
	} {
		if !strings.Contains(headerText, needle) {
			t.Fatalf("expected %q in header xml: %s", needle, headerText)
		}
	}

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "둘째 문단") {
		t.Fatalf("expected target paragraph text in section xml: %s", sectionText)
	}
	if strings.Count(sectionText, "charPrIDRef=\"0\"") < 1 {
		t.Fatalf("expected at least one default run to remain: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "첫 문단\n둘째 문단"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestFindRunsByStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "set-text-style", editableDir, "--paragraph", "1", "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--format", "json")

	searchStdout := runCLI(t, "find-runs-by-style", editableDir, "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--format", "json")
	var searchEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Paragraph int    `json:"paragraph"`
				Run       int    `json:"run"`
				Text      string `json:"text"`
				Bold      bool   `json:"bold"`
				Underline bool   `json:"underline"`
				TextColor string `json:"textColor"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchStdout.Bytes(), &searchEnvelope); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if !searchEnvelope.Success || searchEnvelope.Data.Count == 0 {
		t.Fatalf("unexpected search response: %s", searchStdout.String())
	}
	match := searchEnvelope.Data.Matches[0]
	if match.Paragraph != 1 || match.Text != "둘째 문단" || !match.Bold || !match.Underline || match.TextColor != "#C00000" {
		t.Fatalf("unexpected match: %+v", match)
	}
}

func TestReplaceRunsByStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "set-text-style", editableDir, "--paragraph", "1", "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--format", "json")

	replaceStdout := runCLI(t, "replace-runs-by-style", editableDir, "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--text", "[강조]", "--format", "json")
	var replaceEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count        int `json:"count"`
			Replacements []struct {
				Paragraph    int    `json:"paragraph"`
				Run          int    `json:"run"`
				PreviousText string `json:"previousText"`
				Text         string `json:"text"`
				CharPrIDRef  string `json:"charPrIdRef"`
			} `json:"replacements"`
		} `json:"data"`
	}
	if err := json.Unmarshal(replaceStdout.Bytes(), &replaceEnvelope); err != nil {
		t.Fatalf("decode replace response: %v", err)
	}
	if !replaceEnvelope.Success || replaceEnvelope.Data.Count != 1 {
		t.Fatalf("unexpected replace response: %s", replaceStdout.String())
	}
	replacement := replaceEnvelope.Data.Replacements[0]
	if replacement.Paragraph != 1 || replacement.Run != 0 || replacement.PreviousText != "둘째 문단" || replacement.Text != "[강조]" {
		t.Fatalf("unexpected replacement: %+v", replacement)
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Contains(sectionText, "둘째 문단") {
		t.Fatalf("expected previous run text to be replaced: %s", sectionText)
	}
	if !strings.Contains(sectionText, "[강조]") {
		t.Fatalf("expected replacement run text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "첫 문단\n[강조]"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestFindObjectsWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	imagePath := filepath.Join(workDir, "pixel.png")

	if err := os.WriteFile(imagePath, mustTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "insert-image", editableDir, "--image", imagePath, "--width-mm", "20", "--format", "json")
	runCLI(t, "add-equation", editableDir, "--script", "a+b", "--format", "json")
	runCLI(t, "add-rectangle", editableDir, "--width-mm", "40", "--height-mm", "20", "--format", "json")
	runCLI(t, "add-line", editableDir, "--width-mm", "50", "--height-mm", "10", "--format", "json")
	runCLI(t, "add-ellipse", editableDir, "--width-mm", "40", "--height-mm", "20", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--format", "json")

	searchStdout := runCLI(t, "find-objects", editableDir, "--format", "json")
	var searchEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Type      string `json:"type"`
				Paragraph int    `json:"paragraph"`
				Run       int    `json:"run"`
				Path      string `json:"path"`
				Ref       string `json:"ref"`
				Text      string `json:"text"`
				Rows      int    `json:"rows"`
				Cols      int    `json:"cols"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchStdout.Bytes(), &searchEnvelope); err != nil {
		t.Fatalf("decode object search response: %v", err)
	}
	if !searchEnvelope.Success || searchEnvelope.Data.Count != 8 {
		t.Fatalf("unexpected object search response: %s", searchStdout.String())
	}

	types := map[string]int{}
	var sawNestedTable bool
	var sawImageRef bool
	var sawTextboxText bool
	for _, match := range searchEnvelope.Data.Matches {
		types[match.Type]++
		if match.Type == "table" && match.Rows == 2 && match.Cols == 2 && strings.Contains(match.Text, "내부1") {
			sawNestedTable = true
		}
		if match.Type == "image" && match.Ref != "" {
			sawImageRef = true
		}
		if match.Type == "textbox" && strings.Contains(match.Text, "글상자 첫 줄") {
			sawTextboxText = true
		}
	}
	for _, objectType := range []string{"table", "image", "equation", "rectangle", "line", "ellipse", "textbox"} {
		if types[objectType] == 0 {
			t.Fatalf("expected object type %q in matches: %+v", objectType, types)
		}
	}
	if types["table"] != 2 || !sawNestedTable || !sawImageRef || !sawTextboxText {
		t.Fatalf("unexpected object details: %+v", searchEnvelope.Data.Matches)
	}

	filteredStdout := runCLI(t, "find-objects", editableDir, "--type", "table,textbox", "--format", "json")
	var filteredEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Type string `json:"type"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(filteredStdout.Bytes(), &filteredEnvelope); err != nil {
		t.Fatalf("decode filtered object search response: %v", err)
	}
	if !filteredEnvelope.Success || filteredEnvelope.Data.Count != 3 {
		t.Fatalf("unexpected filtered object search response: %s", filteredStdout.String())
	}
	for _, match := range filteredEnvelope.Data.Matches {
		if match.Type != "table" && match.Type != "textbox" {
			t.Fatalf("unexpected filtered object type: %+v", match)
		}
	}
}

func TestFindByTagWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--format", "json")

	tableStdout := runCLI(t, "find-by-tag", editableDir, "--tag", "hp:tbl", "--format", "json")
	var tableEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag  string `json:"tag"`
				Text string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(tableStdout.Bytes(), &tableEnvelope); err != nil {
		t.Fatalf("decode tag search response: %v", err)
	}
	if !tableEnvelope.Success || tableEnvelope.Data.Count != 2 {
		t.Fatalf("unexpected table tag search response: %s", tableStdout.String())
	}
	for _, match := range tableEnvelope.Data.Matches {
		if !strings.HasSuffix(match.Tag, "tbl") {
			t.Fatalf("unexpected table tag: %+v", match)
		}
	}

	drawTextStdout := runCLI(t, "find-by-tag", editableDir, "--tag", "drawText", "--format", "json")
	var drawTextEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag  string `json:"tag"`
				Text string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(drawTextStdout.Bytes(), &drawTextEnvelope); err != nil {
		t.Fatalf("decode drawText search response: %v", err)
	}
	if !drawTextEnvelope.Success || drawTextEnvelope.Data.Count != 1 {
		t.Fatalf("unexpected drawText search response: %s", drawTextStdout.String())
	}
	if !strings.HasSuffix(drawTextEnvelope.Data.Matches[0].Tag, "drawText") || !strings.Contains(drawTextEnvelope.Data.Matches[0].Text, "글상자 첫 줄") {
		t.Fatalf("unexpected drawText match: %+v", drawTextEnvelope.Data.Matches[0])
	}
}

func TestFindByAttrWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--format", "json")

	idStdout := runCLI(t, "find-by-attr", editableDir, "--attr", "id", "--tag", "tbl", "--format", "json")
	var idEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag   string `json:"tag"`
				Attr  string `json:"attr"`
				Value string `json:"value"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(idStdout.Bytes(), &idEnvelope); err != nil {
		t.Fatalf("decode attr search response: %v", err)
	}
	if !idEnvelope.Success || idEnvelope.Data.Count != 2 {
		t.Fatalf("unexpected id attr search response: %s", idStdout.String())
	}
	for _, match := range idEnvelope.Data.Matches {
		if !strings.HasSuffix(match.Tag, "tbl") || match.Attr != "id" || strings.TrimSpace(match.Value) == "" {
			t.Fatalf("unexpected id attr match: %+v", match)
		}
	}

	editableStdout := runCLI(t, "find-by-attr", editableDir, "--attr", "editable", "--tag", "drawText", "--value", "0", "--format", "json")
	var editableEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag   string `json:"tag"`
				Attr  string `json:"attr"`
				Value string `json:"value"`
				Text  string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(editableStdout.Bytes(), &editableEnvelope); err != nil {
		t.Fatalf("decode editable attr search response: %v", err)
	}
	if !editableEnvelope.Success || editableEnvelope.Data.Count != 1 {
		t.Fatalf("unexpected editable attr search response: %s", editableStdout.String())
	}
	match := editableEnvelope.Data.Matches[0]
	if !strings.HasSuffix(match.Tag, "drawText") || match.Attr != "editable" || match.Value != "0" || !strings.Contains(match.Text, "글상자 첫 줄") {
		t.Fatalf("unexpected editable attr match: %+v", match)
	}
}

func TestFindByXPathWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--format", "json")

	tableStdout := runCLI(t, "find-by-xpath", editableDir, "--expr", ".//hp:tbl[@id]", "--format", "json")
	var tableEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Paragraph int    `json:"paragraph"`
				Run       int    `json:"run"`
				Tag       string `json:"tag"`
				Text      string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(tableStdout.Bytes(), &tableEnvelope); err != nil {
		t.Fatalf("decode xpath search response: %v", err)
	}
	if !tableEnvelope.Success || tableEnvelope.Data.Count != 2 {
		t.Fatalf("unexpected table xpath response: %s", tableStdout.String())
	}
	for _, match := range tableEnvelope.Data.Matches {
		if !strings.HasSuffix(match.Tag, "tbl") || match.Paragraph != 0 || match.Run != 0 {
			t.Fatalf("unexpected table xpath match: %+v", match)
		}
	}

	drawTextStdout := runCLI(t, "find-by-xpath", editableDir, "--expr", ".//hp:drawText[@editable='0']", "--format", "json")
	var drawTextEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag  string `json:"tag"`
				Text string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(drawTextStdout.Bytes(), &drawTextEnvelope); err != nil {
		t.Fatalf("decode drawText xpath response: %v", err)
	}
	if !drawTextEnvelope.Success || drawTextEnvelope.Data.Count != 1 {
		t.Fatalf("unexpected drawText xpath response: %s", drawTextStdout.String())
	}
	if !strings.HasSuffix(drawTextEnvelope.Data.Matches[0].Tag, "drawText") || !strings.Contains(drawTextEnvelope.Data.Matches[0].Text, "글상자 첫 줄") {
		t.Fatalf("unexpected drawText xpath match: %+v", drawTextEnvelope.Data.Matches[0])
	}
}

func TestSectionAddAndDeleteWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 section 본문", "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")
	runCLI(t, "delete-section", editableDir, "--section", "1", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	contentBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "content.hpf"))
	if err != nil {
		t.Fatalf("read content.hpf: %v", err)
	}
	contentText := string(contentBytes)
	if !strings.Contains(contentText, "section1.xml") {
		t.Fatalf("expected remaining section to be renumbered: %s", contentText)
	}
	if strings.Contains(contentText, "section2.xml") {
		t.Fatalf("expected stale section path to be removed: %s", contentText)
	}

	headerBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "header.xml"))
	if err != nil {
		t.Fatalf("read header.xml: %v", err)
	}
	if !strings.Contains(string(headerBytes), "secCnt=\"2\"") {
		t.Fatalf("expected header section count to be updated: %s", string(headerBytes))
	}

	if _, err := os.Stat(filepath.Join(editableDir, "Contents", "section2.xml")); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected stale section file to be removed: %v", err)
	}

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section1.xml"))
	if err != nil {
		t.Fatalf("read new section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"<hp:secPr", "<hp:t"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in new section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid   bool `json:"valid"`
				Summary struct {
					SectionPath []string `json:"sectionPaths"`
				} `json:"summary"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
	if want := []string{"Contents/section0.xml", "Contents/section1.xml"}; strings.Join(inspectEnvelope.Data.Report.Summary.SectionPath, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected section paths: %v", inspectEnvelope.Data.Report.Summary.SectionPath)
	}
}

func TestInsertImageCreatesVisiblePictureXML(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	imagePath := filepath.Join(workDir, "pixel.png")
	archivePath := filepath.Join(workDir, "result.hwpx")

	if err := os.WriteFile(imagePath, mustTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	insertStdout := runCLI(t, "insert-image", editableDir, "--image", imagePath, "--width-mm", "40", "--format", "json")

	var insertEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			ItemID       string `json:"itemId"`
			PlacedWidth  int    `json:"placedWidth"`
			PlacedHeight int    `json:"placedHeight"`
		} `json:"data"`
	}
	if err := json.Unmarshal(insertStdout.Bytes(), &insertEnvelope); err != nil {
		t.Fatalf("decode insert-image response: %v", err)
	}
	if !insertEnvelope.Success || insertEnvelope.Data.ItemID == "" {
		t.Fatalf("unexpected insert-image response: %s", insertStdout.String())
	}
	if insertEnvelope.Data.PlacedWidth <= 0 || insertEnvelope.Data.PlacedHeight <= 0 {
		t.Fatalf("unexpected inserted size: %+v", insertEnvelope.Data)
	}

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "<hp:pic") {
		t.Fatalf("expected visible picture xml: %s", sectionText)
	}
	if !strings.Contains(sectionText, "binaryItemIDRef=\""+insertEnvelope.Data.ItemID+"\"") {
		t.Fatalf("expected picture to reference embedded image id: %s", sectionText)
	}

	contentBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "content.hpf"))
	if err != nil {
		t.Fatalf("read content.hpf: %v", err)
	}
	contentText := string(contentBytes)
	if !strings.Contains(contentText, "isEmbeded=\"1\"") {
		t.Fatalf("expected embedded image manifest flag: %s", contentText)
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestHeaderFooterAndPageNumberWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "set-header", editableDir, "--text", "문서 머리말", "--format", "json")
	runCLI(t, "set-footer", editableDir, "--text", "문서 꼬리말", "--format", "json")
	runCLI(t, "set-page-number", editableDir, "--position", "BOTTOM_CENTER", "--type", "DIGIT", "--side-char", "-", "--start-page", "3", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:header",
		"문서 머리말",
		"<hp:footer",
		"문서 꼬리말",
		"<hp:pageNum",
		"pos=\"BOTTOM_CENTER\"",
		"formatType=\"DIGIT\"",
		"sideChar=\"-\"",
		"pageStartsOn=\"BOTH\"",
		"page=\"3\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestRemoveHeaderFooterPreservesPageNumber(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "set-header", editableDir, "--text", "문서 머리말", "--format", "json")
	runCLI(t, "set-footer", editableDir, "--text", "문서 꼬리말", "--format", "json")
	runCLI(t, "set-page-number", editableDir, "--position", "BOTTOM_CENTER", "--type", "DIGIT", "--format", "json")
	runCLI(t, "remove-header", editableDir, "--format", "json")
	runCLI(t, "remove-footer", editableDir, "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"<hp:header", "<hp:footer", "문서 머리말", "문서 꼬리말"} {
		if strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q to be removed from section xml: %s", needle, sectionText)
		}
	}
	for _, needle := range []string{"<hp:pageNum", "pos=\"BOTTOM_CENTER\"", "formatType=\"DIGIT\""} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q to remain in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestSetColumnsWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "set-columns", editableDir, "--count", "2", "--gap-mm", "8", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"spaceColumns=\"2268\"",
		"<hp:colPr",
		"colCount=\"2\"",
		"sameGap=\"2268\"",
		"<hp:colLine",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestTableMergeAndSplitWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "merge-table-cells", editableDir, "--table", "0", "--start-row", "0", "--start-col", "0", "--end-row", "1", "--end-col", "1", "--format", "json")
	runCLI(t, "set-table-cell", editableDir, "--table", "0", "--row", "1", "--col", "1", "--text", "병합 셀", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"rowSpan=\"2\"",
		"colSpan=\"2\"",
		"병합 셀",
		"width=\"0\"",
		"height=\"0\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after merge: %s", needle, sectionText)
		}
	}

	runCLI(t, "split-table-cell", editableDir, "--table", "0", "--row", "0", "--col", "0", "--format", "json")
	runCLI(t, "set-table-cell", editableDir, "--table", "0", "--row", "1", "--col", "1", "--text", "분리 후 D", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err = os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml after split: %v", err)
	}
	sectionText = string(sectionBytes)
	if strings.Contains(sectionText, "rowSpan=\"2\"") || strings.Contains(sectionText, "colSpan=\"2\"") {
		t.Fatalf("expected merged span to be removed after split: %s", sectionText)
	}
	if !strings.Contains(sectionText, "분리 후 D") {
		t.Fatalf("expected split cell text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{"병합 셀", "분리 후 D"} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestNestedTableWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Count(sectionText, "<hp:tbl") != 2 {
		t.Fatalf("expected outer and nested table in section xml: %s", sectionText)
	}
	for _, needle := range []string{"내부1", "내부4"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{"A", "D", "내부1", "내부4"} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestFooterSupportsInlinePageTokens(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "set-footer", editableDir, "--text", "- {{PAGE}} / {{TOTAL_PAGE}} -", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:footer",
		"<hp:autoNum num=\"1\" numType=\"PAGE\">",
		"<hp:autoNum num=\"1\" numType=\"TOTAL_PAGE\">",
		"<hp:autoNumFormat type=\"DIGIT\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}
}

func TestFootnoteAndEndnoteWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-footnote", editableDir, "--anchor-text", "각주가 있는 본문", "--text", "첫 번째 각주", "--format", "json")
	runCLI(t, "add-endnote", editableDir, "--anchor-text", "미주가 있는 본문", "--text", "첫 번째 미주", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:footNote",
		"<hp:endNote",
		"각주가 있는 본문",
		"첫 번째 각주",
		"미주가 있는 본문",
		"첫 번째 미주",
		"numType=\"FOOTNOTE\"",
		"numType=\"ENDNOTE\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"각주가 있는 본문",
		"첫 번째 각주",
		"미주가 있는 본문",
		"첫 번째 미주",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestMemoWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-memo", editableDir, "--anchor-text", "검토가 필요한 문장", "--text", "첫 번째 메모\n두 번째 메모", "--author", "홍길동", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	headerBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "header.xml"))
	if err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	headerText := string(headerBytes)
	for _, needle := range []string{
		"<hh:memoProperties",
		"<hh:memoPr id=\"0\"",
		"fillColor=\"#CCFF99\"",
	} {
		if !strings.Contains(headerText, needle) {
			t.Fatalf("expected %q in header xml: %s", needle, headerText)
		}
	}

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:memogroup>",
		"<hp:memo id=\"",
		"memoShapeIDRef=\"0\"",
		"type=\"MEMO\"",
		"<hp:stringParam name=\"Author\">홍길동</hp:stringParam>",
		"<hp:stringParam name=\"MemoShapeID\">0</hp:stringParam>",
		"검토가 필요한 문장",
		"첫 번째 메모",
		"두 번째 메모",
		"<hp:fieldEnd",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"검토가 필요한 문장",
		"첫 번째 메모",
		"두 번째 메모",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestBookmarkAndHyperlinkWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-bookmark", editableDir, "--name", "intro", "--text", "소개 위치", "--format", "json")
	runCLI(t, "add-hyperlink", editableDir, "--target", "#intro", "--text", "소개로 이동", "--format", "json")
	runCLI(t, "add-hyperlink", editableDir, "--target", "https://example.com", "--text", "외부 링크", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:bookmark name=\"intro\"",
		"type=\"HYPERLINK\"",
		"name=\"#intro\"",
		"name=\"https://example.com\"",
		"fieldid=\"",
		"<hp:parameters count=\"1\" name=\"\">",
		"<hp:stringParam name=\"Command\">#intro</hp:stringParam>",
		"<hp:stringParam name=\"Command\">https://example.com</hp:stringParam>",
		"<hp:fieldEnd",
		"소개 위치",
		"소개로 이동",
		"외부 링크",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"소개 위치",
		"소개로 이동",
		"외부 링크",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestHeadingTOCAndCrossReferenceWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")

	headingStdout := runCLI(t, "add-heading", editableDir, "--kind", "heading", "--level", "1", "--text", "소개", "--format", "json")
	outlineStdout := runCLI(t, "add-heading", editableDir, "--kind", "outline", "--level", "2", "--text", "세부 항목", "--format", "json")

	var headingEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			BookmarkName string `json:"bookmarkName"`
		} `json:"data"`
	}
	if err := json.Unmarshal(headingStdout.Bytes(), &headingEnvelope); err != nil {
		t.Fatalf("decode add-heading response: %v", err)
	}
	if !headingEnvelope.Success || headingEnvelope.Data.BookmarkName == "" {
		t.Fatalf("unexpected heading response: %s", headingStdout.String())
	}

	var outlineEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			BookmarkName string `json:"bookmarkName"`
		} `json:"data"`
	}
	if err := json.Unmarshal(outlineStdout.Bytes(), &outlineEnvelope); err != nil {
		t.Fatalf("decode outline response: %v", err)
	}
	if !outlineEnvelope.Success || outlineEnvelope.Data.BookmarkName == "" {
		t.Fatalf("unexpected outline response: %s", outlineStdout.String())
	}

	runCLI(t, "insert-toc", editableDir, "--title", "목차", "--max-level", "2", "--format", "json")
	runCLI(t, "add-cross-reference", editableDir, "--bookmark", headingEnvelope.Data.BookmarkName, "--text", "소개로 이동", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"목차",
		"소개",
		"세부 항목",
		"소개로 이동",
		headingEnvelope.Data.BookmarkName,
		outlineEnvelope.Data.BookmarkName,
		"#" + headingEnvelope.Data.BookmarkName,
		"#" + outlineEnvelope.Data.BookmarkName,
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}
	if strings.Count(sectionText, "type=\"HYPERLINK\"") < 3 {
		t.Fatalf("expected toc and cross reference hyperlinks in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"목차",
		"소개",
		"세부 항목",
		"소개로 이동",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestEquationWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-equation", editableDir, "--script", "a+b", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:equation",
		"numberingType=\"EQUATION\"",
		"version=\"Equation Version 60\"",
		"<hp:script>a+b</hp:script>",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if !strings.Contains(textEnvelope.Data.Text, "a+b") {
		t.Fatalf("expected equation script in extracted text: %s", textEnvelope.Data.Text)
	}
}

func TestRectangleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-rectangle", editableDir, "--width-mm", "40", "--height-mm", "20", "--fill-color", "#FFF2CC", "--line-color", "#333333", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:rect",
		"numberingType=\"NONE\"",
		"<hp:lineShape color=\"#333333\"",
		"faceColor=\"#FFF2CC\"",
		"<hc:pt2",
		"<hp:sz width=\"",
		"<hp:pos treatAsChar=\"1\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestLineWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-line", editableDir, "--width-mm", "50", "--height-mm", "10", "--line-color", "#2F5597", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:line",
		"numberingType=\"NONE\"",
		"<hp:lineShape color=\"#2F5597\"",
		"<hc:startPt x=\"0\" y=\"0\">",
		"<hc:endPt x=\"",
		"<hp:sz width=\"",
		"<hp:pos treatAsChar=\"1\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestEllipseWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-ellipse", editableDir, "--width-mm", "40", "--height-mm", "20", "--fill-color", "#FFF2CC", "--line-color", "#333333", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:ellipse",
		"hasArcPr=\"false\"",
		"arcType=\"Normal\"",
		"<hc:center x=\"",
		"<hc:ax1 x=\"",
		"<hc:ax2 x=\"",
		"<hp:lineShape color=\"#333333\"",
		"faceColor=\"#FFF2CC\"",
		"<hp:sz width=\"",
		"<hp:pos treatAsChar=\"1\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestTextBoxWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--fill-color", "#FFF2CC", "--line-color", "#333333", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:rect",
		"<hp:drawText",
		"<hp:textMargin left=\"283\"",
		"<hp:subList",
		"글상자 첫 줄",
		"글상자 둘째 줄",
		"faceColor=\"#FFF2CC\"",
		"<hp:lineShape color=\"#333333\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"글상자 첫 줄",
		"글상자 둘째 줄",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestSetParagraphLayoutWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")

	layoutStdout := runCLI(
		t,
		"set-paragraph-layout", editableDir,
		"--paragraph", "1",
		"--align", "CENTER",
		"--indent-mm", "3",
		"--left-margin-mm", "8",
		"--space-after-mm", "4",
		"--line-spacing-percent", "180",
		"--format", "json",
	)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			ParaPrIDRef string `json:"paraPrIdRef"`
		} `json:"data"`
	}
	if err := json.Unmarshal(layoutStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode set-paragraph-layout response: %v", err)
	}
	if !envelope.Success || envelope.Data.ParaPrIDRef == "" {
		t.Fatalf("unexpected set-paragraph-layout response: %s", layoutStdout.String())
	}

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	paragraphs := sectionDoc.FindElements("//hp:p")
	if len(paragraphs) < 3 {
		t.Fatalf("expected editable paragraphs in section xml")
	}
	if got := paragraphs[2].SelectAttrValue("paraPrIDRef", ""); got != envelope.Data.ParaPrIDRef {
		t.Fatalf("unexpected paragraph paraPrIDRef: %s", got)
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	paraPr := headerDoc.FindElement("//hh:paraPr[@id='" + envelope.Data.ParaPrIDRef + "']")
	if paraPr == nil {
		t.Fatalf("styled paraPr not found: %s", envelope.Data.ParaPrIDRef)
	}
	align := paraPr.FindElement("./hh:align")
	if align == nil || align.SelectAttrValue("horizontal", "") != "CENTER" {
		t.Fatalf("unexpected align element")
	}
	lineSpacing := paraPr.FindElement(".//hh:lineSpacing")
	if lineSpacing == nil || lineSpacing.SelectAttrValue("value", "") != "180" {
		t.Fatalf("expected updated line spacing in paraPr")
	}
}

func TestSetParagraphListWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")

	listStdout := runCLI(
		t,
		"set-paragraph-list", editableDir,
		"--paragraph", "1",
		"--kind", "number",
		"--level", "1",
		"--start-number", "3",
		"--format", "json",
	)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			ParaPrIDRef string `json:"paraPrIdRef"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode set-paragraph-list response: %v", err)
	}
	if !envelope.Success || envelope.Data.ParaPrIDRef == "" {
		t.Fatalf("unexpected set-paragraph-list response: %s", listStdout.String())
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	paraPr := headerDoc.FindElement("//hh:paraPr[@id='" + envelope.Data.ParaPrIDRef + "']")
	if paraPr == nil {
		t.Fatalf("styled paraPr not found: %s", envelope.Data.ParaPrIDRef)
	}
	heading := paraPr.FindElement("./hh:heading")
	if heading == nil {
		t.Fatalf("heading not found")
	}
	if heading.SelectAttrValue("type", "") != "NUMBER" || heading.SelectAttrValue("level", "") != "1" {
		t.Fatalf("unexpected heading attrs")
	}
	numberingID := heading.SelectAttrValue("idRef", "")
	if numberingID == "" {
		t.Fatalf("numbering id missing")
	}
	numbering := headerDoc.FindElement("//hh:numbering[@id='" + numberingID + "']")
	if numbering == nil || numbering.SelectAttrValue("start", "") != "3" {
		t.Fatalf("unexpected numbering start")
	}
}

func TestSetObjectPositionWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	imagePath := filepath.Join(workDir, "tiny.png")
	if err := os.WriteFile(imagePath, mustTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "insert-image", editableDir, "--image", imagePath, "--width-mm", "20", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "40", "--height-mm", "20", "--text", "위치 테스트", "--format", "json")
	runCLI(t, "set-object-position", editableDir, "--type", "image", "--index", "0", "--treat-as-char", "false", "--x-mm", "10", "--y-mm", "5", "--horz-align", "CENTER", "--format", "json")
	runCLI(t, "set-object-position", editableDir, "--type", "textbox", "--index", "0", "--x-mm", "12", "--y-mm", "7", "--vert-align", "BOTTOM", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}

	picturePos := sectionDoc.FindElement("//hp:pic/hp:pos")
	if picturePos == nil {
		t.Fatalf("picture position not found")
	}
	if picturePos.SelectAttrValue("treatAsChar", "") != "0" || picturePos.SelectAttrValue("horzAlign", "") != "CENTER" {
		t.Fatalf("unexpected picture position attrs")
	}
	if picturePos.SelectAttrValue("horzOffset", "") == "0" || picturePos.SelectAttrValue("vertOffset", "") == "0" {
		t.Fatalf("expected updated picture offsets")
	}

	textboxPos := sectionDoc.FindElement("//hp:rect[hp:drawText]/hp:pos")
	if textboxPos == nil {
		t.Fatalf("textbox position not found")
	}
	if textboxPos.SelectAttrValue("vertAlign", "") != "BOTTOM" {
		t.Fatalf("unexpected textbox vertAlign")
	}
}

func runCLI(t *testing.T, args ...string) *bytes.Buffer {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run(args, stdout, stderr); err != nil {
		t.Fatalf("run %v: %v stderr=%s", args, err, stderr.String())
	}
	return stdout
}

func mustTinyPNG(t *testing.T) []byte {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7Z0QAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatalf("decode tiny png: %v", err)
	}
	return data
}
