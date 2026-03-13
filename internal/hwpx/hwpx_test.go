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
