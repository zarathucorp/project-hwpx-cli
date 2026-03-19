package core

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var requiredEntries = []string{
	"mimetype",
	"version.xml",
	"Contents/content.hpf",
	"Contents/header.xml",
}

type contentPackage struct {
	XMLName  xml.Name `xml:"package"`
	Metadata metadata `xml:"metadata"`
	Manifest manifest `xml:"manifest"`
	Spine    spine    `xml:"spine"`
}

type metadata struct {
	Title       string `xml:"title"`
	Creator     string `xml:"creator"`
	Subject     string `xml:"subject"`
	Description string `xml:"description"`
	Language    string `xml:"language"`
}

type manifest struct {
	Items []manifestItem `xml:"item"`
}

type manifestItem struct {
	ID        string `xml:"id,attr"`
	Href      string `xml:"href,attr"`
	MediaType string `xml:"media-type,attr"`
}

type spine struct {
	ItemRefs []itemRef `xml:"itemref"`
}

type itemRef struct {
	IDRef string `xml:"idref,attr"`
}

type head struct {
	XMLName xml.Name `xml:"head"`
	SecCnt  int      `xml:"secCnt,attr"`
}

func Inspect(filePath string) (Report, error) {
	entries, err := readEntriesFromArchive(filePath)
	if err != nil {
		return Report{}, err
	}

	return inspectEntries(entries)
}

func Validate(targetPath string) (Report, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return Report{}, err
	}

	if info.IsDir() {
		entries, readErr := readEntriesFromDir(targetPath)
		if readErr != nil {
			return Report{}, readErr
		}
		return inspectEntries(entries)
	}

	return Inspect(targetPath)
}

func ExtractText(filePath string) (string, error) {
	entries, err := readEntriesFromArchive(filePath)
	if err != nil {
		return "", err
	}

	report, err := inspectEntries(entries)
	if err != nil {
		return "", err
	}
	if !report.Valid {
		return "", fmt.Errorf("cannot extract text from invalid HWPX package: %s", strings.Join(report.Errors, "; "))
	}

	var paragraphs []string
	for _, sectionPath := range report.Summary.SectionPath {
		data := entries[sectionPath]
		texts, extractErr := extractParagraphs(data)
		if extractErr != nil {
			return "", fmt.Errorf("extract section text %s: %w", sectionPath, extractErr)
		}
		paragraphs = append(paragraphs, texts...)
	}

	return strings.Join(paragraphs, "\n"), nil
}

func Unpack(filePath, outputDir string) error {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	for _, archiveFile := range reader.File {
		destination := filepath.Join(outputDir, filepath.FromSlash(archiveFile.Name))
		if archiveFile.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}

		src, openErr := archiveFile.Open()
		if openErr != nil {
			return openErr
		}

		content, readErr := io.ReadAll(src)
		closeErr := src.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}

		if err := os.WriteFile(destination, content, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func Pack(inputDir, outputFile string) error {
	report, err := Validate(inputDir)
	if err != nil {
		return err
	}
	if !report.Valid {
		return fmt.Errorf("cannot pack invalid HWPX directory: %s", strings.Join(report.Errors, "; "))
	}

	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return err
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	if err := addStoredFile(writer, inputDir, "mimetype"); err != nil {
		return err
	}

	err = filepath.WalkDir(inputDir, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		relative, relErr := filepath.Rel(inputDir, current)
		if relErr != nil {
			return relErr
		}
		archivePath := filepath.ToSlash(relative)
		if archivePath == "mimetype" || isInternalWorkingEntry(archivePath) {
			return nil
		}

		return addDeflatedFile(writer, current, archivePath)
	})
	if err != nil {
		return err
	}

	return writer.Close()
}

