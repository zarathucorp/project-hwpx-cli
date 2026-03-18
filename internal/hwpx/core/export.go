package core

import (
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"

	"github.com/beevik/etree"
)

type exportBlock struct {
	Kind  string
	Level int
	Text  string
	Rows  [][]string
}

type exportStyleRef struct {
	ID   string
	Name string
}

func ExportMarkdown(targetPath string) (string, int, error) {
	blocks, err := extractExportBlocks(targetPath)
	if err != nil {
		return "", 0, err
	}
	return renderMarkdown(blocks), len(blocks), nil
}

func ExportHTML(targetPath string) (string, int, error) {
	blocks, err := extractExportBlocks(targetPath)
	if err != nil {
		return "", 0, err
	}
	return renderHTML(blocks), len(blocks), nil
}

func extractExportBlocks(targetPath string) ([]exportBlock, error) {
	entries, err := readEntriesFromPath(targetPath)
	if err != nil {
		return nil, err
	}

	report, err := inspectEntries(entries)
	if err != nil {
		return nil, err
	}
	if !report.Valid {
		return nil, fmt.Errorf("cannot export invalid HWPX package: %s", strings.Join(report.Errors, "; "))
	}

	styleByID, err := parseStyles(entries["Contents/header.xml"])
	if err != nil {
		return nil, fmt.Errorf("parse styles: %w", err)
	}

	var blocks []exportBlock
	for _, sectionPath := range report.Summary.SectionPath {
		sectionBlocks, err := extractSectionBlocks(entries[sectionPath], styleByID)
		if err != nil {
			return nil, fmt.Errorf("extract export blocks %s: %w", sectionPath, err)
		}
		blocks = append(blocks, sectionBlocks...)
	}
	return blocks, nil
}

func readEntriesFromPath(targetPath string) (map[string][]byte, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return readEntriesFromDir(targetPath)
	}
	return readEntriesFromArchive(targetPath)
}

func parseStyles(data []byte) (map[string]exportStyleRef, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return nil, err
	}

	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("header.xml has no root")
	}

	styles := map[string]exportStyleRef{}
	for _, element := range findElementsByTag(root, "hh:style") {
		id := strings.TrimSpace(element.SelectAttrValue("id", ""))
		name := strings.TrimSpace(element.SelectAttrValue("name", ""))
		if id == "" || name == "" {
			continue
		}
		styles[id] = exportStyleRef{
			ID:   id,
			Name: name,
		}
	}
	return styles, nil
}

func extractSectionBlocks(data []byte, styleByID map[string]exportStyleRef) ([]exportBlock, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return nil, err
	}

	root := doc.Root()
	if root == nil {
		return nil, fmt.Errorf("section xml has no root")
	}

	var blocks []exportBlock
	for _, paragraph := range childElementsByTag(root, "hp:p") {
		if paragraphHasSectionProperty(paragraph) {
			continue
		}

		text := strings.TrimSpace(paragraphDirectText(paragraph))
		if text != "" {
			level := headingLevelForExportStyle(styleByID[strings.TrimSpace(paragraph.SelectAttrValue("styleIDRef", ""))].Name)
			kind := "paragraph"
			if level > 0 {
				kind = "heading"
			}
			blocks = append(blocks, exportBlock{
				Kind:  kind,
				Level: level,
				Text:  text,
			})
		}

		blocks = append(blocks, paragraphObjectBlocks(paragraph)...)
	}

	return blocks, nil
}

func paragraphHasSectionProperty(paragraph *etree.Element) bool {
	for _, run := range childElementsByTag(paragraph, "hp:run") {
		if firstChildByTag(run, "hp:secPr") != nil {
			return true
		}
	}
	return false
}

func paragraphDirectText(paragraph *etree.Element) string {
	var builder strings.Builder
	for _, run := range childElementsByTag(paragraph, "hp:run") {
		appendInlineText(&builder, run, true)
	}
	return normalizeInlineText(builder.String())
}

func paragraphObjectBlocks(paragraph *etree.Element) []exportBlock {
	var blocks []exportBlock
	for _, run := range childElementsByTag(paragraph, "hp:run") {
		for _, child := range run.ChildElements() {
			blocks = append(blocks, objectBlocksFromElement(child)...)
		}
	}
	return blocks
}

