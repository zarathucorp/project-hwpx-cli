package hwpx

import (
	"fmt"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

const (
	defaultTableWidth   = 42520
	defaultCellHeight   = 2400
	defaultImageWidth   = 22677
	defaultEquationVer  = "Equation Version 60"
	defaultEquationFont = "HancomEQN"
	defaultSectionPath  = "Contents/section0.xml"
	templateGlob        = "example/*.hwpx"
	pageToken           = "{{PAGE}}"
	totalPageToken      = "{{TOTAL_PAGE}}"
)

type TableSpec struct {
	Rows  int
	Cols  int
	Cells [][]string
}

type tableGridEntry struct {
	cell   *etree.Element
	row    int
	col    int
	anchor [2]int
	span   [2]int
}

type ImageEmbed struct {
	ItemID     string
	BinaryPath string
}

type ImagePlacement struct {
	ItemID      string
	BinaryPath  string
	PixelWidth  int
	PixelHeight int
	Width       int
	Height      int
}

type HeaderFooterSpec struct {
	Text          []string
	ApplyPageType string
}

type PageNumberSpec struct {
	Position   string
	FormatType string
	SideChar   string
	StartPage  int
}

type NoteSpec struct {
	AnchorText string
	Text       []string
}

type MemoSpec struct {
	AnchorText string
	Text       []string
	Author     string
}

type BookmarkSpec struct {
	Name string
	Text string
}

type HyperlinkSpec struct {
	Target string
	Text   string
}

type HeadingSpec struct {
	Kind         string
	Level        int
	Text         string
	BookmarkName string
}

type TOCSpec struct {
	Title    string
	MaxLevel int
}

type CrossReferenceSpec struct {
	BookmarkName string
	Text         string
}

type EquationSpec struct {
	Script string
}

type RectangleSpec struct {
	WidthMM   float64
	HeightMM  float64
	LineColor string
	FillColor string
}

type styleRef struct {
	ID          string
	Name        string
	ParaPrIDRef string
	CharPrIDRef string
}

type headingEntry struct {
	Level        int
	Text         string
	BookmarkName string
}

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
		patterns = append(patterns, filepath.Join(filepath.Dir(currentFile), "..", "..", templateGlob))
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

func copyParagraphAttrs(src, dst *etree.Element) {
	for _, attr := range src.Attr {
		dst.CreateAttr(attr.Key, attr.Value)
	}
}

func copyCharAttr(src, dst *etree.Element) {
	if value := src.SelectAttrValue("charPrIDRef", ""); value != "" {
		dst.CreateAttr("charPrIDRef", value)
	}
}

func AddParagraphs(targetDir string, texts []string) (Report, int, error) {
	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, 0, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, 0, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, 0, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	counter := newIDCounter(root)
	added := 0
	for _, text := range texts {
		root.AddChild(newParagraphElement(counter, text))
		added++
	}

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, 0, err
	}
	return report, added, nil
}