func inspectEntries(entries map[string][]byte) (Report, error) {
	report := Report{
		Valid:       true,
		RenderSafe:  true,
		Errors:      []string{},
		Warnings:    []string{},
		RiskHints:   []string{},
		RiskSignals: map[string]int{},
		Summary: Summary{
			Entries: []string{},
		},
	}

	for path := range entries {
		report.Summary.Entries = append(report.Summary.Entries, path)
	}
	slices.Sort(report.Summary.Entries)

	for _, required := range requiredEntries {
		if _, ok := entries[required]; !ok {
			report.Errors = append(report.Errors, fmt.Sprintf("missing required entry: %s", required))
		}
	}
	if len(report.Errors) > 0 {
		report.Valid = false
		return report, nil
	}

	contentDoc, err := decodeContent(entries["Contents/content.hpf"])
	if err != nil {
		return Report{}, fmt.Errorf("parse content.hpf: %w", err)
	}
	version, err := parseVersion(entries["version.xml"])
	if err != nil {
		return Report{}, fmt.Errorf("parse version.xml: %w", err)
	}
	secCnt, err := parseHeadSecCount(entries["Contents/header.xml"])
	if err != nil {
		return Report{}, fmt.Errorf("parse header.xml: %w", err)
	}

	manifestItems := make([]ManifestItem, 0, len(contentDoc.Manifest.Items))
	manifestByID := make(map[string]manifestItem, len(contentDoc.Manifest.Items))
	binaryPaths := make([]string, 0)

	for _, item := range contentDoc.Manifest.Items {
		manifestItems = append(manifestItems, ManifestItem{
			ID:        item.ID,
			Href:      item.Href,
			MediaType: item.MediaType,
		})
		manifestByID[item.ID] = item

		resolved := resolveEntryPath(item.Href, entries)
		if strings.HasPrefix(resolved, "BinData/") {
			binaryPaths = append(binaryPaths, resolved)
		}

		if _, ok := entries[resolved]; !ok {
			report.Warnings = append(report.Warnings, fmt.Sprintf("manifest item not found on disk: %s", item.Href))
		}
	}

	sectionPaths := make([]string, 0, len(contentDoc.Spine.ItemRefs))
	spineIDs := make([]string, 0, len(contentDoc.Spine.ItemRefs))
	for _, ref := range contentDoc.Spine.ItemRefs {
		spineIDs = append(spineIDs, ref.IDRef)

		item, ok := manifestByID[ref.IDRef]
		if !ok {
			report.Errors = append(report.Errors, fmt.Sprintf("spine references unknown manifest item: %s", ref.IDRef))
			continue
		}

		resolved := resolveEntryPath(item.Href, entries)
		if _, ok := entries[resolved]; !ok {
			report.Errors = append(report.Errors, fmt.Sprintf("missing spine entry: %s", resolved))
			continue
		}

		if isSectionPath(resolved) {
			sectionPaths = append(sectionPaths, resolved)
		}
	}

	if secCnt > 0 && secCnt != len(sectionPaths) {
		report.Warnings = append(report.Warnings, fmt.Sprintf("header.xml secCnt=%d, spine sections=%d", secCnt, len(sectionPaths)))
	}

	report.Valid = len(report.Errors) == 0
	report.Summary.Metadata = contentDoc.Metadata.toMap()
	report.Summary.Version = version
	report.Summary.Manifest = manifestItems
	report.Summary.Spine = spineIDs
	report.Summary.SectionPath = sectionPaths
	report.Summary.BinaryPath = binaryPaths
	report.RiskHints, report.RiskSignals = analyzeRiskHints(entries, sectionPaths)
	report.RenderSafe = report.Valid && len(report.RiskHints) == 0
	return report, nil
}