func objectBlocksFromElement(element *etree.Element) []exportBlock {
	switch {
	case tagMatches(element.Tag, "hp:tbl"):
		return []exportBlock{{
			Kind: "table",
			Rows: tableRowsForExport(element),
		}}
	case tagMatches(element.Tag, "hp:pic"):
		return []exportBlock{{
			Kind: "paragraph",
			Text: imagePlaceholderText(element),
		}}
	case tagMatches(element.Tag, "hp:equation"):
		return []exportBlock{{
			Kind: "paragraph",
			Text: equationPlaceholderText(element),
		}}
	case tagMatches(element.Tag, "hp:rect"):
		if drawText := firstChildByTag(element, "hp:drawText"); drawText != nil {
			text := strings.TrimSpace(drawTextPlainText(drawText))
			if text != "" {
				return []exportBlock{{
					Kind: "paragraph",
					Text: text,
				}}
			}
		}
		return []exportBlock{{
			Kind: "paragraph",
			Text: "[Rectangle]",
		}}
	case tagMatches(element.Tag, "hp:line"):
		return []exportBlock{{
			Kind: "paragraph",
			Text: "[Line]",
		}}
	case tagMatches(element.Tag, "hp:ellipse"):
		return []exportBlock{{
			Kind: "paragraph",
			Text: "[Ellipse]",
		}}
	default:
		return nil
	}
}

func drawTextPlainText(drawText *etree.Element) string {
	var paragraphs []string
	for _, subList := range childElementsByTag(drawText, "hp:subList") {
		for _, paragraph := range childElementsByTag(subList, "hp:p") {
			text := strings.TrimSpace(paragraphDirectText(paragraph))
			if text != "" {
				paragraphs = append(paragraphs, text)
			}
		}
	}
	return strings.Join(paragraphs, "\n")
}

func appendInlineText(builder *strings.Builder, element *etree.Element, skipObjects bool) {
	for _, child := range element.Child {
		switch current := child.(type) {
		case *etree.CharData:
			builder.WriteString(current.Data)
		case *etree.Element:
			if skipObjects && isExportObjectElement(current) {
				continue
			}
			switch localName(current.Tag) {
			case "lineBreak":
				builder.WriteByte('\n')
			case "tab":
				builder.WriteByte('\t')
			default:
				appendInlineText(builder, current, skipObjects)
			}
		}
	}
}

func isExportObjectElement(element *etree.Element) bool {
	switch {
	case tagMatches(element.Tag, "hp:tbl"):
		return true
	case tagMatches(element.Tag, "hp:pic"):
		return true
	case tagMatches(element.Tag, "hp:equation"):
		return true
	case tagMatches(element.Tag, "hp:line"):
		return true
	case tagMatches(element.Tag, "hp:ellipse"):
		return true
	case tagMatches(element.Tag, "hp:rect"):
		return true
	default:
		return false
	}
}

func tableRowsForExport(table *etree.Element) [][]string {
	var rows [][]string
	for _, row := range childElementsByTag(table, "hp:tr") {
		var values []string
		for _, cell := range childElementsByTag(row, "hp:tc") {
			values = append(values, strings.TrimSpace(cellPlainTextForExport(cell)))
		}
		if len(values) > 0 {
			rows = append(rows, values)
		}
	}
	return rows
}

func cellPlainTextForExport(cell *etree.Element) string {
	var builder strings.Builder
	appendInlineText(&builder, cell, false)
	return normalizeInlineText(builder.String())
}

func imagePlaceholderText(element *etree.Element) string {
	for _, key := range []string{"binItemIDRef", "binaryItemIDRef", "itemIDRef"} {
		if value := strings.TrimSpace(element.SelectAttrValue(key, "")); value != "" {
			return fmt.Sprintf("[Image: %s]", value)
		}
	}
	return "[Image]"
}

func equationPlaceholderText(element *etree.Element) string {
	text := strings.TrimSpace(cellPlainTextForExport(element))
	if text == "" {
		return "[Equation]"
	}
	return fmt.Sprintf("[Equation: %s]", text)
}

func headingLevelForExportStyle(styleName string) int {
	name := strings.ToLower(strings.TrimSpace(styleName))
	switch {
	case name == "title":
		return 1
	case strings.HasPrefix(name, "heading "):
		level, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(name, "heading ")))
		if err == nil && level > 0 {
			if level > 6 {
				return 6
			}
			return level
		}
	case strings.HasPrefix(styleName, "개요 "):
		level, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(styleName, "개요 ")))
		if err == nil && level > 0 {
			if level > 6 {
				return 6
			}
			return level
		}
	}
	return 0
}

func renderMarkdown(blocks []exportBlock) string {
	var builder strings.Builder
	for index, block := range blocks {
		if index > 0 {
			builder.WriteString("\n\n")
		}
		switch block.Kind {
		case "heading":
			level := block.Level
			if level <= 0 {
				level = 1
			}
			builder.WriteString(strings.Repeat("#", level))
			builder.WriteByte(' ')
			builder.WriteString(block.Text)
		case "table":
			builder.WriteString(renderMarkdownTable(block.Rows))
		default:
			builder.WriteString(block.Text)
		}
	}
	return strings.TrimSpace(builder.String())
}

func renderMarkdownTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	columnCount := 0
	for _, row := range rows {
		if len(row) > columnCount {
			columnCount = len(row)
		}
	}
	if columnCount == 0 {
		return ""
	}

	normalized := make([][]string, 0, len(rows))
	for _, row := range rows {
		values := make([]string, columnCount)
		for index := 0; index < columnCount; index++ {
			if index < len(row) {
				values[index] = escapeMarkdownTableCell(row[index])
			}
		}
		normalized = append(normalized, values)
	}

	var builder strings.Builder
	writeMarkdownTableRow(&builder, normalized[0])
	builder.WriteByte('\n')
	writeMarkdownTableSeparator(&builder, columnCount)
	for _, row := range normalized[1:] {
		builder.WriteByte('\n')
		writeMarkdownTableRow(&builder, row)
	}
	return builder.String()
}

func writeMarkdownTableRow(builder *strings.Builder, row []string) {
	builder.WriteString("| ")
	builder.WriteString(strings.Join(row, " | "))
	builder.WriteString(" |")
}

func writeMarkdownTableSeparator(builder *strings.Builder, columnCount int) {
	for index := 0; index < columnCount; index++ {
		if index == 0 {
			builder.WriteString("| --- ")
			continue
		}
		builder.WriteString("| --- ")
	}
	builder.WriteString("|")
}

func escapeMarkdownTableCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.ReplaceAll(value, "\n", "<br>")
}

func renderHTML(blocks []exportBlock) string {
	var builder strings.Builder
	builder.WriteString("<!DOCTYPE html>\n<html lang=\"ko\">\n<head>\n")
	builder.WriteString("  <meta charset=\"UTF-8\" />\n")
	builder.WriteString("  <title>HWPX Export</title>\n")
	builder.WriteString("  <style>body{font-family:Apple SD Gothic Neo,Malgun Gothic,sans-serif;line-height:1.6;max-width:960px;margin:40px auto;padding:0 24px;}table{border-collapse:collapse;width:100%;margin:16px 0;}th,td{border:1px solid #d0d7de;padding:8px;vertical-align:top;}p{margin:12px 0;}h1,h2,h3,h4,h5,h6{margin:24px 0 12px;}</style>\n")
	builder.WriteString("</head>\n<body>\n")
	for _, block := range blocks {
		switch block.Kind {
		case "heading":
			level := block.Level
			if level <= 0 {
				level = 1
			}
			builder.WriteString(fmt.Sprintf("<h%d>%s</h%d>\n", level, html.EscapeString(block.Text), level))
		case "table":
			builder.WriteString(renderHTMLTable(block.Rows))
			builder.WriteByte('\n')
		default:
			builder.WriteString("<p>")
			builder.WriteString(strings.ReplaceAll(html.EscapeString(block.Text), "\n", "<br />"))
			builder.WriteString("</p>\n")
		}
	}
	builder.WriteString("</body>\n</html>\n")
	return builder.String()
}

func renderHTMLTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}

	columnCount := 0
	for _, row := range rows {
		if len(row) > columnCount {
			columnCount = len(row)
		}
	}

	var builder strings.Builder
	builder.WriteString("<table>\n")
	for rowIndex, row := range rows {
		builder.WriteString("  <tr>")
		tag := "td"
		if rowIndex == 0 {
			tag = "th"
		}
		for columnIndex := 0; columnIndex < columnCount; columnIndex++ {
			cell := ""
			if columnIndex < len(row) {
				cell = row[columnIndex]
			}
			builder.WriteString("<")
			builder.WriteString(tag)
			builder.WriteString(">")
			builder.WriteString(strings.ReplaceAll(html.EscapeString(cell), "\n", "<br />"))
			builder.WriteString("</")
			builder.WriteString(tag)
			builder.WriteString(">")
		}
		builder.WriteString("</tr>\n")
	}
	builder.WriteString("</table>")
	return builder.String()
}

func normalizeInlineText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	lines := strings.Split(value, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimSpace(line)
	}

	var normalized []string
	for _, line := range lines {
		if len(normalized) > 0 && line == "" && normalized[len(normalized)-1] == "" {
			continue
		}
		normalized = append(normalized, line)
	}

	return strings.TrimSpace(strings.Join(normalized, "\n"))
}

func localName(tag string) string {
	parts := strings.Split(tag, ":")
	return parts[len(parts)-1]
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
	return localName(actual) == localName(expected)
}