func SetParagraphText(targetDir string, paragraphIndex int, text string) (Report, string, error) {
	if paragraphIndex < 0 {
		return Report{}, "", fmt.Errorf("paragraph index must be zero or greater")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	paragraphs := editableParagraphs(root)
	if paragraphIndex >= len(paragraphs) {
		return Report{}, "", fmt.Errorf("paragraph index out of range: %d", paragraphIndex)
	}

	paragraph := paragraphs[paragraphIndex]
	originalText := paragraphPlainText(paragraph)
	replaceParagraphText(paragraph, text)

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, originalText, nil
}

func DeleteParagraph(targetDir string, paragraphIndex int) (Report, string, error) {
	if paragraphIndex < 0 {
		return Report{}, "", fmt.Errorf("paragraph index must be zero or greater")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	paragraphs := editableParagraphs(root)
	if paragraphIndex >= len(paragraphs) {
		return Report{}, "", fmt.Errorf("paragraph index out of range: %d", paragraphIndex)
	}

	paragraph := paragraphs[paragraphIndex]
	removedText := paragraphPlainText(paragraph)
	root.RemoveChild(paragraph)

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, removedText, nil
}

func AddSection(targetDir string) (Report, int, string, error) {
	sectionPaths, err := resolveSectionPaths(targetDir)
	if err != nil {
		return Report{}, 0, "", err
	}
	if len(sectionPaths) == 0 {
		return Report{}, 0, "", fmt.Errorf("no editable section xml found")
	}

	contentDoc, err := loadXML(filepath.Join(targetDir, "Contents", "content.hpf"))
	if err != nil {
		return Report{}, 0, "", err
	}
	contentRoot := contentDoc.Root()
	if contentRoot == nil {
		return Report{}, 0, "", fmt.Errorf("content.hpf has no root")
	}

	newSectionID, newSectionPath, err := nextSectionReference(contentRoot)
	if err != nil {
		return Report{}, 0, "", err
	}

	newSectionDoc, err := newEmptySectionDocument(filepath.Join(targetDir, filepath.FromSlash(sectionPaths[len(sectionPaths)-1])))
	if err != nil {
		return Report{}, 0, "", err
	}

	if err := addSectionManifestItem(contentRoot, newSectionID, newSectionPath); err != nil {
		return Report{}, 0, "", err
	}
	if err := addSectionSpineItem(contentRoot, newSectionID); err != nil {
		return Report{}, 0, "", err
	}
	if err := saveXML(contentDoc, filepath.Join(targetDir, "Contents", "content.hpf")); err != nil {
		return Report{}, 0, "", err
	}

	newSectionFullPath := filepath.Join(targetDir, filepath.FromSlash(newSectionPath))
	if err := os.MkdirAll(filepath.Dir(newSectionFullPath), 0o755); err != nil {
		return Report{}, 0, "", err
	}
	if err := saveXML(newSectionDoc, newSectionFullPath); err != nil {
		return Report{}, 0, "", err
	}

	if err := setHeaderSectionCount(filepath.Join(targetDir, "Contents", "header.xml"), len(sectionPaths)+1); err != nil {
		return Report{}, 0, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, 0, "", err
	}
	return report, len(sectionPaths), newSectionPath, nil
}

func DeleteSection(targetDir string, sectionIndex int) (Report, string, error) {
	if sectionIndex < 0 {
		return Report{}, "", fmt.Errorf("section index must be zero or greater")
	}

	contentDoc, err := loadXML(filepath.Join(targetDir, "Contents", "content.hpf"))
	if err != nil {
		return Report{}, "", err
	}
	contentRoot := contentDoc.Root()
	if contentRoot == nil {
		return Report{}, "", fmt.Errorf("content.hpf has no root")
	}

	sections, err := sectionRefs(contentRoot)
	if err != nil {
		return Report{}, "", err
	}
	if sectionIndex >= len(sections) {
		return Report{}, "", fmt.Errorf("section index out of range: %d", sectionIndex)
	}
	if len(sections) <= 1 {
		return Report{}, "", fmt.Errorf("cannot delete the last section")
	}

	target := sections[sectionIndex]
	if err := removeSectionSpineItem(contentRoot, target.ID); err != nil {
		return Report{}, "", err
	}
	if err := removeSectionManifestItem(contentRoot, target.ID); err != nil {
		return Report{}, "", err
	}
	if err := saveXML(contentDoc, filepath.Join(targetDir, "Contents", "content.hpf")); err != nil {
		return Report{}, "", err
	}

	if err := os.Remove(filepath.Join(targetDir, filepath.FromSlash(target.Path))); err != nil && !os.IsNotExist(err) {
		return Report{}, "", err
	}

	if err := setHeaderSectionCount(filepath.Join(targetDir, "Contents", "header.xml"), len(sections)-1); err != nil {
		return Report{}, "", err
	}
	if err := normalizeSectionReferences(targetDir); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, target.Path, nil
}

func AddTable(targetDir string, spec TableSpec) (Report, int, error) {
	if spec.Rows <= 0 || spec.Cols <= 0 {
		return Report{}, 0, fmt.Errorf("table rows and cols must be positive")
	}
	if err := ensureHeaderSupport(filepath.Join(targetDir, "Contents", "header.xml"), true, false); err != nil {
		return Report{}, 0, err
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, 0, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, 0, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, 0, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	tableIndex := len(findElementsByTag(root, "hp:tbl"))
	counter := newIDCounter(root)
	root.AddChild(newTableParagraphElement(counter, spec))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, 0, err
	}
	return report, tableIndex, nil
}

func SetTableCellText(targetDir string, tableIndex, row, col int, text string) (Report, error) {
	if tableIndex < 0 || row < 0 || col < 0 {
		return Report{}, fmt.Errorf("table, row, and col must be zero or greater")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	tables := findElementsByTag(root, "hp:tbl")
	if tableIndex >= len(tables) {
		return Report{}, fmt.Errorf("table index out of range: %d", tableIndex)
	}

	entry, err := tableCellEntry(tables[tableIndex], row, col)
	if err != nil {
		return Report{}, err
	}

	subList := firstChildByTag(entry.cell, "hp:subList")
	if subList == nil {
		return Report{}, fmt.Errorf("table cell does not contain hp:subList")
	}

	for _, child := range append([]*etree.Element{}, subList.ChildElements()...) {
		if tagMatches(child.Tag, "hp:p") {
			subList.RemoveChild(child)
		}
	}

	counter := newIDCounter(root)
	subList.AddChild(newCellParagraphElement(counter, text))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func MergeTableCells(targetDir string, tableIndex, startRow, startCol, endRow, endCol int) (Report, error) {
	if tableIndex < 0 || startRow < 0 || startCol < 0 || endRow < 0 || endCol < 0 {
		return Report{}, fmt.Errorf("table and coordinates must be zero or greater")
	}
	if startRow > endRow || startCol > endCol {
		return Report{}, fmt.Errorf("merge coordinates must describe a valid rectangle")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	tables := findElementsByTag(root, "hp:tbl")
	if tableIndex >= len(tables) {
		return Report{}, fmt.Errorf("table index out of range: %d", tableIndex)
	}

	table := tables[tableIndex]
	target, err := tableCellEntry(table, startRow, startCol)
	if err != nil {
		return Report{}, err
	}
	if target.anchor != [2]int{startRow, startCol} {
		return Report{}, fmt.Errorf("top-left cell must align with merge starting position")
	}

	newRowSpan := endRow - startRow + 1
	newColSpan := endCol - startCol + 1
	totalWidth := 0
	totalHeight := 0
	widthSeen := map[*etree.Element]struct{}{}
	heightSeen := map[*etree.Element]struct{}{}
	removals := map[*etree.Element]struct{}{}

	for rowIndex := startRow; rowIndex <= endRow; rowIndex++ {
		for colIndex := startCol; colIndex <= endCol; colIndex++ {
			entry, entryErr := tableCellEntry(table, rowIndex, colIndex)
			if entryErr != nil {
				return Report{}, entryErr
			}
			anchorRow := entry.anchor[0]
			anchorCol := entry.anchor[1]
			spanRow := entry.span[0]
			spanCol := entry.span[1]
			if anchorRow < startRow || anchorCol < startCol || anchorRow+spanRow-1 > endRow || anchorCol+spanCol-1 > endCol {
				return Report{}, fmt.Errorf("cells to merge must be entirely inside the merge region")
			}
			if rowIndex == startRow {
				if _, ok := widthSeen[entry.cell]; !ok {
					widthSeen[entry.cell] = struct{}{}
					totalWidth += tableCellWidth(entry.cell)
				}
			}
			if colIndex == startCol {
				if _, ok := heightSeen[entry.cell]; !ok {
					heightSeen[entry.cell] = struct{}{}
					totalHeight += tableCellHeight(entry.cell)
				}
			}
			if entry.cell != target.cell {
				removals[entry.cell] = struct{}{}
			}
		}
	}

	for cell := range removals {
		setTableCellSpan(cell, 1, 1)
		setTableCellSize(cell, 0, 0)
		clearTableCellText(cell)
	}

	setTableCellSpan(target.cell, newRowSpan, newColSpan)
	if totalWidth <= 0 {
		totalWidth = tableCellWidth(target.cell)
	}
	if totalHeight <= 0 {
		totalHeight = tableCellHeight(target.cell)
	}
	setTableCellSize(target.cell, totalWidth, totalHeight)

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func SplitTableCell(targetDir string, tableIndex, row, col int) (Report, error) {
	if tableIndex < 0 || row < 0 || col < 0 {
		return Report{}, fmt.Errorf("table and coordinates must be zero or greater")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	tables := findElementsByTag(root, "hp:tbl")
	if tableIndex >= len(tables) {
		return Report{}, fmt.Errorf("table index out of range: %d", tableIndex)
	}

	table := tables[tableIndex]
	entry, err := tableCellEntry(table, row, col)
	if err != nil {
		return Report{}, err
	}
	if entry.span == [2]int{1, 1} {
		report, validateErr := Validate(targetDir)
		if validateErr != nil {
			return Report{}, validateErr
		}
		return report, nil
	}

	anchorCell := entry.cell
	startRow := entry.anchor[0]
	startCol := entry.anchor[1]
	spanRow := entry.span[0]
	spanCol := entry.span[1]
	widths := distributeSize(tableCellWidth(anchorCell), spanCol)
	heights := distributeSize(tableCellHeight(anchorCell), spanRow)
	if len(widths) == 0 {
		widths = []int{tableCellWidth(anchorCell)}
	}
	if len(heights) == 0 {
		heights = []int{tableCellHeight(anchorCell)}
	}

	rows := childElementsByTag(table, "hp:tr")
	if len(rows) < startRow+spanRow {
		return Report{}, fmt.Errorf("table rows missing while splitting merged cell")
	}

	counter := newIDCounter(root)
	borderFillIDRef := strings.TrimSpace(anchorCell.SelectAttrValue("borderFillIDRef", "3"))
	if borderFillIDRef == "" {
		borderFillIDRef = "3"
	}

	for rowOffset := 0; rowOffset < spanRow; rowOffset++ {
		logicalRow := startRow + rowOffset
		rowElement := rows[logicalRow]
		rowHeight := heights[minInt(rowOffset, len(heights)-1)]

		for colOffset := 0; colOffset < spanCol; colOffset++ {
			logicalCol := startCol + colOffset
			colWidth := widths[minInt(colOffset, len(widths)-1)]

			if rowOffset == 0 && colOffset == 0 {
				setTableCellSpan(anchorCell, 1, 1)
				setTableCellSize(anchorCell, colWidth, rowHeight)
				continue
			}

			cell := physicalCellAt(rowElement, logicalRow, logicalCol)
			if cell == nil {
				cell = newEmptyTableCellElement(counter, logicalRow, logicalCol, colWidth, rowHeight, borderFillIDRef)
				insertTableCell(rowElement, cell, logicalCol)
			}

			setTableCellAddress(cell, logicalRow, logicalCol)
			setTableCellSpan(cell, 1, 1)
			setTableCellSize(cell, colWidth, rowHeight)
		}
	}

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func EmbedImage(targetDir, imagePath string) (Report, ImageEmbed, error) {
	format, mediaType, err := detectImageFormat(imagePath)
	if err != nil {
		return Report{}, ImageEmbed{}, err
	}

	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return Report{}, ImageEmbed{}, err
	}

	contentPath := filepath.Join(targetDir, "Contents", "content.hpf")
	headerPath := filepath.Join(targetDir, "Contents", "header.xml")
	if err := ensureHeaderSupport(headerPath, false, true); err != nil {
		return Report{}, ImageEmbed{}, err
	}

	contentDoc, err := loadXML(contentPath)
	if err != nil {
		return Report{}, ImageEmbed{}, err
	}
	headerDoc, err := loadXML(headerPath)
	if err != nil {
		return Report{}, ImageEmbed{}, err
	}

	itemID := nextBinaryItemID(contentDoc.Root())
	binaryPath := filepath.ToSlash(filepath.Join("BinData", itemID+"."+format))
	if err := os.MkdirAll(filepath.Join(targetDir, "BinData"), 0o755); err != nil {
		return Report{}, ImageEmbed{}, err
	}
	if err := os.WriteFile(filepath.Join(targetDir, filepath.FromSlash(binaryPath)), imageBytes, 0o644); err != nil {
		return Report{}, ImageEmbed{}, err
	}

	if err := addManifestBinaryItem(contentDoc.Root(), itemID, binaryPath, mediaType); err != nil {
		return Report{}, ImageEmbed{}, err
	}
	if err := addHeaderBinaryItem(headerDoc.Root(), filepath.Base(binaryPath), format); err != nil {
		return Report{}, ImageEmbed{}, err
	}

	if err := saveXML(contentDoc, contentPath); err != nil {
		return Report{}, ImageEmbed{}, err
	}
	if err := saveXML(headerDoc, headerPath); err != nil {
		return Report{}, ImageEmbed{}, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, ImageEmbed{}, err
	}
	return report, ImageEmbed{ItemID: itemID, BinaryPath: binaryPath}, nil
}

func InsertImage(targetDir, imagePath string, widthMM float64) (Report, ImagePlacement, error) {
	report, embedded, err := EmbedImage(targetDir, imagePath)
	if err != nil {
		return Report{}, ImagePlacement{}, err
	}
	_ = report

	imageConfig, err := decodeImageConfig(imagePath)
	if err != nil {
		return Report{}, ImagePlacement{}, err
	}
	if imageConfig.Width <= 0 || imageConfig.Height <= 0 {
		return Report{}, ImagePlacement{}, fmt.Errorf("image dimensions must be positive")
	}

	width, height := calculateImageSize(imageConfig.Width, imageConfig.Height, widthMM)

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, ImagePlacement{}, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, ImagePlacement{}, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, ImagePlacement{}, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	counter := newIDCounter(root)
	root.AddChild(newPictureParagraphElement(counter, embedded.ItemID, filepath.Base(imagePath), imageConfig.Width, imageConfig.Height, width, height))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, ImagePlacement{}, err
	}

	report, err = Validate(targetDir)
	if err != nil {
		return Report{}, ImagePlacement{}, err
	}

	return report, ImagePlacement{
		ItemID:      embedded.ItemID,
		BinaryPath:  embedded.BinaryPath,
		PixelWidth:  imageConfig.Width,
		PixelHeight: imageConfig.Height,
		Width:       width,
		Height:      height,
	}, nil
}

func SetHeaderText(targetDir string, spec HeaderFooterSpec) (Report, error) {
	return setHeaderFooter(targetDir, "header", spec)
}

func SetFooterText(targetDir string, spec HeaderFooterSpec) (Report, error) {
	return setHeaderFooter(targetDir, "footer", spec)
}

func SetPageNumber(targetDir string, spec PageNumberSpec) (Report, error) {
	if spec.Position == "" {
		spec.Position = "BOTTOM_CENTER"
	}
	if spec.FormatType == "" {
		spec.FormatType = "DIGIT"
	}
	if spec.StartPage < 0 {
		return Report{}, fmt.Errorf("start page must be zero or greater")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	run, err := ensureSectionControlRun(root)
	if err != nil {
		return Report{}, err
	}

	replaceRunControl(run, "pageNum", newPageNumControlElement(spec))
	if spec.StartPage > 0 {
		if err := setSectionStartPage(run, spec.StartPage); err != nil {
			return Report{}, err
		}
	}

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func AddFootnote(targetDir string, spec NoteSpec) (Report, int, error) {
	return addNote(targetDir, "footNote", spec)
}

func AddEndnote(targetDir string, spec NoteSpec) (Report, int, error) {
	return addNote(targetDir, "endNote", spec)
}

func AddBookmark(targetDir string, spec BookmarkSpec) (Report, error) {
	if strings.TrimSpace(spec.Name) == "" {
		return Report{}, fmt.Errorf("bookmark name must not be empty")
	}
	if strings.TrimSpace(spec.Text) == "" {
		return Report{}, fmt.Errorf("bookmark text must not be empty")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, fmt.Errorf("section xml has no root: %s", sectionPath)
	}
	if bookmarkExists(root, spec.Name) {
		return Report{}, fmt.Errorf("bookmark already exists: %s", spec.Name)
	}

	counter := newIDCounter(root)
	root.AddChild(newBookmarkParagraphElement(counter, spec))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func AddHyperlink(targetDir string, spec HyperlinkSpec) (Report, string, error) {
	if strings.TrimSpace(spec.Target) == "" {
		return Report{}, "", fmt.Errorf("hyperlink target must not be empty")
	}
	if strings.TrimSpace(spec.Text) == "" {
		return Report{}, "", fmt.Errorf("hyperlink text must not be empty")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	target := strings.TrimSpace(spec.Target)
	if strings.HasPrefix(target, "#") {
		name := strings.TrimPrefix(target, "#")
		if !bookmarkExists(root, name) {
			return Report{}, "", fmt.Errorf("bookmark does not exist: %s", name)
		}
	}
	spec.Target = target

	counter := newIDCounter(root)
	fieldID := counter.Next()
	root.AddChild(newHyperlinkParagraphElement(counter, fieldID, spec))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, fieldID, nil
}

func AddHeading(targetDir string, spec HeadingSpec) (Report, string, error) {
	if strings.TrimSpace(spec.Text) == "" {
		return Report{}, "", fmt.Errorf("heading text must not be empty")
	}

	styleByName, _, err := loadStyleRefs(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	style, err := resolveHeadingStyle(styleByName, spec)
	if err != nil {
		return Report{}, "", err
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	counter := newIDCounter(root)
	bookmarkName, err := resolveBookmarkName(root, counter, spec.BookmarkName, "heading")
	if err != nil {
		return Report{}, "", err
	}

	root.AddChild(newStyledParagraphElement(counter, style, spec.Text, bookmarkName))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, bookmarkName, nil
}

func InsertTOC(targetDir string, spec TOCSpec) (Report, int, error) {
	styleByName, styleByID, err := loadStyleRefs(targetDir)
	if err != nil {
		return Report{}, 0, err
	}

	maxLevel := spec.MaxLevel
	if maxLevel <= 0 {
		maxLevel = 3
	}
	title := strings.TrimSpace(spec.Title)
	if title == "" {
		title = "목차"
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, 0, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, 0, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, 0, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	counter := newIDCounter(root)
	entries, err := collectHeadingEntries(root, styleByID, counter, maxLevel)
	if err != nil {
		return Report{}, 0, err
	}
	if len(entries) == 0 {
		return Report{}, 0, fmt.Errorf("no heading paragraphs found for table of contents")
	}

	tocHeadingStyle, err := resolveNamedStyle(styleByName, []string{"TOC Heading"}...)
	if err != nil {
		return Report{}, 0, err
	}

	insertIndex := 1
	root.InsertChildAt(insertIndex, newStyledParagraphElement(counter, tocHeadingStyle, title, ""))
	insertIndex++

	for _, entry := range entries {
		style, resolveErr := resolveTOCStyle(styleByName, entry.Level)
		if resolveErr != nil {
			return Report{}, 0, resolveErr
		}
		root.InsertChildAt(insertIndex, newHyperlinkStyledParagraphElement(counter, style, "#"+entry.BookmarkName, entry.Text))
		insertIndex++
	}

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, 0, err
	}
	return report, len(entries), nil
}

func AddCrossReference(targetDir string, spec CrossReferenceSpec) (Report, string, string, error) {
	bookmarkName := strings.TrimSpace(spec.BookmarkName)
	if bookmarkName == "" {
		return Report{}, "", "", fmt.Errorf("cross reference bookmark must not be empty")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", "", err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", "", err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, "", "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}
	if !bookmarkExists(root, bookmarkName) {
		return Report{}, "", "", fmt.Errorf("bookmark does not exist: %s", bookmarkName)
	}

	text := strings.TrimSpace(spec.Text)
	if text == "" {
		paragraph := findParagraphByBookmark(root, bookmarkName)
		text = strings.TrimSpace(paragraphPlainText(paragraph))
	}
	if text == "" {
		return Report{}, "", "", fmt.Errorf("cross reference text must not be empty")
	}

	counter := newIDCounter(root)
	fieldID := counter.Next()
	root.AddChild(newHyperlinkParagraphElement(counter, fieldID, HyperlinkSpec{
		Target: "#" + bookmarkName,
		Text:   text,
	}))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", "", err
	}
	return report, fieldID, text, nil
}

func AddEquation(targetDir string, spec EquationSpec) (Report, string, error) {
	script := strings.TrimSpace(spec.Script)
	if script == "" {
		return Report{}, "", fmt.Errorf("equation script must not be empty")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	counter := newIDCounter(root)
	equationID := counter.Next()
	root.AddChild(newEquationParagraphElement(counter, equationID, script))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, equationID, nil
}

func AddMemo(targetDir string, spec MemoSpec) (Report, string, string, int, error) {
	if strings.TrimSpace(spec.AnchorText) == "" {
		return Report{}, "", "", 0, fmt.Errorf("memo anchor text must not be empty")
	}
	if len(spec.Text) == 0 {
		return Report{}, "", "", 0, fmt.Errorf("memo text must not be empty")
	}

	headerPath := filepath.Join(targetDir, "Contents", "header.xml")
	if err := ensureMemoSupport(headerPath); err != nil {
		return Report{}, "", "", 0, err
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", "", 0, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", "", 0, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, "", "", 0, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	counter := newIDCounter(root)
	memoNumber := nextMemoNumber(root)
	memoID := counter.Next()
	fieldID := counter.Next()

	memoGroup := ensureMemoGroup(root)
	memoGroup.AddChild(newMemoElement(counter, memoID, spec))
	root.AddChild(newMemoAnchorParagraphElement(counter, memoID, fieldID, memoNumber, spec))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", "", 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", "", 0, err
	}
	return report, memoID, fieldID, memoNumber, nil
}

func AddRectangle(targetDir string, spec RectangleSpec) (Report, string, int, int, error) {
	width := mmToHWPUnit(spec.WidthMM)
	height := mmToHWPUnit(spec.HeightMM)
	if width <= 0 || height <= 0 {
		return Report{}, "", 0, 0, fmt.Errorf("rectangle width and height must be positive")
	}

	lineColor := strings.TrimSpace(spec.LineColor)
	if lineColor == "" {
		lineColor = "#000000"
	}

	fillColor := strings.TrimSpace(spec.FillColor)
	if fillColor == "" {
		fillColor = "#FFFFFF"
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", 0, 0, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", 0, 0, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, "", 0, 0, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	counter := newIDCounter(root)
	shapeID := counter.Next()
	root.AddChild(newRectangleParagraphElement(counter, shapeID, width, height, lineColor, fillColor))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", 0, 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", 0, 0, err
	}
	return report, shapeID, width, height, nil
}

func resolvePrimarySectionPath(targetDir string) (string, error) {
	sectionPaths, err := resolveSectionPaths(targetDir)
	if err != nil {
		return "", err
	}
	if len(sectionPaths) > 0 {
		return sectionPaths[0], nil
	}

	fallback := filepath.Join(targetDir, filepath.FromSlash(defaultSectionPath))
	if _, err := os.Stat(fallback); err == nil {
		return defaultSectionPath, nil
	}
	return "", fmt.Errorf("no editable section xml found")
}

func resolveSectionPaths(targetDir string) ([]string, error) {
	report, err := Validate(targetDir)
	if err != nil {
		return nil, err
	}
	if len(report.Summary.SectionPath) > 0 {
		return report.Summary.SectionPath, nil
	}

	fallback := filepath.Join(targetDir, filepath.FromSlash(defaultSectionPath))
	if _, err := os.Stat(fallback); err == nil {
		return []string{defaultSectionPath}, nil
	}
	return nil, fmt.Errorf("no editable section xml found")
}

func loadStyleRefs(targetDir string) (map[string]styleRef, map[string]styleRef, error) {
	doc, err := loadXML(filepath.Join(targetDir, "Contents", "header.xml"))
	if err != nil {
		return nil, nil, err
	}

	root := doc.Root()
	if root == nil {
		return nil, nil, fmt.Errorf("header.xml has no root")
	}

	byName := map[string]styleRef{}
	byID := map[string]styleRef{}
	for _, element := range findElementsByTag(root, "hh:style") {
		style := styleRef{
			ID:          strings.TrimSpace(element.SelectAttrValue("id", "")),
			Name:        strings.TrimSpace(element.SelectAttrValue("name", "")),
			ParaPrIDRef: strings.TrimSpace(element.SelectAttrValue("paraPrIDRef", "")),
			CharPrIDRef: strings.TrimSpace(element.SelectAttrValue("charPrIDRef", "")),
		}
		if style.ID == "" || style.Name == "" {
			continue
		}
		byID[style.ID] = style
		byName[normalizeStyleName(style.Name)] = style
	}
	return byName, byID, nil
}

func resolveHeadingStyle(styleByName map[string]styleRef, spec HeadingSpec) (styleRef, error) {
	kind := strings.ToLower(strings.TrimSpace(spec.Kind))
	if kind == "" {
		kind = "heading"
	}

	switch kind {
	case "title":
		return resolveNamedStyle(styleByName, "Title")
	case "heading":
		if spec.Level < 1 || spec.Level > 9 {
			return styleRef{}, fmt.Errorf("heading level must be between 1 and 9")
		}
		return resolveNamedStyle(styleByName, fmt.Sprintf("heading %d", spec.Level))
	case "outline":
		if spec.Level < 1 || spec.Level > 7 {
			return styleRef{}, fmt.Errorf("outline level must be between 1 and 7")
		}
		return resolveNamedStyle(styleByName, fmt.Sprintf("개요 %d", spec.Level))
	default:
		return styleRef{}, fmt.Errorf("unsupported heading kind: %s", spec.Kind)
	}
}

func resolveTOCStyle(styleByName map[string]styleRef, level int) (styleRef, error) {
	if level < 1 {
		level = 1
	}
	if level > 9 {
		level = 9
	}

	style, err := resolveNamedStyle(styleByName, fmt.Sprintf("toc %d", level))
	if err == nil {
		return style, nil
	}
	return resolveNamedStyle(styleByName, "본문", "바탕글")
}

func resolveNamedStyle(styleByName map[string]styleRef, names ...string) (styleRef, error) {
	for _, name := range names {
		if style, ok := styleByName[normalizeStyleName(name)]; ok {
			return style, nil
		}
	}
	return styleRef{}, fmt.Errorf("style not found: %s", strings.Join(names, ", "))
}

func normalizeStyleName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func resolveBookmarkName(root *etree.Element, counter *idCounter, requested, prefix string) (string, error) {
	name := strings.TrimSpace(requested)
	if name != "" {
		if bookmarkExists(root, name) {
			return "", fmt.Errorf("bookmark already exists: %s", name)
		}
		return name, nil
	}

	base := strings.TrimSpace(prefix)
	if base == "" {
		base = "bookmark"
	}

	for {
		candidate := fmt.Sprintf("%s-%s", base, counter.Next())
		if !bookmarkExists(root, candidate) {
			return candidate, nil
		}
	}
}

func collectHeadingEntries(root *etree.Element, styleByID map[string]styleRef, counter *idCounter, maxLevel int) ([]headingEntry, error) {
	var entries []headingEntry
	for _, paragraph := range childElementsByTag(root, "hp:p") {
		if hasSectionProperty(paragraph) {
			continue
		}

		style, ok := styleByID[strings.TrimSpace(paragraph.SelectAttrValue("styleIDRef", ""))]
		if !ok {
			continue
		}

		level, include := headingLevelForStyle(style.Name)
		if !include || level > maxLevel {
			continue
		}

		text := strings.TrimSpace(paragraphPlainText(paragraph))
		if text == "" {
			continue
		}

		bookmarkName := firstBookmarkName(paragraph)
		if bookmarkName == "" {
			generated, err := resolveBookmarkName(root, counter, "", "toc")
			if err != nil {
				return nil, err
			}
			bookmarkName = generated
			insertBookmarkRun(paragraph, bookmarkName)
		}

		entries = append(entries, headingEntry{
			Level:        level,
			Text:         text,
			BookmarkName: bookmarkName,
		})
	}
	return entries, nil
}

func headingLevelForStyle(styleName string) (int, bool) {
	name := normalizeStyleName(styleName)
	if strings.HasPrefix(name, "heading ") {
		level, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(name, "heading ")))
		if err == nil && level > 0 {
			return level, true
		}
	}
	if strings.HasPrefix(styleName, "개요 ") {
		level, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(styleName, "개요 ")))
		if err == nil && level > 0 {
			return level, true
		}
	}
	return 0, false
}

func hasSectionProperty(paragraph *etree.Element) bool {
	for _, run := range childElementsByTag(paragraph, "hp:run") {
		if firstChildByTag(run, "hp:secPr") != nil {
			return true
		}
	}
	return false
}

func firstBookmarkName(paragraph *etree.Element) string {
	for _, element := range findElementsByTag(paragraph, "hp:bookmark") {
		name := strings.TrimSpace(element.SelectAttrValue("name", ""))
		if name != "" {
			return name
		}
	}
	return ""
}

func insertBookmarkRun(paragraph *etree.Element, name string) {
	run := etree.NewElement("hp:run")
	run.CreateAttr("charPrIDRef", firstRunCharPrIDRef(paragraph))
	ctrl := run.CreateElement("hp:ctrl")
	bookmark := ctrl.CreateElement("hp:bookmark")
	bookmark.CreateAttr("name", name)
	paragraph.InsertChildAt(0, run)
}

func firstRunCharPrIDRef(paragraph *etree.Element) string {
	for _, child := range paragraph.ChildElements() {
		if !tagMatches(child.Tag, "hp:run") {
			continue
		}
		value := strings.TrimSpace(child.SelectAttrValue("charPrIDRef", ""))
		if value != "" {
			return value
		}
	}
	return "0"
}

func editableParagraphs(root *etree.Element) []*etree.Element {
	var paragraphs []*etree.Element
	for _, paragraph := range childElementsByTag(root, "hp:p") {
		if hasSectionProperty(paragraph) {
			continue
		}
		paragraphs = append(paragraphs, paragraph)
	}
	return paragraphs
}

func replaceParagraphText(paragraph *etree.Element, text string) {
	charPrIDRef := firstRunCharPrIDRef(paragraph)
	for _, child := range append([]*etree.Element{}, paragraph.ChildElements()...) {
		paragraph.RemoveChild(child)
	}

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", charPrIDRef)
	textElement := run.CreateElement("hp:t")
	textElement.SetText(text)
	paragraph.AddChild(newHeaderFooterLineSegElement(text))
}

func paragraphPlainText(paragraph *etree.Element) string {
	if paragraph == nil {
		return ""
	}

	var builder strings.Builder
	var walk func(*etree.Element)
	walk = func(element *etree.Element) {
		if element == nil {
			return
		}

		switch localTag(element.Tag) {
		case "t":
			builder.WriteString(element.Text())
		case "lineBreak":
			builder.WriteByte('\n')
		case "tab":
			builder.WriteByte('\t')
		}

		for _, child := range element.ChildElements() {
			walk(child)
		}
	}
	walk(paragraph)
	return builder.String()
}

func findParagraphByBookmark(root *etree.Element, name string) *etree.Element {
	for _, paragraph := range childElementsByTag(root, "hp:p") {
		if firstBookmarkName(paragraph) == name {
			return paragraph
		}
	}
	return nil
}

func ensureHeaderSupport(headerPath string, includeBorderFill bool, includeBinData bool) error {
	doc, err := loadXML(headerPath)
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return fmt.Errorf("header xml has no root")
	}

	refList := firstChildByTag(root, "hh:refList")
	if refList == nil {
		refList = etree.NewElement("hh:refList")
		root.AddChild(refList)
	}

	if includeBorderFill {
		borderFills := firstChildByTag(refList, "hh:borderFills")
		if borderFills == nil {
			borderFills = etree.NewElement("hh:borderFills")
			refList.AddChild(borderFills)
		}
		ensureBorderFill(borderFills, "1", false)
		ensureBorderFill(borderFills, "2", true)
		ensureBorderFill(borderFills, "3", false)
		borderFills.CreateAttr("itemCnt", strconv.Itoa(len(childElementsByTag(borderFills, "hh:borderFill"))))
	}

	if includeBinData {
		binDataList := firstChildByTag(refList, "hh:binDataList")
		if binDataList == nil {
			binDataList = etree.NewElement("hh:binDataList")
			binDataList.CreateAttr("itemCnt", "0")
			refList.AddChild(binDataList)
		}
		binDataList.CreateAttr("itemCnt", strconv.Itoa(len(childElementsByTag(binDataList, "hh:binItem"))))
	}

	return saveXML(doc, headerPath)
}

func ensureMemoSupport(headerPath string) error {
	doc, err := loadXML(headerPath)
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return fmt.Errorf("header xml has no root")
	}

	refList := firstChildByTag(root, "hh:refList")
	if refList == nil {
		refList = etree.NewElement("hh:refList")
		root.AddChild(refList)
	}

	memoProperties := firstChildByTag(refList, "hh:memoProperties")
	if memoProperties == nil {
		memoProperties = etree.NewElement("hh:memoProperties")
		refList.AddChild(memoProperties)
	}

	ensureMemoShape(memoProperties, "0")
	memoProperties.CreateAttr("itemCnt", strconv.Itoa(len(childElementsByTag(memoProperties, "hh:memoPr"))))

	return saveXML(doc, headerPath)
}

func setHeaderFooter(targetDir, tag string, spec HeaderFooterSpec) (Report, error) {
	if len(spec.Text) == 0 {
		return Report{}, fmt.Errorf("%s text must not be empty", tag)
	}
	if spec.ApplyPageType == "" {
		spec.ApplyPageType = "BOTH"
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	run, err := ensureSectionControlRun(root)
	if err != nil {
		return Report{}, err
	}

	counter := newIDCounter(root)
	replaceRunControl(run, tag, newHeaderFooterControlElement(tag, spec, counter))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func addNote(targetDir, tag string, spec NoteSpec) (Report, int, error) {
	if strings.TrimSpace(spec.AnchorText) == "" {
		return Report{}, 0, fmt.Errorf("%s anchor text must not be empty", tag)
	}
	if len(spec.Text) == 0 {
		return Report{}, 0, fmt.Errorf("%s text must not be empty", tag)
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, 0, err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, 0, err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, 0, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	counter := newIDCounter(root)
	noteNumber := nextNoteNumber(root, tag)
	root.AddChild(newNoteParagraphElement(counter, tag, spec, noteNumber))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, 0, err
	}

	return report, noteNumber, nil
}

func ensureBorderFill(borderFills *etree.Element, id string, transparentFill bool) {
	for _, child := range childElementsByTag(borderFills, "hh:borderFill") {
		if child.SelectAttrValue("id", "") == id {
			return
		}
	}

	borderFill := etree.NewElement("hh:borderFill")
	borderFill.CreateAttr("id", id)
	borderFill.CreateAttr("threeD", "0")
	borderFill.CreateAttr("shadow", "0")
	borderFill.CreateAttr("centerLine", "NONE")
	borderFill.CreateAttr("breakCellSeparateLine", "0")
	borderFill.AddChild(newBorderLineElement("hh:slash", "NONE", "0.1 mm", "#000000"))
	borderFill.AddChild(newBorderLineElement("hh:backSlash", "NONE", "0.1 mm", "#000000"))

	borderType := "NONE"
	borderWidth := "0.1 mm"
	if id == "3" {
		borderType = "SOLID"
		borderWidth = "0.12 mm"
	}

	borderFill.AddChild(newBorderLineElement("hh:leftBorder", borderType, borderWidth, "#000000"))
	borderFill.AddChild(newBorderLineElement("hh:rightBorder", borderType, borderWidth, "#000000"))
	borderFill.AddChild(newBorderLineElement("hh:topBorder", borderType, borderWidth, "#000000"))
	borderFill.AddChild(newBorderLineElement("hh:bottomBorder", borderType, borderWidth, "#000000"))
	borderFill.AddChild(newBorderLineElement("hh:diagonal", "SOLID", "0.1 mm", "#000000"))

	if transparentFill {
		fillBrush := etree.NewElement("hc:fillBrush")
		winBrush := etree.NewElement("hc:winBrush")
		winBrush.CreateAttr("faceColor", "none")
		winBrush.CreateAttr("hatchColor", "#999999")
		winBrush.CreateAttr("alpha", "0")
		fillBrush.AddChild(winBrush)
		borderFill.AddChild(fillBrush)
	}

	borderFills.AddChild(borderFill)
}

func ensureMemoShape(memoProperties *etree.Element, id string) {
	for _, child := range childElementsByTag(memoProperties, "hh:memoPr") {
		if child.SelectAttrValue("id", "") == id {
			return
		}
	}

	memoShape := etree.NewElement("hh:memoPr")
	memoShape.CreateAttr("id", id)
	memoShape.CreateAttr("width", "55")
	memoShape.CreateAttr("lineWidth", "0.12 mm")
	memoShape.CreateAttr("lineType", "SOLID")
	memoShape.CreateAttr("lineColor", "#000000")
	memoShape.CreateAttr("fillColor", "#CCFF99")
	memoShape.CreateAttr("activeColor", "#FFFF99")
	memoShape.CreateAttr("memoType", "NORMAL")
	memoProperties.AddChild(memoShape)
}

func newBorderLineElement(tag, borderType, width, color string) *etree.Element {
	element := etree.NewElement(tag)
	element.CreateAttr("type", borderType)
	if tag == "hh:slash" || tag == "hh:backSlash" {
		element.CreateAttr("Crooked", "0")
		element.CreateAttr("isCounter", "0")
		return element
	}
	element.CreateAttr("width", width)
	element.CreateAttr("color", color)
	return element
}

func addManifestBinaryItem(root *etree.Element, itemID, href, mediaType string) error {
	manifest := firstChildByTag(root, "opf:manifest")
	if manifest == nil {
		return fmt.Errorf("content.hpf is missing opf:manifest")
	}

	for _, item := range childElementsByTag(manifest, "opf:item") {
		if item.SelectAttrValue("id", "") == itemID || item.SelectAttrValue("href", "") == href {
			return nil
		}
	}

	item := etree.NewElement("opf:item")
	item.CreateAttr("id", itemID)
	item.CreateAttr("href", href)
	item.CreateAttr("media-type", mediaType)
	item.CreateAttr("isEmbeded", "1")
	manifest.AddChild(item)
	return nil
}

func addSectionManifestItem(root *etree.Element, itemID, href string) error {
	manifest := firstChildByTag(root, "opf:manifest")
	if manifest == nil {
		return fmt.Errorf("content.hpf is missing opf:manifest")
	}

	for _, item := range childElementsByTag(manifest, "opf:item") {
		if item.SelectAttrValue("id", "") == itemID || item.SelectAttrValue("href", "") == href {
			return fmt.Errorf("section manifest item already exists: %s", itemID)
		}
	}

	item := etree.NewElement("opf:item")
	item.CreateAttr("id", itemID)
	item.CreateAttr("href", href)
	item.CreateAttr("media-type", "application/xml")
	manifest.AddChild(item)
	return nil
}

func addSectionSpineItem(root *etree.Element, itemID string) error {
	spine := firstChildByTag(root, "opf:spine")
	if spine == nil {
		return fmt.Errorf("content.hpf is missing opf:spine")
	}

	itemRef := etree.NewElement("opf:itemref")
	itemRef.CreateAttr("idref", itemID)
	itemRef.CreateAttr("linear", "yes")
	spine.AddChild(itemRef)
	return nil
}

func removeSectionManifestItem(root *etree.Element, itemID string) error {
	manifest := firstChildByTag(root, "opf:manifest")
	if manifest == nil {
		return fmt.Errorf("content.hpf is missing opf:manifest")
	}

	for _, item := range childElementsByTag(manifest, "opf:item") {
		if item.SelectAttrValue("id", "") != itemID {
			continue
		}
		manifest.RemoveChild(item)
		return nil
	}
	return fmt.Errorf("section manifest item not found: %s", itemID)
}

func removeSectionSpineItem(root *etree.Element, itemID string) error {
	spine := firstChildByTag(root, "opf:spine")
	if spine == nil {
		return fmt.Errorf("content.hpf is missing opf:spine")
	}

	for _, itemRef := range childElementsByTag(spine, "opf:itemref") {
		if itemRef.SelectAttrValue("idref", "") != itemID {
			continue
		}
		spine.RemoveChild(itemRef)
		return nil
	}
	return fmt.Errorf("section spine item not found: %s", itemID)
}

func addHeaderBinaryItem(root *etree.Element, binaryName, format string) error {
	refList := firstChildByTag(root, "hh:refList")
	if refList == nil {
		return fmt.Errorf("header.xml is missing hh:refList")
	}

	binDataList := firstChildByTag(refList, "hh:binDataList")
	if binDataList == nil {
		binDataList = etree.NewElement("hh:binDataList")
		refList.AddChild(binDataList)
	}

	for _, item := range childElementsByTag(binDataList, "hh:binItem") {
		if item.SelectAttrValue("BinData", "") == binaryName {
			binDataList.CreateAttr("itemCnt", strconv.Itoa(len(childElementsByTag(binDataList, "hh:binItem"))))
			return nil
		}
	}

	maxID := -1
	for _, item := range childElementsByTag(binDataList, "hh:binItem") {
		value, err := strconv.Atoi(item.SelectAttrValue("id", "-1"))
		if err == nil && value > maxID {
			maxID = value
		}
	}

	item := etree.NewElement("hh:binItem")
	item.CreateAttr("id", strconv.Itoa(maxID+1))
	item.CreateAttr("Type", "Embedding")
	item.CreateAttr("BinData", binaryName)
	item.CreateAttr("Format", format)
	binDataList.AddChild(item)
	binDataList.CreateAttr("itemCnt", strconv.Itoa(len(childElementsByTag(binDataList, "hh:binItem"))))
	return nil
}

func nextBinaryItemID(root *etree.Element) string {
	maxValue := 0
	for _, item := range findElementsByTag(root, "opf:item") {
		id := item.SelectAttrValue("id", "")
		if !strings.HasPrefix(id, "image") {
			continue
		}
		value, err := strconv.Atoi(strings.TrimPrefix(id, "image"))
		if err == nil && value > maxValue {
			maxValue = value
		}
	}
	return fmt.Sprintf("image%d", maxValue+1)
}

type sectionRef struct {
	ID   string
	Path string
}

func nextSectionReference(root *etree.Element) (string, string, error) {
	sections, err := sectionRefs(root)
	if err != nil {
		return "", "", err
	}

	maxValue := -1
	for _, section := range sections {
		for _, candidate := range []string{section.ID, filepath.Base(section.Path)} {
			value, ok := parseSectionNumber(candidate)
			if ok && value > maxValue {
				maxValue = value
			}
		}
	}

	nextValue := maxValue + 1
	return fmt.Sprintf("section%d", nextValue), fmt.Sprintf("Contents/section%d.xml", nextValue), nil
}

func sectionRefs(root *etree.Element) ([]sectionRef, error) {
	manifest := firstChildByTag(root, "opf:manifest")
	if manifest == nil {
		return nil, fmt.Errorf("content.hpf is missing opf:manifest")
	}
	spine := firstChildByTag(root, "opf:spine")
	if spine == nil {
		return nil, fmt.Errorf("content.hpf is missing opf:spine")
	}

	manifestByID := map[string]*etree.Element{}
	for _, item := range childElementsByTag(manifest, "opf:item") {
		manifestByID[item.SelectAttrValue("id", "")] = item
	}

	var sections []sectionRef
	for _, itemRef := range childElementsByTag(spine, "opf:itemref") {
		idref := strings.TrimSpace(itemRef.SelectAttrValue("idref", ""))
		item := manifestByID[idref]
		if item == nil {
			continue
		}
		href := strings.TrimSpace(item.SelectAttrValue("href", ""))
		if !isSectionPath(href) && !isSectionPath(resolveEntryPath(href, nil)) {
			continue
		}
		sections = append(sections, sectionRef{ID: idref, Path: href})
	}
	return sections, nil
}

func normalizeSectionReferences(targetDir string) error {
	doc, err := loadXML(filepath.Join(targetDir, "Contents", "content.hpf"))
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return fmt.Errorf("content.hpf has no root")
	}

	manifest := firstChildByTag(root, "opf:manifest")
	if manifest == nil {
		return fmt.Errorf("content.hpf is missing opf:manifest")
	}
	spine := firstChildByTag(root, "opf:spine")
	if spine == nil {
		return fmt.Errorf("content.hpf is missing opf:spine")
	}

	manifestByID := map[string]*etree.Element{}
	for _, item := range childElementsByTag(manifest, "opf:item") {
		manifestByID[item.SelectAttrValue("id", "")] = item
	}

	type sectionBinding struct {
		ref      sectionRef
		itemRef  *etree.Element
		manifest *etree.Element
		tempPath string
	}

	var bindings []sectionBinding
	for _, itemRef := range childElementsByTag(spine, "opf:itemref") {
		idref := strings.TrimSpace(itemRef.SelectAttrValue("idref", ""))
		item := manifestByID[idref]
		if item == nil {
			continue
		}
		href := strings.TrimSpace(item.SelectAttrValue("href", ""))
		if !isSectionPath(href) && !isSectionPath(resolveEntryPath(href, nil)) {
			continue
		}
		bindings = append(bindings, sectionBinding{
			ref:      sectionRef{ID: idref, Path: href},
			itemRef:  itemRef,
			manifest: item,
		})
	}

	for index := range bindings {
		desiredPath := fmt.Sprintf("Contents/section%d.xml", index)
		if bindings[index].ref.Path == desiredPath {
			continue
		}

		currentFullPath := filepath.Join(targetDir, filepath.FromSlash(bindings[index].ref.Path))
		tempPath := filepath.Join(targetDir, "Contents", fmt.Sprintf(".section-tmp-%d.xml", index))
		if err := os.Rename(currentFullPath, tempPath); err != nil {
			return err
		}
		bindings[index].tempPath = tempPath
	}

	for index := range bindings {
		desiredID := fmt.Sprintf("section%d", index)
		desiredPath := fmt.Sprintf("Contents/section%d.xml", index)

		bindings[index].manifest.RemoveAttr("id")
		bindings[index].manifest.CreateAttr("id", desiredID)
		bindings[index].manifest.RemoveAttr("href")
		bindings[index].manifest.CreateAttr("href", desiredPath)

		bindings[index].itemRef.RemoveAttr("idref")
		bindings[index].itemRef.CreateAttr("idref", desiredID)
	}

	for index := range bindings {
		if bindings[index].tempPath == "" {
			continue
		}
		desiredFullPath := filepath.Join(targetDir, "Contents", fmt.Sprintf("section%d.xml", index))
		if err := os.Rename(bindings[index].tempPath, desiredFullPath); err != nil {
			return err
		}
	}

	return saveXML(doc, filepath.Join(targetDir, "Contents", "content.hpf"))
}

func parseSectionNumber(value string) (int, bool) {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "Contents/")
	trimmed = strings.TrimSuffix(trimmed, ".xml")
	if !strings.HasPrefix(trimmed, "section") {
		return 0, false
	}
	number, err := strconv.Atoi(strings.TrimPrefix(trimmed, "section"))
	if err != nil {
		return 0, false
	}
	return number, true
}

func ensureSectionControlRun(root *etree.Element) (*etree.Element, error) {
	firstParagraph := firstChildByTag(root, "hp:p")
	if firstParagraph == nil {
		return nil, fmt.Errorf("section xml is missing first paragraph")
	}
	firstRun := firstChildByTag(firstParagraph, "hp:run")
	if firstRun == nil {
		return nil, fmt.Errorf("section xml is missing first run")
	}
	if firstChildByTag(firstRun, "hp:secPr") == nil {
		return nil, fmt.Errorf("section xml first run is missing hp:secPr")
	}
	return firstRun, nil
}

func newEmptySectionDocument(sourcePath string) (*etree.Document, error) {
	doc, err := loadXML(sourcePath)
	if err != nil {
		return nil, err
	}

	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("section xml has no root: %s", sourcePath)
	}

	firstParagraph := firstChildByTag(root, "hp:p")
	firstRun := (*etree.Element)(nil)
	if firstParagraph != nil {
		firstRun = firstChildByTag(firstParagraph, "hp:run")
	}

	sectionProperty := (*etree.Element)(nil)
	if firstRun != nil {
		sectionProperty = firstChildByTag(firstRun, "hp:secPr")
	}
	if sectionProperty == nil {
		fallbackDoc := etree.NewDocument()
		if err := fallbackDoc.ReadFromString(defaultSectionXML); err != nil {
			return nil, err
		}
		return fallbackDoc, nil
	}

	newDoc := etree.NewDocument()
	newRoot := root.Copy()
	for _, child := range append([]*etree.Element{}, newRoot.ChildElements()...) {
		newRoot.RemoveChild(child)
	}
	newDoc.SetRoot(newRoot)

	paragraph := etree.NewElement("hp:p")
	if firstParagraph != nil {
		copyParagraphAttrs(firstParagraph, paragraph)
	}
	paragraph.RemoveAttr("id")
	paragraph.CreateAttr("id", "1")
	if paragraph.SelectAttr("paraPrIDRef") == nil {
		paragraph.CreateAttr("paraPrIDRef", "0")
	}
	if paragraph.SelectAttr("styleIDRef") == nil {
		paragraph.CreateAttr("styleIDRef", "0")
	}
	if paragraph.SelectAttr("pageBreak") == nil {
		paragraph.CreateAttr("pageBreak", "0")
	}
	if paragraph.SelectAttr("columnBreak") == nil {
		paragraph.CreateAttr("columnBreak", "0")
	}
	if paragraph.SelectAttr("merged") == nil {
		paragraph.CreateAttr("merged", "0")
	}

	sectionRun := etree.NewElement("hp:run")
	if firstRun != nil {
		copyCharAttr(firstRun, sectionRun)
	}
	if sectionRun.SelectAttr("charPrIDRef") == nil {
		sectionRun.CreateAttr("charPrIDRef", "0")
	}
	sectionRun.AddChild(sectionProperty.Copy())
	if firstRun != nil {
		for _, child := range firstRun.ChildElements() {
			if tagMatches(child.Tag, "hp:ctrl") {
				sectionRun.AddChild(child.Copy())
			}
		}
	}
	paragraph.AddChild(sectionRun)

	emptyRun := etree.NewElement("hp:run")
	if firstRun != nil {
		copyCharAttr(firstRun, emptyRun)
	}
	if emptyRun.SelectAttr("charPrIDRef") == nil {
		emptyRun.CreateAttr("charPrIDRef", "0")
	}
	emptyRun.CreateElement("hp:t")
	paragraph.AddChild(emptyRun)
	newRoot.AddChild(paragraph)

	return newDoc, nil
}

func replaceRunControl(run *etree.Element, targetTag string, ctrl *etree.Element) {
	for _, child := range append([]*etree.Element{}, run.ChildElements()...) {
		if !tagMatches(child.Tag, "hp:ctrl") {
			continue
		}
		for _, nested := range child.ChildElements() {
			if tagMatches(nested.Tag, "hp:"+targetTag) {
				run.RemoveChild(child)
				break
			}
		}
	}
	run.AddChild(ctrl)
}

func setSectionStartPage(run *etree.Element, startPage int) error {
	sectionProperty := firstChildByTag(run, "hp:secPr")
	if sectionProperty == nil {
		return fmt.Errorf("section run is missing hp:secPr")
	}
	startNum := firstChildByTag(sectionProperty, "hp:startNum")
	if startNum == nil {
		return fmt.Errorf("section property is missing hp:startNum")
	}
	startNum.RemoveAttr("page")
	startNum.CreateAttr("page", strconv.Itoa(startPage))
	return nil
}

func setHeaderSectionCount(headerPath string, sectionCount int) error {
	doc, err := loadXML(headerPath)
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return fmt.Errorf("header xml has no root")
	}

	root.RemoveAttr("secCnt")
	root.CreateAttr("secCnt", strconv.Itoa(maxInt(sectionCount, 1)))
	return saveXML(doc, headerPath)
}

func detectImageFormat(imagePath string) (string, string, error) {
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(imagePath)), ".")
	switch format {
	case "png":
		return format, "image/png", nil
	case "jpg", "jpeg":
		return format, "image/jpeg", nil
	case "gif":
		return format, "image/gif", nil
	case "bmp":
		return format, "image/bmp", nil
	case "svg":
		return format, "image/svg+xml", nil
	default:
		return "", "", fmt.Errorf("unsupported image format: %s", format)
	}
}

func decodeImageConfig(imagePath string) (image.Config, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return image.Config{}, err
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return image.Config{}, err
	}

	switch format {
	case "png", "jpeg", "gif":
		return config, nil
	default:
		return image.Config{}, fmt.Errorf("image placement only supports png, jpeg, and gif: %s", format)
	}
}

func loadXML(path string) (*etree.Document, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(path); err != nil {
		return nil, err
	}
	return doc, nil
}

func saveXML(doc *etree.Document, path string) error {
	doc.Indent(2)
	doc.WriteSettings = etree.WriteSettings{CanonicalEndTags: true}
	return doc.WriteToFile(path)
}

func findElementsByTag(root *etree.Element, tag string) []*etree.Element {
	if root == nil {
		return nil
	}

	var result []*etree.Element
	if tagMatches(root.Tag, tag) {
		result = append(result, root)
	}
	for _, child := range root.ChildElements() {
		result = append(result, findElementsByTag(child, tag)...)
	}
	return result
}

func childElementsByTag(root *etree.Element, tag string) []*etree.Element {
	if root == nil {
		return nil
	}

	var result []*etree.Element
	for _, child := range root.ChildElements() {
		if tagMatches(child.Tag, tag) {
			result = append(result, child)
		}
	}
	return result
}

func firstChildByTag(root *etree.Element, tag string) *etree.Element {
	for _, child := range root.ChildElements() {
		if tagMatches(child.Tag, tag) {
			return child
		}
	}
	return nil
}

func tableCellEntry(table *etree.Element, row, col int) (tableGridEntry, error) {
	if row < 0 || col < 0 {
		return tableGridEntry{}, fmt.Errorf("row and col must be zero or greater")
	}

	rowCount, colCount := tableDimensions(table)
	if row >= rowCount {
		return tableGridEntry{}, fmt.Errorf("row index out of range: %d", row)
	}
	if col >= colCount {
		return tableGridEntry{}, fmt.Errorf("col index out of range: %d", col)
	}

	grid, err := buildTableGrid(table)
	if err != nil {
		return tableGridEntry{}, err
	}
	entry, ok := grid[[2]int{row, col}]
	if !ok {
		return tableGridEntry{}, fmt.Errorf("cell coordinates are covered by a merged cell without an accessible anchor: (%d,%d)", row, col)
	}
	return entry, nil
}

func buildTableGrid(table *etree.Element) (map[[2]int]tableGridEntry, error) {
	grid := map[[2]int]tableGridEntry{}
	for _, rowElement := range childElementsByTag(table, "hp:tr") {
		for _, cell := range childElementsByTag(rowElement, "hp:tc") {
			addrRow, addrCol := tableCellAddress(cell)
			spanRow, spanCol := tableCellSpan(cell)
			if spanRow <= 0 {
				spanRow = 1
			}
			if spanCol <= 0 {
				spanCol = 1
			}
			deactivated := isDeactivatedTableCell(cell, spanRow, spanCol)
			for logicalRow := addrRow; logicalRow < addrRow+spanRow; logicalRow++ {
				for logicalCol := addrCol; logicalCol < addrCol+spanCol; logicalCol++ {
					key := [2]int{logicalRow, logicalCol}
					entry := tableGridEntry{
						cell:   cell,
						row:    logicalRow,
						col:    logicalCol,
						anchor: [2]int{addrRow, addrCol},
						span:   [2]int{spanRow, spanCol},
					}
					existing, exists := grid[key]
					if !exists {
						grid[key] = entry
						continue
					}
					if existing.cell == cell {
						continue
					}

					existingDeactivated := isDeactivatedTableCell(existing.cell, existing.span[0], existing.span[1])
					existingSpansMultiple := existing.span[0] != 1 || existing.span[1] != 1
					entrySpansMultiple := spanRow != 1 || spanCol != 1
					if deactivated && existingSpansMultiple {
						continue
					}
					if existingDeactivated && entrySpansMultiple {
						grid[key] = entry
						continue
					}
					return nil, fmt.Errorf("table grid contains overlapping cell spans")
				}
			}
		}
	}
	return grid, nil
}

func isDeactivatedTableCell(cell *etree.Element, spanRow, spanCol int) bool {
	if spanRow != 1 || spanCol != 1 {
		return false
	}
	if tableCellWidth(cell) != 0 || tableCellHeight(cell) != 0 {
		return false
	}
	return strings.TrimSpace(paragraphPlainText(cell)) == ""
}

func tableDimensions(table *etree.Element) (int, int) {
	rowCount, _ := strconv.Atoi(strings.TrimSpace(table.SelectAttrValue("rowCnt", "0")))
	colCount, _ := strconv.Atoi(strings.TrimSpace(table.SelectAttrValue("colCnt", "0")))
	if rowCount <= 0 {
		rowCount = len(childElementsByTag(table, "hp:tr"))
	}
	if colCount <= 0 {
		firstRow := firstChildByTag(table, "hp:tr")
		if firstRow != nil {
			colCount = len(childElementsByTag(firstRow, "hp:tc"))
		}
	}
	return rowCount, colCount
}

func tableCellAddress(cell *etree.Element) (int, int) {
	addr := firstChildByTag(cell, "hp:cellAddr")
	if addr == nil {
		return 0, 0
	}
	row, _ := strconv.Atoi(strings.TrimSpace(addr.SelectAttrValue("rowAddr", "0")))
	col, _ := strconv.Atoi(strings.TrimSpace(addr.SelectAttrValue("colAddr", "0")))
	return row, col
}

func setTableCellAddress(cell *etree.Element, row, col int) {
	addr := firstChildByTag(cell, "hp:cellAddr")
	if addr == nil {
		addr = etree.NewElement("hp:cellAddr")
		cell.AddChild(addr)
	}
	addr.RemoveAttr("rowAddr")
	addr.CreateAttr("rowAddr", strconv.Itoa(row))
	addr.RemoveAttr("colAddr")
	addr.CreateAttr("colAddr", strconv.Itoa(col))
}

func tableCellSpan(cell *etree.Element) (int, int) {
	span := firstChildByTag(cell, "hp:cellSpan")
	if span == nil {
		return 1, 1
	}
	rowSpan, _ := strconv.Atoi(strings.TrimSpace(span.SelectAttrValue("rowSpan", "1")))
	colSpan, _ := strconv.Atoi(strings.TrimSpace(span.SelectAttrValue("colSpan", "1")))
	if rowSpan <= 0 {
		rowSpan = 1
	}
	if colSpan <= 0 {
		colSpan = 1
	}
	return rowSpan, colSpan
}

func setTableCellSpan(cell *etree.Element, rowSpan, colSpan int) {
	span := firstChildByTag(cell, "hp:cellSpan")
	if span == nil {
		span = etree.NewElement("hp:cellSpan")
		cell.AddChild(span)
	}
	span.RemoveAttr("rowSpan")
	span.CreateAttr("rowSpan", strconv.Itoa(maxInt(rowSpan, 1)))
	span.RemoveAttr("colSpan")
	span.CreateAttr("colSpan", strconv.Itoa(maxInt(colSpan, 1)))
}

func tableCellWidth(cell *etree.Element) int {
	size := firstChildByTag(cell, "hp:cellSz")
	if size == nil {
		return 0
	}
	width, _ := strconv.Atoi(strings.TrimSpace(size.SelectAttrValue("width", "0")))
	return width
}

func tableCellHeight(cell *etree.Element) int {
	size := firstChildByTag(cell, "hp:cellSz")
	if size == nil {
		return 0
	}
	height, _ := strconv.Atoi(strings.TrimSpace(size.SelectAttrValue("height", "0")))
	return height
}

func setTableCellSize(cell *etree.Element, width, height int) {
	size := firstChildByTag(cell, "hp:cellSz")
	if size == nil {
		size = etree.NewElement("hp:cellSz")
		cell.AddChild(size)
	}
	size.RemoveAttr("width")
	size.CreateAttr("width", strconv.Itoa(maxInt(width, 0)))
	size.RemoveAttr("height")
	size.CreateAttr("height", strconv.Itoa(maxInt(height, 0)))
}

func clearTableCellText(cell *etree.Element) {
	for _, textElement := range findElementsByTag(cell, "hp:t") {
		textElement.SetText("")
	}
}

func distributeSize(total, count int) []int {
	if total <= 0 || count <= 0 {
		return nil
	}
	base := total / count
	remainder := total % count
	values := make([]int, count)
	for index := range values {
		values[index] = base
		if index == count-1 {
			values[index] += remainder
		}
	}
	return values
}

func physicalCellAt(rowElement *etree.Element, logicalRow, logicalCol int) *etree.Element {
	for _, cell := range childElementsByTag(rowElement, "hp:tc") {
		row, col := tableCellAddress(cell)
		if row == logicalRow && col == logicalCol {
			return cell
		}
	}
	return nil
}

func insertTableCell(rowElement, cell *etree.Element, logicalCol int) {
	existingCells := childElementsByTag(rowElement, "hp:tc")
	insertIndex := len(existingCells)
	for index, existing := range existingCells {
		_, col := tableCellAddress(existing)
		if col > logicalCol {
			insertIndex = index
			break
		}
	}
	rowElement.InsertChildAt(insertIndex, cell)
}

func tagMatches(actual, expected string) bool {
	if actual == expected {
		return true
	}
	return localTag(actual) == localTag(expected)
}

func localTag(value string) string {
	if index := strings.IndexByte(value, ':'); index >= 0 {
		return value[index+1:]
	}
	return value
}

type idCounter struct {
	next int
}

func newIDCounter(root *etree.Element) *idCounter {
	maxValue := 0
	for _, element := range findElementsByAttr(root, "id") {
		value, err := strconv.Atoi(element.SelectAttrValue("id", "0"))
		if err == nil && value > maxValue {
			maxValue = value
		}
	}
	return &idCounter{next: maxValue + 1}
}

func (c *idCounter) Next() string {
	value := c.next
	c.next++
	return strconv.Itoa(value)
}

func findElementsByAttr(root *etree.Element, attrName string) []*etree.Element {
	if root == nil {
		return nil
	}

	var result []*etree.Element
	if root.SelectAttr(attrName) != nil {
		result = append(result, root)
	}
	for _, child := range root.ChildElements() {
		result = append(result, findElementsByAttr(child, attrName)...)
	}
	return result
}

func newParagraphElement(counter *idCounter, text string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")
	textElement := run.CreateElement("hp:t")
	if text != "" {
		textElement.SetText(text)
	}

	return paragraph
}

func newStyledParagraphElement(counter *idCounter, style styleRef, text, bookmarkName string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", fallbackString(style.ParaPrIDRef, "0"))
	paragraph.CreateAttr("styleIDRef", fallbackString(style.ID, "0"))
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	charPrIDRef := fallbackString(style.CharPrIDRef, "0")
	if bookmarkName != "" {
		markerRun := paragraph.CreateElement("hp:run")
		markerRun.CreateAttr("charPrIDRef", charPrIDRef)
		markerCtrl := markerRun.CreateElement("hp:ctrl")
		bookmark := markerCtrl.CreateElement("hp:bookmark")
		bookmark.CreateAttr("name", bookmarkName)
	}

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", charPrIDRef)
	textElement := run.CreateElement("hp:t")
	textElement.SetText(text)

	paragraph.AddChild(newHeaderFooterLineSegElement(text))
	return paragraph
}

func newCellParagraphElement(counter *idCounter, text string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")
	textElement := run.CreateElement("hp:t")
	if text != "" {
		textElement.SetText(text)
	}
	return paragraph
}

func newTableParagraphElement(counter *idCounter, spec TableSpec) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")

	table := run.CreateElement("hp:tbl")
	table.CreateAttr("id", counter.Next())
	table.CreateAttr("zOrder", "0")
	table.CreateAttr("numberingType", "TABLE")
	table.CreateAttr("textWrap", "TOP_AND_BOTTOM")
	table.CreateAttr("textFlow", "BOTH_SIDES")
	table.CreateAttr("lock", "0")
	table.CreateAttr("dropcapstyle", "None")
	table.CreateAttr("pageBreak", "CELL")
	table.CreateAttr("repeatHeader", "0")
	table.CreateAttr("rowCnt", strconv.Itoa(spec.Rows))
	table.CreateAttr("colCnt", strconv.Itoa(spec.Cols))
	table.CreateAttr("cellSpacing", "0")
	table.CreateAttr("borderFillIDRef", "3")
	table.CreateAttr("noAdjust", "0")

	size := table.CreateElement("hp:sz")
	size.CreateAttr("width", strconv.Itoa(defaultTableWidth))
	size.CreateAttr("widthRelTo", "ABSOLUTE")
	size.CreateAttr("height", strconv.Itoa(spec.Rows*defaultCellHeight))
	size.CreateAttr("heightRelTo", "ABSOLUTE")
	size.CreateAttr("protect", "0")

	position := table.CreateElement("hp:pos")
	position.CreateAttr("treatAsChar", "1")
	position.CreateAttr("affectLSpacing", "0")
	position.CreateAttr("flowWithText", "1")
	position.CreateAttr("allowOverlap", "0")
	position.CreateAttr("holdAnchorAndSO", "0")
	position.CreateAttr("vertRelTo", "PARA")
	position.CreateAttr("horzRelTo", "COLUMN")
	position.CreateAttr("vertAlign", "TOP")
	position.CreateAttr("horzAlign", "LEFT")
	position.CreateAttr("vertOffset", "0")
	position.CreateAttr("horzOffset", "0")

	table.AddChild(newMarginElement("hp:outMargin"))
	table.AddChild(newMarginElement("hp:inMargin"))

	baseWidth := defaultTableWidth / spec.Cols
	remainder := defaultTableWidth % spec.Cols

	for rowIndex := 0; rowIndex < spec.Rows; rowIndex++ {
		rowElement := table.CreateElement("hp:tr")
		for colIndex := 0; colIndex < spec.Cols; colIndex++ {
			width := baseWidth
			if colIndex == spec.Cols-1 {
				width += remainder
			}

			cell := rowElement.CreateElement("hp:tc")
			cell.CreateAttr("name", "")
			cell.CreateAttr("header", "0")
			cell.CreateAttr("hasMargin", "0")
			cell.CreateAttr("protect", "0")
			cell.CreateAttr("editable", "0")
			cell.CreateAttr("dirty", "1")
			cell.CreateAttr("borderFillIDRef", "3")

			subList := cell.CreateElement("hp:subList")
			subList.CreateAttr("id", "")
			subList.CreateAttr("textDirection", "HORIZONTAL")
			subList.CreateAttr("lineWrap", "BREAK")
			subList.CreateAttr("vertAlign", "CENTER")
			subList.CreateAttr("linkListIDRef", "0")
			subList.CreateAttr("linkListNextIDRef", "0")
			subList.CreateAttr("textWidth", "0")
			subList.CreateAttr("textHeight", "0")
			subList.CreateAttr("hasTextRef", "0")
			subList.CreateAttr("hasNumRef", "0")

			cellText := ""
			if rowIndex < len(spec.Cells) && colIndex < len(spec.Cells[rowIndex]) {
				cellText = spec.Cells[rowIndex][colIndex]
			}
			subList.AddChild(newCellParagraphElement(counter, cellText))

			cellAddr := cell.CreateElement("hp:cellAddr")
			cellAddr.CreateAttr("colAddr", strconv.Itoa(colIndex))
			cellAddr.CreateAttr("rowAddr", strconv.Itoa(rowIndex))

			cellSpan := cell.CreateElement("hp:cellSpan")
			cellSpan.CreateAttr("colSpan", "1")
			cellSpan.CreateAttr("rowSpan", "1")

			cellSize := cell.CreateElement("hp:cellSz")
			cellSize.CreateAttr("width", strconv.Itoa(width))
			cellSize.CreateAttr("height", strconv.Itoa(defaultCellHeight))

			cell.AddChild(newMarginElement("hp:cellMargin"))
		}
	}

	return paragraph
}

func newMarginElement(tag string) *etree.Element {
	element := etree.NewElement(tag)
	element.CreateAttr("left", "0")
	element.CreateAttr("right", "0")
	element.CreateAttr("top", "0")
	element.CreateAttr("bottom", "0")
	return element
}

func newEquationParagraphElement(counter *idCounter, equationID, script string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")

	equation := run.CreateElement("hp:equation")
	equation.CreateAttr("id", equationID)
	equation.CreateAttr("zOrder", "0")
	equation.CreateAttr("numberingType", "EQUATION")
	equation.CreateAttr("textWrap", "TOP_AND_BOTTOM")
	equation.CreateAttr("textFlow", "BOTH_SIDES")
	equation.CreateAttr("lock", "0")
	equation.CreateAttr("dropcapstyle", "None")
	equation.CreateAttr("version", defaultEquationVer)
	equation.CreateAttr("baseLine", "0")
	equation.CreateAttr("textColor", "#000000")
	equation.CreateAttr("baseUnit", "1000")
	equation.CreateAttr("lineMode", "CHAR")
	equation.CreateAttr("font", defaultEquationFont)

	size := equation.CreateElement("hp:sz")
	size.CreateAttr("width", "0")
	size.CreateAttr("widthRelTo", "ABSOLUTE")
	size.CreateAttr("height", "0")
	size.CreateAttr("heightRelTo", "ABSOLUTE")
	size.CreateAttr("protect", "0")

	position := equation.CreateElement("hp:pos")
	position.CreateAttr("treatAsChar", "1")
	position.CreateAttr("affectLSpacing", "1")
	position.CreateAttr("flowWithText", "1")
	position.CreateAttr("allowOverlap", "0")
	position.CreateAttr("holdAnchorAndSO", "0")
	position.CreateAttr("vertRelTo", "PARA")
	position.CreateAttr("horzRelTo", "COLUMN")
	position.CreateAttr("vertAlign", "TOP")
	position.CreateAttr("horzAlign", "LEFT")
	position.CreateAttr("vertOffset", "0")
	position.CreateAttr("horzOffset", "0")

	equation.AddChild(newMarginElement("hp:outMargin"))

	equationScript := equation.CreateElement("hp:script")
	equationScript.SetText(script)

	paragraph.AddChild(newHeaderFooterLineSegElement(script))
	return paragraph
}

func newHeaderFooterControlElement(tag string, spec HeaderFooterSpec, counter *idCounter) *etree.Element {
	ctrl := etree.NewElement("hp:ctrl")

	element := ctrl.CreateElement("hp:" + tag)
	element.CreateAttr("id", "")
	element.CreateAttr("applyPageType", spec.ApplyPageType)

	subList := element.CreateElement("hp:subList")
	subList.CreateAttr("id", "")
	subList.CreateAttr("textDirection", "HORIZONTAL")
	subList.CreateAttr("lineWrap", "BREAK")
	if tag == "header" {
		subList.CreateAttr("vertAlign", "TOP")
	} else {
		subList.CreateAttr("vertAlign", "BOTTOM")
	}
	subList.CreateAttr("linkListIDRef", "0")
	subList.CreateAttr("linkListNextIDRef", "0")
	subList.CreateAttr("textWidth", "0")
	subList.CreateAttr("textHeight", "0")
	subList.CreateAttr("hasTextRef", "0")
	subList.CreateAttr("hasNumRef", "0")

	for _, text := range spec.Text {
		subList.AddChild(newHeaderFooterParagraphElement(counter, text))
	}

	return ctrl
}

func newPageNumControlElement(spec PageNumberSpec) *etree.Element {
	ctrl := etree.NewElement("hp:ctrl")
	pageNum := ctrl.CreateElement("hp:pageNum")
	pageNum.CreateAttr("pos", spec.Position)
	pageNum.CreateAttr("formatType", spec.FormatType)
	pageNum.CreateAttr("sideChar", spec.SideChar)
	return ctrl
}

func newNoteParagraphElement(counter *idCounter, tag string, spec NoteSpec, noteNumber int) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	textRun := paragraph.CreateElement("hp:run")
	textRun.CreateAttr("charPrIDRef", "0")
	textElement := textRun.CreateElement("hp:t")
	textElement.SetText(spec.AnchorText)

	noteRun := paragraph.CreateElement("hp:run")
	noteRun.CreateAttr("charPrIDRef", "0")
	noteRun.AddChild(newNoteControlElement(counter, tag, spec, noteNumber))

	paragraph.AddChild(newHeaderFooterLineSegElement(spec.AnchorText + "00"))
	return paragraph
}

