package hwpx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

const (
	defaultTableWidth  = 42520
	defaultCellHeight  = 2400
	defaultSectionPath = "Contents/section0.xml"
	templateGlob       = "example/*.hwpx"
)

type TableSpec struct {
	Rows  int
	Cols  int
	Cells [][]string
}

type ImageEmbed struct {
	ItemID     string
	BinaryPath string
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
	matches, err := filepath.Glob(templateGlob)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no template archive matched %s", templateGlob)
	}
	return matches[0], nil
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

	rows := childElementsByTag(tables[tableIndex], "hp:tr")
	if row >= len(rows) {
		return Report{}, fmt.Errorf("row index out of range: %d", row)
	}

	cells := childElementsByTag(rows[row], "hp:tc")
	if col >= len(cells) {
		return Report{}, fmt.Errorf("col index out of range: %d", col)
	}

	subList := firstChildByTag(cells[col], "hp:subList")
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

func resolvePrimarySectionPath(targetDir string) (string, error) {
	report, err := Validate(targetDir)
	if err != nil {
		return "", err
	}
	if len(report.Summary.SectionPath) > 0 {
		return report.Summary.SectionPath[0], nil
	}

	fallback := filepath.Join(targetDir, filepath.FromSlash(defaultSectionPath))
	if _, err := os.Stat(fallback); err == nil {
		return defaultSectionPath, nil
	}
	return "", fmt.Errorf("no editable section xml found")
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
	manifest.AddChild(item)
	return nil
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
		if !strings.HasPrefix(id, "BIN") {
			continue
		}
		value, err := strconv.Atoi(strings.TrimPrefix(id, "BIN"))
		if err == nil && value > maxValue {
			maxValue = value
		}
	}
	return fmt.Sprintf("BIN%04d", maxValue+1)
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