func analyzeRiskHints(entries map[string][]byte, sectionPaths []string) ([]string, map[string]int) {
	riskHints := []string{}
	riskSignals := map[string]int{}

	if len(sectionPaths) > 1 {
		riskHints = append(riskHints, "section-risk")
		riskSignals["sectionCount"] = len(sectionPaths)
	}

	tableCount := 0
	mergedCellCount := 0
	nestedTableCount := 0
	objectCount := 0
	tocMarkerCount := 0
	headerFooterCount := 0
	pageNumberCount := 0

	for _, sectionPath := range sectionPaths {
		content := string(entries[sectionPath])
		tableCount += strings.Count(content, "<hp:tbl")
		headerFooterCount += strings.Count(content, "<hp:header")
		headerFooterCount += strings.Count(content, "<hp:footer")
		pageNumberCount += strings.Count(content, "<hp:pageNum")

		rowSpanMatches := countSpanMarkers(content, "rowSpan")
		colSpanMatches := countSpanMarkers(content, "colSpan")
		mergedCellCount += rowSpanMatches + colSpanMatches

		sectionTableCount := strings.Count(content, "<hp:tbl")
		sectionParagraphCount := strings.Count(content, "<hp:p")
		if sectionTableCount > 1 && sectionParagraphCount > 0 {
			nestedTableCount += strings.Count(content, "<hp:tbl") - 1
		}

		objectCount += strings.Count(content, "<hp:pic")
		objectCount += strings.Count(content, "<hp:rect")
		objectCount += strings.Count(content, "<hp:ellipse")
		objectCount += strings.Count(content, "<hp:line")
		objectCount += strings.Count(content, "<hp:equation")
		objectCount += strings.Count(content, "<hp:container")
		objectCount += strings.Count(content, "<hp:drawText")
	}

	headerContent := string(entries["Contents/header.xml"])
	tocMarkerCount += strings.Count(strings.ToLower(headerContent), "toc heading")
	tocMarkerCount += strings.Count(strings.ToLower(headerContent), "toc ")
	for _, sectionPath := range sectionPaths {
		content := string(entries[sectionPath])
		tocMarkerCount += strings.Count(content, "목차")
	}

	riskSignals["tableCount"] = tableCount
	riskSignals["mergedCellCount"] = mergedCellCount
	riskSignals["nestedTableCount"] = nestedTableCount
	riskSignals["objectCount"] = objectCount
	riskSignals["tocMarkerCount"] = tocMarkerCount
	riskSignals["headerFooterCount"] = headerFooterCount
	riskSignals["pageNumberCount"] = pageNumberCount

	if tocMarkerCount > 0 {
		riskHints = append(riskHints, "toc-risk")
	}
	if mergedCellCount > 0 || nestedTableCount > 0 || tableCount >= 20 {
		riskHints = append(riskHints, "table-risk")
	}
	if objectCount > 0 || headerFooterCount > 0 || pageNumberCount > 0 {
		riskHints = append(riskHints, "layout-risk")
	}

	return dedupeStrings(riskHints), pruneZeroRiskSignals(riskSignals)
}

func countSpanMarkers(content, attrName string) int {
	count := 0
	search := attrName + "=\""
	offset := 0
	for {
		index := strings.Index(content[offset:], search)
		if index < 0 {
			return count
		}
		start := offset + index + len(search)
		end := strings.Index(content[start:], "\"")
		if end < 0 {
			return count
		}
		value := content[start : start+end]
		if value != "" && value != "1" {
			count++
		}
		offset = start + end + 1
	}
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}

	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func pruneZeroRiskSignals(signals map[string]int) map[string]int {
	result := map[string]int{}
	for key, value := range signals {
		if value == 0 {
			continue
		}
		result[key] = value
	}
	return result
}

func (m metadata) toMap() map[string]string {
	values := map[string]string{}

	if m.Title != "" {
		values["title"] = m.Title
	}
	if m.Creator != "" {
		values["creator"] = m.Creator
	}
	if m.Subject != "" {
		values["subject"] = m.Subject
	}
	if m.Description != "" {
		values["description"] = m.Description
	}
	if m.Language != "" {
		values["language"] = m.Language
	}

	return values
}

func readEntriesFromArchive(filePath string) (map[string][]byte, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	entries := make(map[string][]byte, len(reader.File))
	for _, archiveFile := range reader.File {
		if archiveFile.FileInfo().IsDir() {
			continue
		}

		src, openErr := archiveFile.Open()
		if openErr != nil {
			return nil, openErr
		}

		content, readErr := io.ReadAll(src)
		closeErr := src.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}

		entries[archiveFile.Name] = content
	}

	return entries, nil
}