func newBookmarkParagraphElement(counter *idCounter, spec BookmarkSpec) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	markerRun := paragraph.CreateElement("hp:run")
	markerRun.CreateAttr("charPrIDRef", "0")
	markerCtrl := markerRun.CreateElement("hp:ctrl")
	bookmark := markerCtrl.CreateElement("hp:bookmark")
	bookmark.CreateAttr("name", spec.Name)

	textRun := paragraph.CreateElement("hp:run")
	textRun.CreateAttr("charPrIDRef", "0")
	textElement := textRun.CreateElement("hp:t")
	textElement.SetText(spec.Text)

	paragraph.AddChild(newHeaderFooterLineSegElement(spec.Text))
	return paragraph
}

func newHyperlinkParagraphElement(counter *idCounter, fieldID string, spec HyperlinkSpec) *etree.Element {
	return newHyperlinkStyledParagraphElementWithFieldID(counter, styleRef{
		ID:          "0",
		ParaPrIDRef: "0",
		CharPrIDRef: "0",
	}, fieldID, spec.Target, spec.Text)
}

func newHyperlinkStyledParagraphElement(counter *idCounter, style styleRef, target, text string) *etree.Element {
	return newHyperlinkStyledParagraphElementWithFieldID(counter, style, counter.Next(), target, text)
}

