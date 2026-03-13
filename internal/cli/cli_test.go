package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
