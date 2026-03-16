package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/beevik/etree"
)

func CreateEditableDocument(outputDir string) (Report, error) {
	if err := ensureEmptyDir(outputDir); err != nil {
		return Report{}, err
	}

	templatePath, err := findTemplateArchive()
	if err == nil {
		if err := Unpack(templatePath, outputDir); err != nil {
			return Report{}, err
		}
		if err := resetSectionToTemplateBase(filepath.Join(outputDir, "Contents", "section0.xml")); err != nil {
			return Report{}, err
		}
		if err := os.WriteFile(filepath.Join(outputDir, "Preview", "PrvText.txt"), []byte(""), 0o644); err != nil && !os.IsNotExist(err) {
			return Report{}, err
		}
	} else {
		if err := os.MkdirAll(filepath.Join(outputDir, "META-INF"), 0o755); err != nil {
			return Report{}, err
		}
		if err := os.MkdirAll(filepath.Join(outputDir, "Contents"), 0o755); err != nil {
			return Report{}, err
		}

		files := map[string]string{
			"mimetype":               "application/hwp+zip",
			"version.xml":            defaultVersionXML,
			"settings.xml":           defaultSettingsXML,
			"META-INF/container.xml": defaultContainerXML,
			"Contents/content.hpf":   defaultContentXML,
			"Contents/header.xml":    defaultHeaderXML,
			"Contents/section0.xml":  defaultSectionXML,
		}

		for relativePath, content := range files {
			fullPath := filepath.Join(outputDir, filepath.FromSlash(relativePath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return Report{}, err
			}
			if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
				return Report{}, err
			}
		}
	}

	report, err := Validate(outputDir)
	if err != nil {
		return Report{}, err
	}
	if !report.Valid {
		return Report{}, fmt.Errorf("created invalid editable document: %s", strings.Join(report.Errors, "; "))
	}
	return report, nil
}

func ensureEmptyDir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output path is not a directory: %s", path)
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return readErr
		}
		if len(entries) > 0 {
			return fmt.Errorf("output directory must be empty: %s", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func findTemplateArchive() (string, error) {
	patterns := []string{templateGlob}
	if _, currentFile, _, ok := runtime.Caller(0); ok {
		currentDir := filepath.Dir(currentFile)
		patterns = append(patterns,
			filepath.Join(currentDir, "..", "..", templateGlob),
			filepath.Join(currentDir, "..", "..", "..", templateGlob),
		)
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return "", err
		}
		if len(matches) > 0 {
			sort.Strings(matches)
			return matches[0], nil
		}
	}
	return "", fmt.Errorf("no template archive matched %s", strings.Join(patterns, ", "))
}

func resetSectionToTemplateBase(sectionPath string) error {
	doc, err := loadXML(sectionPath)
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return fmt.Errorf("template section xml has no root")
	}

	firstParagraph := firstChildByTag(root, "hp:p")
	if firstParagraph == nil {
		return fmt.Errorf("template section xml is missing first paragraph")
	}

	firstRun := firstChildByTag(firstParagraph, "hp:run")
	if firstRun == nil {
		return fmt.Errorf("template section xml is missing first run")
	}

	sectionProperty := firstChildByTag(firstRun, "hp:secPr")
	if sectionProperty == nil {
		return fmt.Errorf("template section xml is missing hp:secPr")
	}

	columnControl := firstChildByTag(firstRun, "hp:ctrl")
	if columnControl == nil {
		return fmt.Errorf("template section xml is missing hp:ctrl")
	}

	for _, child := range append([]*etree.Element{}, root.ChildElements()...) {
		root.RemoveChild(child)
	}

	paragraph := etree.NewElement("hp:p")
	copyParagraphAttrs(firstParagraph, paragraph)
	paragraph.RemoveAttr("id")
	paragraph.CreateAttr("id", "1")

	sectionRun := etree.NewElement("hp:run")
	copyCharAttr(firstRun, sectionRun)
	sectionRun.AddChild(sectionProperty.Copy())
	sectionRun.AddChild(columnControl.Copy())
	paragraph.AddChild(sectionRun)

	emptyRun := etree.NewElement("hp:run")
	copyCharAttr(firstRun, emptyRun)
	emptyRun.CreateElement("hp:t")
	paragraph.AddChild(emptyRun)
	root.AddChild(paragraph)

	return saveXML(doc, sectionPath)
}