func newHyperlinkStyledParagraphElementWithFieldID(counter *idCounter, style styleRef, fieldID, target, text string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", fallbackString(style.ParaPrIDRef, "0"))
	paragraph.CreateAttr("styleIDRef", fallbackString(style.ID, "0"))
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")
	charPrIDRef := fallbackString(style.CharPrIDRef, "0")

	beginRun := paragraph.CreateElement("hp:run")
	beginRun.CreateAttr("charPrIDRef", charPrIDRef)
	beginCtrl := beginRun.CreateElement("hp:ctrl")
	fieldBegin := beginCtrl.CreateElement("hp:fieldBegin")
	fieldBegin.CreateAttr("id", fieldID)
	fieldBegin.CreateAttr("type", "HYPERLINK")
	fieldBegin.CreateAttr("name", strings.TrimSpace(target))
	fieldBegin.CreateAttr("editable", "false")
	fieldBegin.CreateAttr("dirty", "false")
	fieldBegin.CreateAttr("fieldid", fieldID)

	parameters := fieldBegin.CreateElement("hp:parameters")
	parameters.CreateAttr("count", "1")
	parameters.CreateAttr("name", "")
	command := parameters.CreateElement("hp:stringParam")
	command.CreateAttr("name", "Command")
	command.SetText(strings.TrimSpace(target))

	textRun := paragraph.CreateElement("hp:run")
	textRun.CreateAttr("charPrIDRef", charPrIDRef)
	textElement := textRun.CreateElement("hp:t")
	textElement.SetText(text)

	endRun := paragraph.CreateElement("hp:run")
	endRun.CreateAttr("charPrIDRef", charPrIDRef)
	endCtrl := endRun.CreateElement("hp:ctrl")
	fieldEnd := endCtrl.CreateElement("hp:fieldEnd")
	fieldEnd.CreateAttr("beginIDRef", fieldID)

	paragraph.AddChild(newHeaderFooterLineSegElement(text))
	return paragraph
}

