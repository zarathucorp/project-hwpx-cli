package hwpx

import (
	"os"
	"path/filepath"
	"testing"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "test", "fixtures", "minimal")
}

func TestValidateFixtureDirectory(t *testing.T) {
	report, err := Validate(fixtureDir(t))
	if err != nil {
		t.Fatalf("validate fixture: %v", err)
	}

	if !report.Valid {
		t.Fatalf("fixture should be valid: %+v", report.Errors)
	}
	if !report.RenderSafe {
		t.Fatalf("fixture should be render-safe: %+v %+v", report.RiskHints, report.RiskSignals)
	}

	if got := report.Summary.SectionPath[0]; got != "Contents/section0.xml" {
		t.Fatalf("unexpected section path: %s", got)
	}
}

func TestPackInspectAndExtractText(t *testing.T) {
	workDir := t.TempDir()
	archivePath := filepath.Join(workDir, "fixture.hwpx")

	if err := Pack(fixtureDir(t), archivePath); err != nil {
		t.Fatalf("pack fixture: %v", err)
	}

	report, err := Inspect(archivePath)
	if err != nil {
		t.Fatalf("inspect archive: %v", err)
	}

	if !report.Valid {
		t.Fatalf("archive should be valid: %+v", report.Errors)
	}
	if !report.RenderSafe {
		t.Fatalf("archive should be render-safe: %+v %+v", report.RiskHints, report.RiskSignals)
	}

	if got := report.Summary.Metadata["title"]; got != "Fixture Title" {
		t.Fatalf("unexpected title: %s", got)
	}

	text, err := ExtractText(archivePath)
	if err != nil {
		t.Fatalf("extract text: %v", err)
	}

	if text != "Hello HWPX\nSecond paragraph" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestUnpackRecreatesFiles(t *testing.T) {
	workDir := t.TempDir()
	archivePath := filepath.Join(workDir, "fixture.hwpx")
	unpackDir := filepath.Join(workDir, "unpacked")

	if err := Pack(fixtureDir(t), archivePath); err != nil {
		t.Fatalf("pack fixture: %v", err)
	}

	if err := Unpack(archivePath, unpackDir); err != nil {
		t.Fatalf("unpack archive: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(unpackDir, "Contents", "header.xml"))
	if err != nil {
		t.Fatalf("read unpacked header: %v", err)
	}

	if string(content) == "" {
		t.Fatal("header.xml should not be empty")
	}
}

func TestValidateAndPackIgnoreInternalWorkingFiles(t *testing.T) {
	workDir := t.TempDir()
	archivePath := filepath.Join(workDir, "fixture.hwpx")
	unpackDir := filepath.Join(workDir, "unpacked")
	roundtripPath := filepath.Join(workDir, "roundtrip.hwpx")

	if err := Pack(fixtureDir(t), archivePath); err != nil {
		t.Fatalf("pack fixture: %v", err)
	}
	if err := Unpack(archivePath, unpackDir); err != nil {
		t.Fatalf("unpack fixture: %v", err)
	}

	if err := os.WriteFile(filepath.Join(unpackDir, ".hwpxctl.lock"), []byte(`{"pid":123,"command":"append-text"}`), 0o644); err != nil {
		t.Fatalf("write lock file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(unpackDir, ".hwpxctl-test.tmp"), []byte("<temp/>"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	report, err := Validate(unpackDir)
	if err != nil {
		t.Fatalf("validate unpacked dir with internal files: %v", err)
	}
	if !report.Valid {
		t.Fatalf("unpacked dir should remain valid: %+v", report.Errors)
	}

	if err := Pack(unpackDir, roundtripPath); err != nil {
		t.Fatalf("pack unpacked dir with internal files: %v", err)
	}

	roundtripReport, err := Inspect(roundtripPath)
	if err != nil {
		t.Fatalf("inspect roundtrip archive: %v", err)
	}
	for _, entry := range roundtripReport.Summary.Entries {
		if entry == ".hwpxctl.lock" || entry == ".hwpxctl-test.tmp" {
			t.Fatalf("internal working file should be excluded from archive: %s", entry)
		}
	}
}
