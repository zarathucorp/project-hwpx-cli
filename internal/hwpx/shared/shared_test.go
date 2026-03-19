package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveXMLUsesAtomicReplaceWithoutLeavingTempFiles(t *testing.T) {
	workDir := t.TempDir()
	xmlPath := filepath.Join(workDir, "section.xml")

	sourcePath := filepath.Join("..", "..", "..", "test", "fixtures", "minimal", "Contents", "section0.xml")
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read fixture xml: %v", err)
	}
	if err := os.WriteFile(xmlPath, content, 0o644); err != nil {
		t.Fatalf("seed xml file: %v", err)
	}

	doc, err := loadXML(xmlPath)
	if err != nil {
		t.Fatalf("load xml: %v", err)
	}
	doc.Root().CreateAttr("data-test", "atomic")

	if err := saveXML(doc, xmlPath); err != nil {
		t.Fatalf("save xml: %v", err)
	}

	saved, err := os.ReadFile(xmlPath)
	if err != nil {
		t.Fatalf("read saved xml: %v", err)
	}
	if len(saved) == 0 {
		t.Fatal("saved xml should not be empty")
	}

	tempFiles, err := filepath.Glob(filepath.Join(workDir, ".hwpxctl-*.tmp"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(tempFiles) != 0 {
		t.Fatalf("expected no leftover temp files, got %v", tempFiles)
	}
}