func newNoteControlElement(counter *idCounter, tag string, spec NoteSpec, noteNumber int) *etree.Element {
	ctrl := etree.NewElement("hp:ctrl")
	note := ctrl.CreateElement("hp:" + tag)
	note.CreateAttr("number", strconv.Itoa(noteNumber))
	note.CreateAttr("instId", counter.Next())

	subList := note.CreateElement("hp:subList")
	subList.CreateAttr("id", "")
	subList.CreateAttr("textDirection", "HORIZONTAL")
	subList.CreateAttr("lineWrap", "BREAK")
	subList.CreateAttr("vertAlign", "TOP")
	subList.CreateAttr("linkListIDRef", "0")
	subList.CreateAttr("linkListNextIDRef", "0")
	subList.CreateAttr("textWidth", "0")
	subList.CreateAttr("textHeight", "0")
	subList.CreateAttr("hasTextRef", "0")
	subList.CreateAttr("hasNumRef", "1")

	for index, text := range spec.Text {
		subList.AddChild(newNoteBodyParagraphElement(counter, tag, noteNumber, index == 0, text))
	}

	return ctrl
}

func newNoteBodyParagraphElement(counter *idCounter, tag string, noteNumber int, includeNumber bool, text string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	if includeNumber {
		numberRun := paragraph.CreateElement("hp:run")
		numberRun.CreateAttr("charPrIDRef", "0")
		numberCtrl := numberRun.CreateElement("hp:ctrl")
		numberCtrl.AddChild(newNoteAutoNumElement(tag, noteNumber))
	}

	textRun := paragraph.CreateElement("hp:run")
	textRun.CreateAttr("charPrIDRef", "0")
	textElement := textRun.CreateElement("hp:t")
	if includeNumber {
		textElement.SetText(" " + text)
	} else {
		textElement.SetText(text)
	}

	paragraph.AddChild(newHeaderFooterLineSegElement(text + "00"))
	return paragraph
}

