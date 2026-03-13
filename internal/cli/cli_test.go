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
