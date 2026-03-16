package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