func newNoteAutoNumElement(tag string, noteNumber int) *etree.Element {
	numType := "FOOTNOTE"
	if tag == "endNote" {
		numType = "ENDNOTE"
	}

	autoNum := etree.NewElement("hp:autoNum")
	autoNum.CreateAttr("num", strconv.Itoa(noteNumber))
	autoNum.CreateAttr("numType", numType)

	format := autoNum.CreateElement("hp:autoNumFormat")
	format.CreateAttr("type", "DIGIT")
	format.CreateAttr("userChar", "")
	format.CreateAttr("prefixChar", "")
	format.CreateAttr("suffixChar", ")")
	format.CreateAttr("supscript", "0")

	return autoNum
}

func newHeaderFooterParagraphElement(counter *idCounter, text string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	segments := splitHeaderFooterSegments(text)
	for _, segment := range segments {
		run := paragraph.CreateElement("hp:run")
		run.CreateAttr("charPrIDRef", "0")
		if segment.token == "" {
			textElement := run.CreateElement("hp:t")
			textElement.SetText(segment.text)
			continue
		}

		ctrl := run.CreateElement("hp:ctrl")
		ctrl.AddChild(newAutoNumElement(segment.token))
	}

	paragraph.AddChild(newHeaderFooterLineSegElement(text))
	return paragraph
}

func newHeaderFooterLineSegElement(text string) *etree.Element {
	lineSegArray := etree.NewElement("hp:linesegarray")
	lineSeg := lineSegArray.CreateElement("hp:lineseg")
	lineSeg.CreateAttr("textpos", "0")
	lineSeg.CreateAttr("vertpos", "0")
	lineSeg.CreateAttr("vertsize", "1200")
	lineSeg.CreateAttr("textheight", "1200")
	lineSeg.CreateAttr("baseline", "1020")
	lineSeg.CreateAttr("spacing", "720")
	lineSeg.CreateAttr("horzpos", "0")
	lineSeg.CreateAttr("horzsize", strconv.Itoa(maxInt(defaultTableWidth, len([]rune(headerFooterDisplayText(text)))*900)))
	lineSeg.CreateAttr("flags", "393216")
	return lineSegArray
}

func maxInt(left, right int) int {
	if left >= right {
		return left
	}
	return right
}

func minInt(left, right int) int {
	if left <= right {
		return left
	}
	return right
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

type headerFooterSegment struct {
	text  string
	token string
}

func splitHeaderFooterSegments(text string) []headerFooterSegment {
	if text == "" {
		return []headerFooterSegment{{text: ""}}
	}

	var segments []headerFooterSegment
	remaining := text
	for remaining != "" {
		index, token := nextHeaderFooterToken(remaining)
		if index < 0 {
			segments = append(segments, headerFooterSegment{text: remaining})
			break
		}
		if index > 0 {
			segments = append(segments, headerFooterSegment{text: remaining[:index]})
		}
		segments = append(segments, headerFooterSegment{token: token})
		remaining = remaining[index+len(token):]
	}

	if len(segments) == 0 {
		return []headerFooterSegment{{text: ""}}
	}
	return segments
}

func nextHeaderFooterToken(text string) (int, string) {
	pageIndex := strings.Index(text, pageToken)
	totalIndex := strings.Index(text, totalPageToken)

	switch {
	case pageIndex < 0 && totalIndex < 0:
		return -1, ""
	case pageIndex < 0:
		return totalIndex, totalPageToken
	case totalIndex < 0:
		return pageIndex, pageToken
	case pageIndex <= totalIndex:
		return pageIndex, pageToken
	default:
		return totalIndex, totalPageToken
	}
}

func headerFooterDisplayText(text string) string {
	replacer := strings.NewReplacer(
		pageToken, "0000",
		totalPageToken, "0000",
	)
	return replacer.Replace(text)
}

func newAutoNumElement(token string) *etree.Element {
	numType := "PAGE"
	if token == totalPageToken {
		numType = "TOTAL_PAGE"
	}

	autoNum := etree.NewElement("hp:autoNum")
	autoNum.CreateAttr("num", "1")
	autoNum.CreateAttr("numType", numType)

	format := autoNum.CreateElement("hp:autoNumFormat")
	format.CreateAttr("type", "DIGIT")
	format.CreateAttr("userChar", "")
	format.CreateAttr("prefixChar", "")
	format.CreateAttr("suffixChar", "")
	format.CreateAttr("supscript", "0")

	return autoNum
}

func nextNoteNumber(root *etree.Element, tag string) int {
	maxNumber := 0
	for _, element := range findElementsByTag(root, "hp:"+tag) {
		value, err := strconv.Atoi(element.SelectAttrValue("number", "0"))
		if err == nil && value > maxNumber {
			maxNumber = value
		}
	}
	return maxNumber + 1
}

func bookmarkExists(root *etree.Element, name string) bool {
	for _, element := range findElementsByTag(root, "hp:bookmark") {
		if element.SelectAttrValue("name", "") == name {
			return true
		}
	}
	return false
}

func nextMemoNumber(root *etree.Element) int {
	maxNumber := 0
	for _, element := range findElementsByTag(root, "hp:fieldBegin") {
		if element.SelectAttrValue("type", "") != "MEMO" {
			continue
		}
		parameters := firstChildByTag(element, "hp:parameters")
		if parameters == nil {
			continue
		}
		for _, param := range childElementsByTag(parameters, "hp:integerParam") {
			if param.SelectAttrValue("name", "") != "Number" {
				continue
			}
			value, err := strconv.Atoi(strings.TrimSpace(param.Text()))
			if err == nil && value > maxNumber {
				maxNumber = value
			}
		}
	}
	return maxNumber + 1
}

func ensureMemoGroup(root *etree.Element) *etree.Element {
	memoGroup := firstChildByTag(root, "hp:memogroup")
	if memoGroup != nil {
		return memoGroup
	}

	memoGroup = etree.NewElement("hp:memogroup")
	root.AddChild(memoGroup)
	return memoGroup
}

func newPictureParagraphElement(counter *idCounter, itemID, sourceName string, pixelWidth, pixelHeight, width, height int) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")

	pictureID := counter.Next()
	picture := run.CreateElement("hp:pic")
	picture.CreateAttr("id", pictureID)
	picture.CreateAttr("zOrder", "0")
	picture.CreateAttr("numberingType", "PICTURE")
	picture.CreateAttr("textWrap", "TOP_AND_BOTTOM")
	picture.CreateAttr("textFlow", "BOTH_SIDES")
	picture.CreateAttr("lock", "0")
	picture.CreateAttr("dropcapstyle", "None")
	picture.CreateAttr("href", "")
	picture.CreateAttr("groupLevel", "0")
	picture.CreateAttr("instid", pictureID)
	picture.CreateAttr("reverse", "0")

	offset := picture.CreateElement("hp:offset")
	offset.CreateAttr("x", "0")
	offset.CreateAttr("y", "0")

	originalWidth := pixelWidth * 75
	originalHeight := pixelHeight * 75
	if originalWidth <= 0 {
		originalWidth = width
	}
	if originalHeight <= 0 {
		originalHeight = height
	}

	orgSize := picture.CreateElement("hp:orgSz")
	orgSize.CreateAttr("width", strconv.Itoa(originalWidth))
	orgSize.CreateAttr("height", strconv.Itoa(originalHeight))

	currentSize := picture.CreateElement("hp:curSz")
	currentSize.CreateAttr("width", strconv.Itoa(width))
	currentSize.CreateAttr("height", strconv.Itoa(height))

	flip := picture.CreateElement("hp:flip")
	flip.CreateAttr("horizontal", "0")
	flip.CreateAttr("vertical", "0")

	rotation := picture.CreateElement("hp:rotationInfo")
	rotation.CreateAttr("angle", "0")
	rotation.CreateAttr("centerX", strconv.Itoa(width/2))
	rotation.CreateAttr("centerY", strconv.Itoa(height/2))
	rotation.CreateAttr("rotateimage", "1")

	renderingInfo := picture.CreateElement("hp:renderingInfo")
	renderingInfo.AddChild(newMatrixElement("hc:transMatrix"))
	renderingInfo.AddChild(newScaleMatrixElement(width, height, originalWidth, originalHeight))
	renderingInfo.AddChild(newMatrixElement("hc:rotMatrix"))

	imageRef := picture.CreateElement("hc:img")
	imageRef.CreateAttr("binaryItemIDRef", itemID)
	imageRef.CreateAttr("bright", "0")
	imageRef.CreateAttr("contrast", "0")
	imageRef.CreateAttr("effect", "REAL_PIC")
	imageRef.CreateAttr("alpha", "0")

	imageRect := picture.CreateElement("hp:imgRect")
	appendPoint(imageRect, "hc:pt0", 0, 0)
	appendPoint(imageRect, "hc:pt1", originalWidth, 0)
	appendPoint(imageRect, "hc:pt2", originalWidth, originalHeight)
	appendPoint(imageRect, "hc:pt3", 0, originalHeight)

	imageClip := picture.CreateElement("hp:imgClip")
	imageClip.CreateAttr("left", "0")
	imageClip.CreateAttr("right", strconv.Itoa(originalWidth))
	imageClip.CreateAttr("top", "0")
	imageClip.CreateAttr("bottom", strconv.Itoa(originalHeight))

	picture.AddChild(newMarginElement("hp:inMargin"))

	imageDim := picture.CreateElement("hp:imgDim")
	imageDim.CreateAttr("dimwidth", strconv.Itoa(originalWidth))
	imageDim.CreateAttr("dimheight", strconv.Itoa(originalHeight))

	picture.CreateElement("hp:effects")

	size := picture.CreateElement("hp:sz")
	size.CreateAttr("width", strconv.Itoa(width))
	size.CreateAttr("widthRelTo", "ABSOLUTE")
	size.CreateAttr("height", strconv.Itoa(height))
	size.CreateAttr("heightRelTo", "ABSOLUTE")
	size.CreateAttr("protect", "0")

	position := picture.CreateElement("hp:pos")
	position.CreateAttr("treatAsChar", "1")
	position.CreateAttr("affectLSpacing", "0")
	position.CreateAttr("flowWithText", "1")
	position.CreateAttr("allowOverlap", "0")
	position.CreateAttr("holdAnchorAndSO", "0")
	position.CreateAttr("vertRelTo", "PARA")
	position.CreateAttr("horzRelTo", "COLUMN")
	position.CreateAttr("vertAlign", "TOP")
	position.CreateAttr("horzAlign", "LEFT")
	position.CreateAttr("vertOffset", "0")
	position.CreateAttr("horzOffset", "0")

	picture.AddChild(newMarginElement("hp:outMargin"))

	shapeComment := picture.CreateElement("hp:shapeComment")
	shapeComment.SetText(fmt.Sprintf("그림입니다.\n원본 그림의 이름: %s\n원본 그림의 크기: 가로 %dpixel, 세로 %dpixel", sourceName, pixelWidth, pixelHeight))

	run.CreateElement("hp:t")
	paragraph.AddChild(newPictureLineSegElement(width, height))
	return paragraph
}