func readEntriesFromDir(root string) (map[string][]byte, error) {
	entries := map[string][]byte{}

	err := filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		relative, err := filepath.Rel(root, current)
		if err != nil {
			return err
		}
		archivePath := filepath.ToSlash(relative)
		if isInternalWorkingEntry(archivePath) {
			return nil
		}

		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}

		entries[archivePath] = content
		return nil
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func isInternalWorkingEntry(path string) bool {
	base := filepath.Base(path)
	if base == ".hwpxctl.lock" {
		return true
	}
	return strings.HasPrefix(base, ".hwpxctl-") && strings.HasSuffix(base, ".tmp")
}

func decodeContent(data []byte) (contentPackage, error) {
	var document contentPackage
	if err := xml.Unmarshal(data, &document); err != nil {
		return contentPackage{}, err
	}
	return document, nil
}

func parseVersion(data []byte) (map[string]string, error) {
	var root struct {
		XMLName     xml.Name
		AppVersion  string `xml:"appVersion,attr"`
		HWPXVersion string `xml:"hwpxVersion,attr"`
	}
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	return map[string]string{
		"appVersion":  root.AppVersion,
		"hwpxVersion": root.HWPXVersion,
	}, nil
}

func parseHeadSecCount(data []byte) (int, error) {
	var doc head
	if err := xml.Unmarshal(data, &doc); err != nil {
		return 0, err
	}
	return doc.SecCnt, nil
}

func extractParagraphs(data []byte) ([]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var (
		inText     bool
		paragraphs []string
		stack      []*paragraphState
	)

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		switch current := token.(type) {
		case xml.StartElement:
			switch current.Name.Local {
			case "p":
				stack = append(stack, &paragraphState{})
			case "t", "script":
				if len(stack) > 0 {
					inText = true
				}
			case "lineBreak":
				if len(stack) > 0 {
					stack[len(stack)-1].builder.WriteByte('\n')
				}
			case "tab":
				if len(stack) > 0 {
					stack[len(stack)-1].builder.WriteByte('\t')
				}
			}
		case xml.CharData:
			if len(stack) > 0 && inText {
				stack[len(stack)-1].builder.Write([]byte(current))
			}
		case xml.EndElement:
			switch current.Name.Local {
			case "t", "script":
				inText = false
			case "p":
				if len(stack) == 0 {
					continue
				}
				last := stack[len(stack)-1]
				stack = stack[:len(stack)-1]

				text := last.builder.String()
				if len(stack) > 0 {
					parent := stack[len(stack)-1]
					if text != "" {
						parent.children = append(parent.children, text)
					}
					if len(last.children) > 0 {
						parent.children = append(parent.children, last.children...)
					}
					continue
				}

				if text != "" {
					paragraphs = append(paragraphs, text)
				}
				if len(last.children) > 0 {
					paragraphs = append(paragraphs, last.children...)
				}
			}
		}
	}

	return paragraphs, nil
}

type paragraphState struct {
	builder  strings.Builder
	children []string
}

func resolveEntryPath(href string, entries map[string][]byte) string {
	normalized := strings.TrimLeft(filepath.ToSlash(href), "/")
	candidates := []string{normalized}

	if !strings.HasPrefix(normalized, "Contents/") && !strings.HasPrefix(normalized, "BinData/") {
		candidates = append(candidates, filepath.ToSlash(filepath.Join("Contents", normalized)))
	}

	for _, candidate := range candidates {
		if _, ok := entries[candidate]; ok {
			return candidate
		}
	}

	return candidates[0]
}

func isSectionPath(value string) bool {
	base := filepath.Base(value)
	return strings.HasPrefix(base, "section") && strings.HasSuffix(base, ".xml")
}

func addStoredFile(writer *zip.Writer, root, archivePath string) error {
	fullPath := filepath.Join(root, filepath.FromSlash(archivePath))
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}

	header := &zip.FileHeader{
		Name:   archivePath,
		Method: zip.Store,
	}
	dst, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = dst.Write(content)
	return err
}

func addDeflatedFile(writer *zip.Writer, fullPath, archivePath string) error {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}

	header := &zip.FileHeader{
		Name:   archivePath,
		Method: zip.Deflate,
	}
	dst, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = dst.Write(content)
	return err
}