func newRectangleParagraphElement(counter *idCounter, shapeID string, width, height int, lineColor, fillColor string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")

	rect := run.CreateElement("hp:rect")
	rect.CreateAttr("id", shapeID)
	rect.CreateAttr("zOrder", "0")
	rect.CreateAttr("numberingType", "NONE")
	rect.CreateAttr("lock", "0")
	rect.CreateAttr("dropcapstyle", "None")
	rect.CreateAttr("href", "")
	rect.CreateAttr("groupLevel", "0")
	rect.CreateAttr("instid", shapeID)
	rect.CreateAttr("ratio", "0")

	offset := rect.CreateElement("hp:offset")
	offset.CreateAttr("x", "0")
	offset.CreateAttr("y", "0")

	orgSize := rect.CreateElement("hp:orgSz")
	orgSize.CreateAttr("width", strconv.Itoa(width))
	orgSize.CreateAttr("height", strconv.Itoa(height))

	currentSize := rect.CreateElement("hp:curSz")
	currentSize.CreateAttr("width", strconv.Itoa(width))
	currentSize.CreateAttr("height", strconv.Itoa(height))

	flip := rect.CreateElement("hp:flip")
	flip.CreateAttr("horizontal", "0")
	flip.CreateAttr("vertical", "0")

	rotation := rect.CreateElement("hp:rotationInfo")
	rotation.CreateAttr("angle", "0")
	rotation.CreateAttr("centerX", strconv.Itoa(width/2))
	rotation.CreateAttr("centerY", strconv.Itoa(height/2))
	rotation.CreateAttr("rotateimage", "1")

	renderingInfo := rect.CreateElement("hp:renderingInfo")
	renderingInfo.AddChild(newMatrixElement("hc:transMatrix"))
	renderingInfo.AddChild(newMatrixElement("hc:scaMatrix"))
	renderingInfo.AddChild(newMatrixElement("hc:rotMatrix"))

	lineShape := rect.CreateElement("hp:lineShape")
	lineShape.CreateAttr("color", lineColor)
	lineShape.CreateAttr("width", "283")
	lineShape.CreateAttr("style", "SOLID")
	lineShape.CreateAttr("endCap", "FLAT")
	lineShape.CreateAttr("headStyle", "NORMAL")
	lineShape.CreateAttr("tailStyle", "NORMAL")
	lineShape.CreateAttr("outlineStyle", "NORMAL")

	fillBrush := rect.CreateElement("hp:fillBrush")
	winBrush := fillBrush.CreateElement("hc:winBrush")
	winBrush.CreateAttr("faceColor", fillColor)
	winBrush.CreateAttr("hatchColor", "#FFFFFF")

	shadow := rect.CreateElement("hp:shadow")
	shadow.CreateAttr("type", "NONE")
	shadow.CreateAttr("color", "#B2B2B2")
	shadow.CreateAttr("offsetX", "0")
	shadow.CreateAttr("offsetY", "0")
	shadow.CreateAttr("alpha", "0")

	appendPoint(rect, "hc:pt0", 0, 0)
	appendPoint(rect, "hc:pt1", width, 0)
	appendPoint(rect, "hc:pt2", width, height)
	appendPoint(rect, "hc:pt3", 0, height)

	size := rect.CreateElement("hp:sz")
	size.CreateAttr("width", strconv.Itoa(width))
	size.CreateAttr("widthRelTo", "ABSOLUTE")
	size.CreateAttr("height", strconv.Itoa(height))
	size.CreateAttr("heightRelTo", "ABSOLUTE")
	size.CreateAttr("protect", "0")

	position := rect.CreateElement("hp:pos")
	position.CreateAttr("treatAsChar", "1")
	position.CreateAttr("affectLSpacing", "0")
	position.CreateAttr("flowWithText", "1")
	position.CreateAttr("allowOverlap", "0")
	position.CreateAttr("holdAnchorAndSO", "0")
	position.CreateAttr("vertRelTo", "PARA")
	position.CreateAttr("horzRelTo", "COLUMN")
	position.CreateAttr("vertAlign", "TOP")
	position.CreateAttr("horzAlign", "LEFT")
	position.CreateAttr("vertOffset", "0")
	position.CreateAttr("horzOffset", "0")

	rect.AddChild(newMarginElement("hp:outMargin"))

	run.CreateElement("hp:t")
	paragraph.AddChild(newPictureLineSegElement(width, height))
	return paragraph
}

func newMemoElement(counter *idCounter, memoID string, spec MemoSpec) *etree.Element {
	memo := etree.NewElement("hp:memo")
	memo.CreateAttr("id", memoID)
	memo.CreateAttr("memoShapeIDRef", "0")

	for _, text := range spec.Text {
		memo.AddChild(newMemoParagraphElement(counter, text))
	}
	return memo
}

func newEmptyTableCellElement(counter *idCounter, row, col, width, height int, borderFillIDRef string) *etree.Element {
	cell := etree.NewElement("hp:tc")
	cell.CreateAttr("name", "")
	cell.CreateAttr("header", "0")
	cell.CreateAttr("hasMargin", "0")
	cell.CreateAttr("protect", "0")
	cell.CreateAttr("editable", "0")
	cell.CreateAttr("dirty", "1")
	cell.CreateAttr("borderFillIDRef", borderFillIDRef)

	subList := cell.CreateElement("hp:subList")
	subList.CreateAttr("id", "")
	subList.CreateAttr("textDirection", "HORIZONTAL")
	subList.CreateAttr("lineWrap", "BREAK")
	subList.CreateAttr("vertAlign", "CENTER")
	subList.CreateAttr("linkListIDRef", "0")
	subList.CreateAttr("linkListNextIDRef", "0")
	subList.CreateAttr("textWidth", "0")
	subList.CreateAttr("textHeight", "0")
	subList.CreateAttr("hasTextRef", "0")
	subList.CreateAttr("hasNumRef", "0")
	subList.AddChild(newCellParagraphElement(counter, ""))

	cellAddr := cell.CreateElement("hp:cellAddr")
	cellAddr.CreateAttr("colAddr", strconv.Itoa(col))
	cellAddr.CreateAttr("rowAddr", strconv.Itoa(row))

	cellSpan := cell.CreateElement("hp:cellSpan")
	cellSpan.CreateAttr("colSpan", "1")
	cellSpan.CreateAttr("rowSpan", "1")

	cellSize := cell.CreateElement("hp:cellSz")
	cellSize.CreateAttr("width", strconv.Itoa(width))
	cellSize.CreateAttr("height", strconv.Itoa(height))

	cell.AddChild(newMarginElement("hp:cellMargin"))
	return cell
}

func newMemoParagraphElement(counter *idCounter, text string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")
	textElement := run.CreateElement("hp:t")
	textElement.SetText(text)

	paragraph.AddChild(newHeaderFooterLineSegElement(text))
	return paragraph
}

func newMemoAnchorParagraphElement(counter *idCounter, memoID, fieldID string, memoNumber int, spec MemoSpec) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	beginRun := paragraph.CreateElement("hp:run")
	beginRun.CreateAttr("charPrIDRef", "0")
	beginCtrl := beginRun.CreateElement("hp:ctrl")
	fieldBegin := beginCtrl.CreateElement("hp:fieldBegin")
	fieldBegin.CreateAttr("id", fieldID)
	fieldBegin.CreateAttr("type", "MEMO")
	fieldBegin.CreateAttr("editable", "true")
	fieldBegin.CreateAttr("dirty", "false")
	fieldBegin.CreateAttr("fieldid", fieldID)

	parameters := fieldBegin.CreateElement("hp:parameters")
	parameters.CreateAttr("count", "5")
	parameters.CreateAttr("name", "")

	idParam := parameters.CreateElement("hp:stringParam")
	idParam.CreateAttr("name", "ID")
	idParam.SetText(memoID)

	numberParam := parameters.CreateElement("hp:integerParam")
	numberParam.CreateAttr("name", "Number")
	numberParam.SetText(strconv.Itoa(maxInt(memoNumber, 1)))

	dateParam := parameters.CreateElement("hp:stringParam")
	dateParam.CreateAttr("name", "CreateDateTime")
	dateParam.SetText(time.Now().Format("2006-01-02 15:04:05"))

	authorParam := parameters.CreateElement("hp:stringParam")
	authorParam.CreateAttr("name", "Author")
	authorParam.SetText(strings.TrimSpace(spec.Author))

	shapeParam := parameters.CreateElement("hp:stringParam")
	shapeParam.CreateAttr("name", "MemoShapeID")
	shapeParam.SetText("0")

	subList := fieldBegin.CreateElement("hp:subList")
	subList.CreateAttr("id", "")
	subList.CreateAttr("textDirection", "HORIZONTAL")
	subList.CreateAttr("lineWrap", "BREAK")
	subList.CreateAttr("vertAlign", "TOP")
	subList.CreateAttr("linkListIDRef", "0")
	subList.CreateAttr("linkListNextIDRef", "0")
	subList.CreateAttr("textWidth", "0")
	subList.CreateAttr("textHeight", "0")
	subList.CreateAttr("hasTextRef", "0")
	subList.CreateAttr("hasNumRef", "0")

	subParagraph := etree.NewElement("hp:p")
	subParagraph.CreateAttr("id", counter.Next())
	subParagraph.CreateAttr("paraPrIDRef", "0")
	subParagraph.CreateAttr("styleIDRef", "0")
	subParagraph.CreateAttr("pageBreak", "0")
	subParagraph.CreateAttr("columnBreak", "0")
	subParagraph.CreateAttr("merged", "0")
	subRun := subParagraph.CreateElement("hp:run")
	subRun.CreateAttr("charPrIDRef", "0")
	subText := subRun.CreateElement("hp:t")
	subText.SetText(memoID)
	subList.AddChild(subParagraph)

	textRun := paragraph.CreateElement("hp:run")
	textRun.CreateAttr("charPrIDRef", "0")
	textElement := textRun.CreateElement("hp:t")
	textElement.SetText(spec.AnchorText)

	endRun := paragraph.CreateElement("hp:run")
	endRun.CreateAttr("charPrIDRef", "0")
	endCtrl := endRun.CreateElement("hp:ctrl")
	fieldEnd := endCtrl.CreateElement("hp:fieldEnd")
	fieldEnd.CreateAttr("beginIDRef", fieldID)
	fieldEnd.CreateAttr("fieldid", fieldID)

	paragraph.AddChild(newHeaderFooterLineSegElement(spec.AnchorText + "00"))
	return paragraph
}

func newMatrixElement(tag string) *etree.Element {
	matrix := etree.NewElement(tag)
	matrix.CreateAttr("e1", "1")
	matrix.CreateAttr("e2", "0")
	matrix.CreateAttr("e3", "0")
	matrix.CreateAttr("e4", "0")
	matrix.CreateAttr("e5", "1")
	matrix.CreateAttr("e6", "0")
	return matrix
}

func newScaleMatrixElement(width, height, originalWidth, originalHeight int) *etree.Element {
	matrix := etree.NewElement("hc:scaMatrix")
	matrix.CreateAttr("e1", formatMatrixValue(width, originalWidth))
	matrix.CreateAttr("e2", "0")
	matrix.CreateAttr("e3", "0")
	matrix.CreateAttr("e4", "0")
	matrix.CreateAttr("e5", formatMatrixValue(height, originalHeight))
	matrix.CreateAttr("e6", "0")
	return matrix
}

func formatMatrixValue(current, original int) string {
	if current <= 0 || original <= 0 {
		return "1"
	}
	return strconv.FormatFloat(float64(current)/float64(original), 'f', 6, 64)
}

func newPictureLineSegElement(width, height int) *etree.Element {
	lineSegArray := etree.NewElement("hp:linesegarray")
	lineSeg := lineSegArray.CreateElement("hp:lineseg")
	lineSeg.CreateAttr("textpos", "0")
	lineSeg.CreateAttr("vertpos", "0")
	lineSeg.CreateAttr("vertsize", strconv.Itoa(height))
	lineSeg.CreateAttr("textheight", strconv.Itoa(height))
	lineSeg.CreateAttr("baseline", strconv.Itoa(int(float64(height)*0.85+0.5)))
	lineSeg.CreateAttr("spacing", "600")
	lineSeg.CreateAttr("horzpos", "0")
	lineSeg.CreateAttr("horzsize", strconv.Itoa(width))
	lineSeg.CreateAttr("flags", "393216")
	return lineSegArray
}

func appendPoint(parent *etree.Element, tag string, x, y int) {
	point := parent.CreateElement(tag)
	point.CreateAttr("x", strconv.Itoa(x))
	point.CreateAttr("y", strconv.Itoa(y))
}

func calculateImageSize(pixelWidth, pixelHeight int, widthMM float64) (int, int) {
	width := defaultImageWidth
	if widthMM > 0 {
		width = int(widthMM*7200.0/25.4 + 0.5)
	}
	if width <= 0 {
		width = defaultImageWidth
	}
	if width > defaultTableWidth {
		width = defaultTableWidth
	}

	height := int(float64(width)*float64(pixelHeight)/float64(pixelWidth) + 0.5)
	if height <= 0 {
		height = width
	}
	return width, height
}

func mmToHWPUnit(value float64) int {
	if value <= 0 {
		return 0
	}
	return int(value*7200.0/25.4 + 0.5)
}

func PrintToPDF(inputPath, outputPath, workspaceDir string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("print-pdf is only supported on macOS")
	}
	if filepath.Ext(strings.ToLower(outputPath)) != ".pdf" {
		return fmt.Errorf("output path must end with .pdf")
	}
	if _, err := os.Stat("/Applications/Hancom Office HWP Viewer.app"); err != nil {
		return fmt.Errorf("Hancom Office HWP Viewer.app is required: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	stageDir := workspaceDir
	if stageDir == "" {
		stageDir = filepath.Dir(outputPath)
	}
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return err
	}

	stageBase := fmt.Sprintf("hwpxctl-print-%d", time.Now().UnixNano())
	stageFile := filepath.Join(stageDir, stageBase+".pdf")
	_ = os.Remove(stageFile)

	workspaceName := filepath.Base(stageDir)
	sourceDir := filepath.Base(filepath.Dir(inputPath))
	docName := filepath.Base(inputPath)

	if err := runHancomPrintScript(inputPath, docName, workspaceName, sourceDir, stageBase); err != nil {
		return err
	}

	foundPath, err := waitForPrintedPDF(stageBase+".pdf", stageDir, filepath.Dir(inputPath))
	if err != nil {
		return err
	}
	defer os.Remove(foundPath)

	if err := os.Rename(foundPath, outputPath); err == nil {
		return nil
	}
	if err := copyFile(foundPath, outputPath); err != nil {
		return err
	}
	return os.Remove(foundPath)
}

func runHancomPrintScript(inputPath, docName, workspaceName, sourceDir, stageBase string) error {
	script := `
on clickMenuItemIfExists(theButton, itemName)
	tell application "System Events"
		tell process "Hancom Office HWP Viewer"
			try
				click theButton
				delay 0.4
				click menu item itemName of menu 1 of theButton
				return true
			on error
				return false
			end try
		end tell
	end tell
end clickMenuItemIfExists

on describeOpenWindows()
	tell application "System Events"
		tell process "Hancom Office HWP Viewer"
			set windowDescriptions to {}
			repeat with w in windows
				set windowName to ""
				set windowText to ""
				try
					set windowName to name of w
				end try
				try
					set textValues to value of static texts of w
					if (count of textValues) > 0 then
						set windowText to item 1 of textValues
					end if
				end try
				if windowText is not "" then
					set end of windowDescriptions to windowName & ": " & windowText
				else
					set end of windowDescriptions to windowName
				end if
			end repeat
			return windowDescriptions as string
		end tell
	end tell
end describeOpenWindows

on run argv
	set inputPath to item 1 of argv
	set docName to item 2 of argv
	set workspaceName to item 3 of argv
	set sourceDirName to item 4 of argv
	set stageBase to item 5 of argv

	try
		tell application "Hancom Office HWP Viewer" to quit
	end try
	delay 2
	do shell script "open -a " & quoted form of "/Applications/Hancom Office HWP Viewer.app" & " " & quoted form of inputPath
	delay 4

	tell application "System Events"
		tell process "Hancom Office HWP Viewer"
			set frontmost to true
			set targetWindow to missing value
			repeat 40 times
				repeat with w in windows
					if name of w is docName then
						set targetWindow to w
						exit repeat
					end if
				end repeat
				if targetWindow is not missing value then exit repeat
				delay 0.5
			end repeat
			if targetWindow is missing value then error "viewer window not found: " & my describeOpenWindows()

			click menu item "인쇄..." of menu "파일" of menu bar item "파일" of menu bar 1
			repeat 40 times
				if exists sheet 1 of targetWindow then exit repeat
				delay 0.5
			end repeat
			if not (exists sheet 1 of targetWindow) then error "print dialog did not open"

			set printSheet to sheet 1 of targetWindow
			set pdfButton to menu button 1 of group 2 of splitter group 1 of printSheet
			click pdfButton
			delay 0.4
			click menu item "PDF로 저장…" of menu 1 of pdfButton

			repeat 40 times
				if exists sheet 1 of printSheet then exit repeat
				delay 0.5
			end repeat
			if not (exists sheet 1 of printSheet) then error "pdf save dialog did not open"

			set saveSheet to sheet 1 of printSheet
			set saveGroup to splitter group 1 of saveSheet

			set locationButton to pop up button "위치:" of saveGroup
			set selectedLocation to false
			if workspaceName is not "" then
				set selectedLocation to my clickMenuItemIfExists(locationButton, workspaceName)
			end if
			if (not selectedLocation) and sourceDirName is not "" then
				set selectedLocation to my clickMenuItemIfExists(locationButton, sourceDirName)
			end if
			delay 0.8

			click text field "별도 저장:" of saveGroup
			delay 0.2
			keystroke "a" using {command down}
			delay 0.2
			keystroke stageBase
			delay 0.4
			click button "저장" of saveGroup

			repeat 20 times
				if exists window "" then
					try
						click button "확인" of window ""
					end try
					error "print dialog reported an error"
				end if
				if not (exists saveSheet) then exit repeat
				delay 0.5
			end repeat
		end tell
	end tell
end run
`

	cmd := exec.Command("osascript", "-", inputPath, docName, workspaceName, sourceDir, stageBase)
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, trimmed)
	}
	return nil
}

func waitForPrintedPDF(fileName string, candidateDirs ...string) (string, error) {
	dirs := uniqueDirs(candidateDirs)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		for _, dir := range dirs {
			if dir == "" {
				continue
			}
			path := filepath.Join(dir, fileName)
			info, err := os.Stat(path)
			if err == nil && info.Size() > 0 {
				return path, nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", fmt.Errorf("printed pdf was not created")
}

func uniqueDirs(values []string) []string {
	seen := map[string]struct{}{}
	dirs := make([]string, 0, len(values)+3)
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		dirs = append(dirs, value)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		for _, extra := range []string{
			filepath.Join(home, "Documents"),
			filepath.Join(home, "Desktop"),
			filepath.Join(home, "Downloads"),
		} {
			if _, ok := seen[extra]; ok {
				continue
			}
			seen[extra] = struct{}{}
			dirs = append(dirs, extra)
		}
	}

	sort.Strings(dirs)
	return dirs
}

const defaultVersionXML = `<?xml version="1.0" encoding="UTF-8"?>
<hv:version xmlns:hv="http://www.hancom.co.kr/hwpml/2011/version" appVersion="11.0.0.0" hwpxVersion="1.0" />
`

const defaultContainerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="Contents/content.hpf" media-type="application/oebps-package+xml" />
  </rootfiles>
</container>
`

const defaultSettingsXML = `<?xml version="1.0" encoding="UTF-8"?>
<ha:settings xmlns:ha="http://www.hancom.co.kr/hwpml/2011/app">
  <ha:CaretPosition listIDRef="0" paraIDRef="0" pos="0" />
</ha:settings>
`

const defaultContentXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<opf:package
  xmlns:opf="http://www.idpf.org/2007/opf"
  xmlns:hh="http://www.hancom.co.kr/hwpml/2011/head"
  xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"
  xmlns:hs="http://www.hancom.co.kr/hwpml/2011/section"
  version=""
  unique-identifier=""
  id=""
>
  <opf:metadata>
    <opf:title />
    <opf:language>ko</opf:language>
  </opf:metadata>
  <opf:manifest>
    <opf:item id="header" href="Contents/header.xml" media-type="application/xml" />
    <opf:item id="section0" href="Contents/section0.xml" media-type="application/xml" />
    <opf:item id="settings" href="settings.xml" media-type="application/xml" />
  </opf:manifest>
  <opf:spine>
    <opf:itemref idref="header" linear="yes" />
    <opf:itemref idref="section0" linear="yes" />
  </opf:spine>
</opf:package>
`

const defaultHeaderXML = `<?xml version="1.0" encoding="UTF-8"?>
<hh:head
  xmlns:hc="http://www.hancom.co.kr/hwpml/2011/core"
  xmlns:hh="http://www.hancom.co.kr/hwpml/2011/head"
  xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"
  version="1.5"
  secCnt="1"
>
  <hh:beginNum page="1" footnote="1" endnote="1" pic="1" tbl="1" equation="1" />
  <hh:refList>
    <hh:borderFills itemCnt="3">
      <hh:borderFill id="1" threeD="0" shadow="0" centerLine="NONE" breakCellSeparateLine="0">
        <hh:slash type="NONE" Crooked="0" isCounter="0" />
        <hh:backSlash type="NONE" Crooked="0" isCounter="0" />
        <hh:leftBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:rightBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:topBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:bottomBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:diagonal type="SOLID" width="0.1 mm" color="#000000" />
      </hh:borderFill>
      <hh:borderFill id="2" threeD="0" shadow="0" centerLine="NONE" breakCellSeparateLine="0">
        <hh:slash type="NONE" Crooked="0" isCounter="0" />
        <hh:backSlash type="NONE" Crooked="0" isCounter="0" />
        <hh:leftBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:rightBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:topBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:bottomBorder type="NONE" width="0.1 mm" color="#000000" />
        <hh:diagonal type="SOLID" width="0.1 mm" color="#000000" />
        <hc:fillBrush>
          <hc:winBrush faceColor="none" hatchColor="#999999" alpha="0" />
        </hc:fillBrush>
      </hh:borderFill>
      <hh:borderFill id="3" threeD="0" shadow="0" centerLine="NONE" breakCellSeparateLine="0">
        <hh:slash type="NONE" Crooked="0" isCounter="0" />
        <hh:backSlash type="NONE" Crooked="0" isCounter="0" />
        <hh:leftBorder type="SOLID" width="0.12 mm" color="#000000" />
        <hh:rightBorder type="SOLID" width="0.12 mm" color="#000000" />
        <hh:topBorder type="SOLID" width="0.12 mm" color="#000000" />
        <hh:bottomBorder type="SOLID" width="0.12 mm" color="#000000" />
        <hh:diagonal type="SOLID" width="0.1 mm" color="#000000" />
      </hh:borderFill>
    </hh:borderFills>
    <hh:binDataList itemCnt="0" />
  </hh:refList>
</hh:head>
`

const defaultSectionXML = `<?xml version="1.0" encoding="UTF-8"?>
<hs:sec
  xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"
  xmlns:hs="http://www.hancom.co.kr/hwpml/2011/section"
>
  <hp:p id="1" paraPrIDRef="0" styleIDRef="0" pageBreak="0" columnBreak="0" merged="0">
    <hp:run charPrIDRef="0">
      <hp:secPr
        id=""
        textDirection="HORIZONTAL"
        spaceColumns="1134"
        tabStop="8000"
        tabStopVal="4000"
        tabStopUnit="HWPUNIT"
        outlineShapeIDRef="1"
        memoShapeIDRef="0"
        textVerticalWidthHead="0"
        masterPageCnt="0"
      >
        <hp:grid lineGrid="0" charGrid="0" wonggojiFormat="0" />
        <hp:startNum pageStartsOn="BOTH" page="0" pic="0" tbl="0" equation="0" />
        <hp:visibility
          hideFirstHeader="0"
          hideFirstFooter="0"
          hideFirstMasterPage="0"
          border="SHOW_ALL"
          fill="SHOW_ALL"
          hideFirstPageNum="0"
          hideFirstEmptyLine="0"
          showLineNumber="0"
        />
        <hp:lineNumberShape restartType="0" countBy="0" distance="0" startNumber="0" />
        <hp:pagePr landscape="WIDELY" width="59528" height="84186" gutterType="LEFT_ONLY">
          <hp:margin header="4252" footer="4252" gutter="0" left="8504" right="8504" top="5668" bottom="4252" />
        </hp:pagePr>
        <hp:footNotePr>
          <hp:autoNumFormat type="DIGIT" userChar="" prefixChar="" suffixChar=")" supscript="0" />
          <hp:noteLine length="-1" type="SOLID" width="0.12 mm" color="#000000" />
          <hp:noteSpacing betweenNotes="283" belowLine="567" aboveLine="850" />
          <hp:numbering type="CONTINUOUS" newNum="1" />
          <hp:placement place="EACH_COLUMN" beneathText="0" />
        </hp:footNotePr>
        <hp:endNotePr>
          <hp:autoNumFormat type="DIGIT" userChar="" prefixChar="" suffixChar=")" supscript="0" />
          <hp:noteLine length="14692344" type="SOLID" width="0.12 mm" color="#000000" />
          <hp:noteSpacing betweenNotes="0" belowLine="567" aboveLine="850" />
          <hp:numbering type="CONTINUOUS" newNum="1" />
          <hp:placement place="END_OF_DOCUMENT" beneathText="0" />
        </hp:endNotePr>
        <hp:pageBorderFill type="BOTH" borderFillIDRef="1" textBorder="PAPER" headerInside="0" footerInside="0" fillArea="PAPER">
          <hp:offset left="1417" right="1417" top="1417" bottom="1417" />
        </hp:pageBorderFill>
        <hp:pageBorderFill type="EVEN" borderFillIDRef="1" textBorder="PAPER" headerInside="0" footerInside="0" fillArea="PAPER">
          <hp:offset left="1417" right="1417" top="1417" bottom="1417" />
        </hp:pageBorderFill>
        <hp:pageBorderFill type="ODD" borderFillIDRef="1" textBorder="PAPER" headerInside="0" footerInside="0" fillArea="PAPER">
          <hp:offset left="1417" right="1417" top="1417" bottom="1417" />
        </hp:pageBorderFill>
      </hp:secPr>
      <hp:ctrl>
        <hp:colPr id="" type="NEWSPAPER" layout="LEFT" colCount="1" sameSz="1" sameGap="0" />
      </hp:ctrl>
    </hp:run>
    <hp:run charPrIDRef="0">
      <hp:t />
    </hp:run>
  </hp:p>
</hs:sec>
`

func copyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Close()
}
