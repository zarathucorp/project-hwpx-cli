package shared

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

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

func SetParagraphLayout(targetDir string, paragraphIndex int, spec ParagraphLayoutSpec) (Report, string, error) {
	if paragraphIndex < 0 {
		return Report{}, "", fmt.Errorf("paragraph index must be zero or greater")
	}
	if !paragraphLayoutHasChanges(spec) {
		return Report{}, "", fmt.Errorf("paragraph layout must include at least one change")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return Report{}, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	paragraphs := editableParagraphs(sectionRoot)
	if paragraphIndex >= len(paragraphs) {
		return Report{}, "", fmt.Errorf("paragraph index out of range: %d", paragraphIndex)
	}
	paragraph := paragraphs[paragraphIndex]
	previousParaPrID := strings.TrimSpace(paragraph.SelectAttrValue("paraPrIDRef", "0"))

	headerPath := filepath.Join(targetDir, "Contents", "header.xml")
	headerDoc, err := loadXML(headerPath)
	if err != nil {
		return Report{}, "", err
	}
	headerRoot := headerDoc.Root()
	if headerRoot == nil {
		return Report{}, "", fmt.Errorf("header xml has no root")
	}

	paraProperties := ensureParagraphProperties(headerRoot)
	styledID, err := ensureStyledParaPr(paraProperties, previousParaPrID, func(paraPr *etree.Element) error {
		applyParagraphLayoutToParaPr(paraPr, spec)
		return nil
	})
	if err != nil {
		return Report{}, "", err
	}

	setElementAttr(paragraph, "paraPrIDRef", styledID)

	if err := saveXML(headerDoc, headerPath); err != nil {
		return Report{}, "", err
	}
	if err := saveXML(sectionDoc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, styledID, nil
}

func SetParagraphList(targetDir string, paragraphIndex int, spec ParagraphListSpec) (Report, string, error) {
	if paragraphIndex < 0 {
		return Report{}, "", fmt.Errorf("paragraph index must be zero or greater")
	}

	kind := strings.ToUpper(strings.TrimSpace(spec.Kind))
	if kind == "" {
		return Report{}, "", fmt.Errorf("paragraph list kind must not be empty")
	}
	if kind != "BULLET" && kind != "NUMBER" && kind != "NONE" {
		return Report{}, "", fmt.Errorf("unsupported paragraph list kind: %s", spec.Kind)
	}
	if spec.Level < 0 {
		return Report{}, "", fmt.Errorf("paragraph list level must be zero or greater")
	}
	if kind == "BULLET" && spec.StartNumber != nil {
		return Report{}, "", fmt.Errorf("bullet list does not support start number")
	}
	if spec.StartNumber != nil && *spec.StartNumber <= 0 {
		return Report{}, "", fmt.Errorf("start number must be positive")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return Report{}, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	paragraphs := editableParagraphs(sectionRoot)
	if paragraphIndex >= len(paragraphs) {
		return Report{}, "", fmt.Errorf("paragraph index out of range: %d", paragraphIndex)
	}
	paragraph := paragraphs[paragraphIndex]
	previousParaPrID := strings.TrimSpace(paragraph.SelectAttrValue("paraPrIDRef", "0"))

	headerPath := filepath.Join(targetDir, "Contents", "header.xml")
	headerDoc, err := loadXML(headerPath)
	if err != nil {
		return Report{}, "", err
	}
	headerRoot := headerDoc.Root()
	if headerRoot == nil {
		return Report{}, "", fmt.Errorf("header xml has no root")
	}

	paraProperties := ensureParagraphProperties(headerRoot)
	refList := firstChildByTag(headerRoot, "hh:refList")
	if refList == nil {
		return Report{}, "", fmt.Errorf("header.xml is missing hh:refList")
	}

	styledID, err := ensureStyledParaPr(paraProperties, previousParaPrID, func(paraPr *etree.Element) error {
		return applyParagraphListToParaPr(refList, paraPr, spec)
	})
	if err != nil {
		return Report{}, "", err
	}

	setElementAttr(paragraph, "paraPrIDRef", styledID)

	if err := saveXML(headerDoc, headerPath); err != nil {
		return Report{}, "", err
	}
	if err := saveXML(sectionDoc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, styledID, nil
}

func AddRunText(targetDir string, paragraphIndex int, runIndex *int, text string) (Report, int, string, error) {
	if paragraphIndex < 0 {
		return Report{}, 0, "", fmt.Errorf("paragraph index must be zero or greater")
	}
	if strings.TrimSpace(text) == "" {
		return Report{}, 0, "", fmt.Errorf("run text must not be empty")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, 0, "", err
	}

	doc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, 0, "", err
	}

	root := doc.Root()
	if root == nil {
		return Report{}, 0, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	paragraphs := editableParagraphs(root)
	if paragraphIndex >= len(paragraphs) {
		return Report{}, 0, "", fmt.Errorf("paragraph index out of range: %d", paragraphIndex)
	}

	paragraph := paragraphs[paragraphIndex]
	runs := childElementsByTag(paragraph, "hp:run")
	insertIndex := len(runs)
	if runIndex != nil {
		if *runIndex < 0 || *runIndex > len(runs) {
			return Report{}, 0, "", fmt.Errorf("run index out of range: %d", *runIndex)
		}
		insertIndex = *runIndex
	}

	charPrIDRef := resolveInsertedRunCharPrIDRef(paragraph, runs, insertIndex)
	insertRunText(paragraph, insertIndex, charPrIDRef, text)
	refreshParagraphLineSeg(paragraph)

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, 0, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, 0, "", err
	}
	return report, insertIndex, charPrIDRef, nil
}

func SetRunText(targetDir string, paragraphIndex, runIndex int, text string) (Report, string, string, error) {
	if paragraphIndex < 0 {
		return Report{}, "", "", fmt.Errorf("paragraph index must be zero or greater")
	}
	if runIndex < 0 {
		return Report{}, "", "", fmt.Errorf("run index must be zero or greater")
	}
	if strings.TrimSpace(text) == "" {
		return Report{}, "", "", fmt.Errorf("run text must not be empty")
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

	paragraphs := editableParagraphs(root)
	if paragraphIndex >= len(paragraphs) {
		return Report{}, "", "", fmt.Errorf("paragraph index out of range: %d", paragraphIndex)
	}

	paragraph := paragraphs[paragraphIndex]
	runs, err := editableRunsForParagraph(paragraph, &runIndex)
	if err != nil {
		return Report{}, "", "", err
	}

	targetRun := runs[0]
	previousText := elementPlainText(targetRun)
	charPrIDRef := fallbackString(strings.TrimSpace(targetRun.SelectAttrValue("charPrIDRef", "")), "0")
	replaceRunText(targetRun, text)
	refreshParagraphLineSeg(paragraph)

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", "", err
	}
	return report, previousText, charPrIDRef, nil
}

func FindRunsByStyle(targetDir string, filter RunStyleFilter) ([]RunStyleMatch, error) {
	if !runStyleFilterHasConditions(filter) {
		return nil, fmt.Errorf("run style filter must include at least one condition")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return nil, err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return nil, err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return nil, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	headerDoc, err := loadXML(filepath.Join(targetDir, "Contents", "header.xml"))
	if err != nil {
		return nil, err
	}
	headerRoot := headerDoc.Root()
	if headerRoot == nil {
		return nil, fmt.Errorf("header xml has no root")
	}

	charProperties := firstChildByTag(firstChildByTag(headerRoot, "hh:refList"), "hh:charProperties")
	charPrStates := buildCharPrStateMap(charProperties)

	var matches []RunStyleMatch
	for paragraphIndex, paragraph := range editableParagraphs(sectionRoot) {
		for runIndex, run := range childElementsByTag(paragraph, "hp:run") {
			state := resolveRunStyleState(run, charPrStates)
			if !runStyleMatchesFilter(state, filter) {
				continue
			}
			matches = append(matches, RunStyleMatch{
				Paragraph:   paragraphIndex,
				Run:         runIndex,
				Text:        elementPlainText(run),
				CharPrIDRef: state.CharPrIDRef,
				Bold:        state.Bold,
				Italic:      state.Italic,
				Underline:   state.Underline,
				TextColor:   state.TextColor,
			})
		}
	}
	return matches, nil
}

func ReplaceRunsByStyle(targetDir string, filter RunStyleFilter, text string) (Report, []RunTextReplacement, error) {
	if !runStyleFilterHasConditions(filter) {
		return Report{}, nil, fmt.Errorf("run style filter must include at least one condition")
	}
	if strings.TrimSpace(text) == "" {
		return Report{}, nil, fmt.Errorf("replacement text must not be empty")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, nil, err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, nil, err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return Report{}, nil, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	headerDoc, err := loadXML(filepath.Join(targetDir, "Contents", "header.xml"))
	if err != nil {
		return Report{}, nil, err
	}
	headerRoot := headerDoc.Root()
	if headerRoot == nil {
		return Report{}, nil, fmt.Errorf("header xml has no root")
	}

	charProperties := firstChildByTag(firstChildByTag(headerRoot, "hh:refList"), "hh:charProperties")
	charPrStates := buildCharPrStateMap(charProperties)
	paragraphs := editableParagraphs(sectionRoot)
	modifiedParagraphs := make(map[int]struct{})
	replacements := make([]RunTextReplacement, 0)

	for paragraphIndex, paragraph := range paragraphs {
		for runIndex, run := range childElementsByTag(paragraph, "hp:run") {
			state := resolveRunStyleState(run, charPrStates)
			if !runStyleMatchesFilter(state, filter) {
				continue
			}

			replacements = append(replacements, RunTextReplacement{
				Paragraph:    paragraphIndex,
				Run:          runIndex,
				PreviousText: elementPlainText(run),
				Text:         text,
				CharPrIDRef:  state.CharPrIDRef,
			})
			replaceRunText(run, text)
			modifiedParagraphs[paragraphIndex] = struct{}{}
		}
	}

	for paragraphIndex := range modifiedParagraphs {
		refreshParagraphLineSeg(paragraphs[paragraphIndex])
	}

	if len(replacements) > 0 {
		if err := saveXML(sectionDoc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
			return Report{}, nil, err
		}
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, nil, err
	}
	return report, replacements, nil
}

func SetObjectPosition(targetDir string, spec ObjectPositionSpec) (Report, string, error) {
	objectType := strings.ToLower(strings.TrimSpace(spec.Type))
	if objectType == "" {
		return Report{}, "", fmt.Errorf("object type must not be empty")
	}
	if spec.Index < 0 {
		return Report{}, "", fmt.Errorf("object index must be zero or greater")
	}
	if spec.TreatAsChar == nil && spec.XMM == nil && spec.YMM == nil && strings.TrimSpace(spec.HorzAlign) == "" && strings.TrimSpace(spec.VertAlign) == "" {
		return Report{}, "", fmt.Errorf("object position must include at least one change")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, "", err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, "", err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return Report{}, "", fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	element := findObjectElementByTypeAndIndex(sectionRoot, objectType, spec.Index)
	if element == nil {
		return Report{}, "", fmt.Errorf("object not found: type=%s index=%d", objectType, spec.Index)
	}
	position := firstChildByTag(element, "hp:pos")
	if position == nil {
		return Report{}, "", fmt.Errorf("object position element is missing")
	}

	if spec.TreatAsChar != nil {
		if *spec.TreatAsChar {
			setElementAttr(position, "treatAsChar", "1")
			setElementAttr(position, "flowWithText", "1")
			setElementAttr(position, "allowOverlap", "0")
		} else {
			setElementAttr(position, "treatAsChar", "0")
			setElementAttr(position, "flowWithText", "0")
			setElementAttr(position, "allowOverlap", "1")
		}
	}
	if spec.XMM != nil {
		setElementAttr(position, "horzOffset", strconv.Itoa(mmToHWPUnit(*spec.XMM)))
	}
	if spec.YMM != nil {
		setElementAttr(position, "vertOffset", strconv.Itoa(mmToHWPUnit(*spec.YMM)))
	}
	if value := normalizePositionAlign(spec.HorzAlign, []string{"LEFT", "CENTER", "RIGHT"}); value != "" {
		setElementAttr(position, "horzAlign", value)
	}
	if value := normalizePositionAlign(spec.VertAlign, []string{"TOP", "CENTER", "BOTTOM"}); value != "" {
		setElementAttr(position, "vertAlign", value)
	}

	if err := saveXML(sectionDoc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", err
	}
	return report, objectElementID(element), nil
}

func FindObjects(targetDir string, filter ObjectFilter) ([]ObjectMatch, error) {
	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return nil, err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return nil, err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return nil, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	typeFilter := normalizeObjectTypeFilter(filter.Types)
	matches := make([]ObjectMatch, 0)
	index := 0

	for paragraphIndex, paragraph := range editableParagraphs(sectionRoot) {
		for runIndex, run := range childElementsByTag(paragraph, "hp:run") {
			basePath := fmt.Sprintf("hp:p[%d]/hp:run[%d]", paragraphIndex, runIndex)
			collectObjectMatches(run, paragraphIndex, runIndex, basePath, typeFilter, &index, &matches)
		}
	}

	return matches, nil
}

func FindByTag(targetDir string, filter TagFilter) ([]TagMatch, error) {
	tag := strings.TrimSpace(filter.Tag)
	if tag == "" {
		return nil, fmt.Errorf("tag filter must include a tag")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return nil, err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return nil, err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return nil, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	matches := make([]TagMatch, 0)
	index := 0
	for paragraphIndex, paragraph := range editableParagraphs(sectionRoot) {
		for runIndex, run := range childElementsByTag(paragraph, "hp:run") {
			basePath := fmt.Sprintf("hp:p[%d]/hp:run[%d]", paragraphIndex, runIndex)
			collectTagMatches(run, paragraphIndex, runIndex, basePath, tag, &index, &matches)
		}
	}
	return matches, nil
}

func FindByAttr(targetDir string, filter AttributeFilter) ([]AttributeMatch, error) {
	attr := strings.TrimSpace(filter.Attr)
	if attr == "" {
		return nil, fmt.Errorf("attribute filter must include an attribute name")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return nil, err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return nil, err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return nil, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	matches := make([]AttributeMatch, 0)
	index := 0
	for paragraphIndex, paragraph := range editableParagraphs(sectionRoot) {
		for runIndex, run := range childElementsByTag(paragraph, "hp:run") {
			basePath := fmt.Sprintf("hp:p[%d]/hp:run[%d]", paragraphIndex, runIndex)
			collectAttributeMatches(run, paragraphIndex, runIndex, basePath, filter, &index, &matches)
		}
	}
	return matches, nil
}

func FindByXPath(targetDir string, filter XPathFilter) ([]XPathMatch, error) {
	expr := strings.TrimSpace(filter.Expr)
	if expr == "" {
		return nil, fmt.Errorf("xpath filter must include an expression")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return nil, err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return nil, err
	}
	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return nil, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	path, err := etree.CompilePath(expr)
	if err != nil {
		return nil, err
	}

	contexts := make(map[*etree.Element]searchContext)
	paragraphIndexes := editableParagraphIndexMap(sectionRoot)
	collectSearchContexts(sectionRoot, searchContext{
		Paragraph: -1,
		Run:       -1,
		Path:      sectionRoot.Tag,
	}, paragraphIndexes, contexts)

	elements := sectionRoot.FindElementsPath(path)
	matches := make([]XPathMatch, 0, len(elements))
	for index, element := range elements {
		context, ok := contexts[element]
		if !ok {
			context = searchContext{Paragraph: -1, Run: -1, Path: element.Tag}
		}
		matches = append(matches, XPathMatch{
			Index:     index,
			Paragraph: context.Paragraph,
			Run:       context.Run,
			Path:      context.Path,
			Tag:       element.Tag,
			Text:      strings.TrimSpace(elementPlainText(element)),
		})
	}
	return matches, nil
}

func ApplyTextStyle(targetDir string, paragraphIndex int, runIndex *int, spec TextStyleSpec) (Report, []string, int, error) {
	if paragraphIndex < 0 {
		return Report{}, nil, 0, fmt.Errorf("paragraph index must be zero or greater")
	}
	if !textStyleHasChanges(spec) {
		return Report{}, nil, 0, fmt.Errorf("text style must include at least one change")
	}

	sectionPath, err := resolvePrimarySectionPath(targetDir)
	if err != nil {
		return Report{}, nil, 0, err
	}

	sectionDoc, err := loadXML(filepath.Join(targetDir, filepath.FromSlash(sectionPath)))
	if err != nil {
		return Report{}, nil, 0, err
	}

	sectionRoot := sectionDoc.Root()
	if sectionRoot == nil {
		return Report{}, nil, 0, fmt.Errorf("section xml has no root: %s", sectionPath)
	}

	paragraphs := editableParagraphs(sectionRoot)
	if paragraphIndex >= len(paragraphs) {
		return Report{}, nil, 0, fmt.Errorf("paragraph index out of range: %d", paragraphIndex)
	}

	targetParagraph := paragraphs[paragraphIndex]
	targetRuns, err := editableRunsForParagraph(targetParagraph, runIndex)
	if err != nil {
		return Report{}, nil, 0, err
	}

	headerPath := filepath.Join(targetDir, "Contents", "header.xml")
	headerDoc, err := loadXML(headerPath)
	if err != nil {
		return Report{}, nil, 0, err
	}

	headerRoot := headerDoc.Root()
	if headerRoot == nil {
		return Report{}, nil, 0, fmt.Errorf("header xml has no root")
	}

	charProperties := ensureCharProperties(headerRoot)
	cache := make(map[string]string)
	usedIDs := make(map[string]struct{})

	for _, run := range targetRuns {
		baseID := strings.TrimSpace(run.SelectAttrValue("charPrIDRef", "0"))
		if baseID == "" {
			baseID = "0"
		}

		styledID, ok := cache[baseID]
		if !ok {
			styledID, err = ensureStyledCharPr(charProperties, baseID, spec)
			if err != nil {
				return Report{}, nil, 0, err
			}
			cache[baseID] = styledID
		}

		setElementAttr(run, "charPrIDRef", styledID)
		usedIDs[styledID] = struct{}{}
	}

	if err := saveXML(headerDoc, headerPath); err != nil {
		return Report{}, nil, 0, err
	}
	if err := saveXML(sectionDoc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, nil, 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, nil, 0, err
	}
	return report, mapKeysSorted(usedIDs), len(targetRuns), nil
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
	parent := paragraph.Parent()
	if parent == nil {
		return Report{}, "", fmt.Errorf("paragraph has no parent: %d", paragraphIndex)
	}
	parent.RemoveChild(paragraph)

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

func AddNestedTable(targetDir string, tableIndex, row, col int, spec TableSpec) (Report, error) {
	if tableIndex < 0 || row < 0 || col < 0 {
		return Report{}, fmt.Errorf("table, row, and col must be zero or greater")
	}
	if spec.Rows <= 0 || spec.Cols <= 0 {
		return Report{}, fmt.Errorf("table rows and cols must be positive")
	}
	if err := ensureHeaderSupport(filepath.Join(targetDir, "Contents", "header.xml"), true, false); err != nil {
		return Report{}, err
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

	hasVisibleText := strings.TrimSpace(paragraphPlainText(subList)) != ""
	if !hasVisibleText {
		clearSubListParagraphs(subList)
	}

	counter := newIDCounter(root)
	nestedWidth := tableCellWidth(entry.cell)
	if nestedWidth <= 0 || nestedWidth > defaultTableWidth {
		nestedWidth = defaultTableWidth
	}
	subList.AddChild(newTableParagraphElementWithWidth(counter, spec, nestedWidth))

	currentHeight := tableCellHeight(entry.cell)
	nestedHeight := spec.Rows * defaultCellHeight
	targetHeight := nestedHeight
	if hasVisibleText {
		targetHeight += currentHeight
	}
	if currentHeight < targetHeight {
		setTableCellSize(entry.cell, tableCellWidth(entry.cell), targetHeight)
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

func RemoveHeader(targetDir string) (Report, error) {
	return removeHeaderFooter(targetDir, "header")
}

func RemoveFooter(targetDir string) (Report, error) {
	return removeHeaderFooter(targetDir, "footer")
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

func SetColumns(targetDir string, spec ColumnSpec) (Report, error) {
	if spec.Count <= 0 {
		return Report{}, fmt.Errorf("column count must be positive")
	}
	if spec.GapMM < 0 {
		return Report{}, fmt.Errorf("column gap must be zero or greater")
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

	sectionProperty := firstChildByTag(run, "hp:secPr")
	if sectionProperty == nil {
		return Report{}, fmt.Errorf("section run is missing hp:secPr")
	}

	gap := mmToHWPUnit(spec.GapMM)
	sectionProperty.RemoveAttr("spaceColumns")
	sectionProperty.CreateAttr("spaceColumns", strconv.Itoa(gap))

	replaceRunControl(run, "colPr", newColumnControl(run, spec.Count, gap))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, err
	}
	return report, nil
}

func SetPageLayout(targetDir string, spec PageLayoutSpec) (Report, error) {
	if !pageLayoutHasChanges(spec) {
		return Report{}, fmt.Errorf("page layout spec must include at least one option")
	}
	if err := validatePageLayoutSpec(spec); err != nil {
		return Report{}, err
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

	sectionProperty := firstChildByTag(run, "hp:secPr")
	if sectionProperty == nil {
		return Report{}, fmt.Errorf("section run is missing hp:secPr")
	}

	pagePr := ensureSectionPagePr(sectionProperty)
	applyPageLayoutToPagePr(pagePr, spec)
	if pageLayoutHasBorderChanges(spec) {
		applyPageLayoutToBorderFill(sectionProperty, spec)
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

func AddLine(targetDir string, spec LineSpec) (Report, string, int, int, error) {
	width := mmToHWPUnit(spec.WidthMM)
	height := mmToHWPUnit(spec.HeightMM)
	if width <= 0 || height <= 0 {
		return Report{}, "", 0, 0, fmt.Errorf("line width and height must be positive")
	}

	lineColor := strings.TrimSpace(spec.LineColor)
	if lineColor == "" {
		lineColor = "#000000"
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
	root.AddChild(newLineParagraphElement(counter, shapeID, width, height, lineColor))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", 0, 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", 0, 0, err
	}
	return report, shapeID, width, height, nil
}

func AddEllipse(targetDir string, spec EllipseSpec) (Report, string, int, int, error) {
	width := mmToHWPUnit(spec.WidthMM)
	height := mmToHWPUnit(spec.HeightMM)
	if width <= 0 || height <= 0 {
		return Report{}, "", 0, 0, fmt.Errorf("ellipse width and height must be positive")
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
	root.AddChild(newEllipseParagraphElement(counter, shapeID, width, height, lineColor, fillColor))

	if err := saveXML(doc, filepath.Join(targetDir, filepath.FromSlash(sectionPath))); err != nil {
		return Report{}, "", 0, 0, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, "", 0, 0, err
	}
	return report, shapeID, width, height, nil
}

func AddTextBox(targetDir string, spec TextBoxSpec) (Report, string, int, int, error) {
	width := mmToHWPUnit(spec.WidthMM)
	height := mmToHWPUnit(spec.HeightMM)
	if width <= 0 || height <= 0 {
		return Report{}, "", 0, 0, fmt.Errorf("textbox width and height must be positive")
	}

	hasVisibleText := false
	for _, text := range spec.Text {
		if strings.TrimSpace(text) != "" {
			hasVisibleText = true
			break
		}
	}
	if !hasVisibleText {
		return Report{}, "", 0, 0, fmt.Errorf("textbox text must not be empty")
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
	root.AddChild(newTextBoxParagraphElement(counter, shapeID, width, height, lineColor, fillColor, spec.Text))

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
	return elementPlainText(paragraph)
}

func elementPlainText(root *etree.Element) string {
	if root == nil {
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
	walk(root)
	return builder.String()
}

func editableRunsForParagraph(paragraph *etree.Element, runIndex *int) ([]*etree.Element, error) {
	runs := childElementsByTag(paragraph, "hp:run")
	if len(runs) == 0 {
		return nil, fmt.Errorf("paragraph has no editable runs")
	}

	if runIndex == nil {
		return runs, nil
	}
	if *runIndex < 0 || *runIndex >= len(runs) {
		return nil, fmt.Errorf("run index out of range: %d", *runIndex)
	}
	return []*etree.Element{runs[*runIndex]}, nil
}

func resolveInsertedRunCharPrIDRef(paragraph *etree.Element, runs []*etree.Element, insertIndex int) string {
	if insertIndex > 0 && insertIndex-1 < len(runs) {
		value := strings.TrimSpace(runs[insertIndex-1].SelectAttrValue("charPrIDRef", ""))
		if value != "" {
			return value
		}
	}
	if insertIndex < len(runs) {
		value := strings.TrimSpace(runs[insertIndex].SelectAttrValue("charPrIDRef", ""))
		if value != "" {
			return value
		}
	}
	return firstRunCharPrIDRef(paragraph)
}

func insertRunText(paragraph *etree.Element, runIndex int, charPrIDRef, text string) {
	run := etree.NewElement("hp:run")
	run.CreateAttr("charPrIDRef", fallbackString(strings.TrimSpace(charPrIDRef), "0"))
	textElement := run.CreateElement("hp:t")
	textElement.SetText(text)

	currentRun := 0
	for index, child := range paragraph.Child {
		element, ok := child.(*etree.Element)
		if !ok {
			continue
		}
		if tagMatches(element.Tag, "hp:run") {
			if currentRun == runIndex {
				paragraph.InsertChildAt(index, run)
				return
			}
			currentRun++
			continue
		}
		if currentRun == runIndex && tagMatches(element.Tag, "hp:linesegarray") {
			paragraph.InsertChildAt(index, run)
			return
		}
	}

	paragraph.AddChild(run)
}

func replaceRunText(run *etree.Element, text string) {
	if run == nil {
		return
	}
	for _, child := range append([]*etree.Element{}, run.ChildElements()...) {
		run.RemoveChild(child)
	}
	textElement := run.CreateElement("hp:t")
	textElement.SetText(text)
}

func refreshParagraphLineSeg(paragraph *etree.Element) {
	for _, child := range append([]*etree.Element{}, childElementsByTag(paragraph, "hp:linesegarray")...) {
		paragraph.RemoveChild(child)
	}
	paragraph.AddChild(newHeaderFooterLineSegElement(paragraphPlainText(paragraph)))
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

func ensureCharProperties(root *etree.Element) *etree.Element {
	refList := firstChildByTag(root, "hh:refList")
	if refList == nil {
		refList = etree.NewElement("hh:refList")
		root.AddChild(refList)
	}

	charProperties := firstChildByTag(refList, "hh:charProperties")
	if charProperties == nil {
		charProperties = etree.NewElement("hh:charProperties")
		refList.AddChild(charProperties)
	}

	if findCharPrByID(charProperties, "0") == nil {
		charProperties.AddChild(defaultCharPrElement())
	}
	setElementAttr(charProperties, "itemCnt", strconv.Itoa(len(childElementsByTag(charProperties, "hh:charPr"))))
	return charProperties
}

func ensureParagraphProperties(root *etree.Element) *etree.Element {
	refList := firstChildByTag(root, "hh:refList")
	if refList == nil {
		refList = etree.NewElement("hh:refList")
		root.AddChild(refList)
	}

	paragraphProperties := firstChildByTag(refList, "hh:paraProperties")
	if paragraphProperties == nil {
		paragraphProperties = etree.NewElement("hh:paraProperties")
		refList.AddChild(paragraphProperties)
	}

	if findParaPrByID(paragraphProperties, "0") == nil {
		paragraphProperties.AddChild(defaultParaPrElement())
	}
	setElementAttr(paragraphProperties, "itemCnt", strconv.Itoa(len(childElementsByTag(paragraphProperties, "hh:paraPr"))))
	return paragraphProperties
}

func defaultCharPrElement() *etree.Element {
	charPr := etree.NewElement("hh:charPr")
	charPr.CreateAttr("id", "0")
	charPr.CreateAttr("height", "1000")
	charPr.CreateAttr("textColor", "#000000")
	charPr.CreateAttr("shadeColor", "none")
	charPr.CreateAttr("useFontSpace", "0")
	charPr.CreateAttr("useKerning", "0")
	charPr.CreateAttr("symMark", "NONE")
	charPr.CreateAttr("borderFillIDRef", "2")

	fontRef := charPr.CreateElement("hh:fontRef")
	for _, key := range []string{"hangul", "latin", "hanja", "japanese", "other", "symbol", "user"} {
		fontRef.CreateAttr(key, "0")
	}

	ratio := charPr.CreateElement("hh:ratio")
	for _, key := range []string{"hangul", "latin", "hanja", "japanese", "other", "symbol", "user"} {
		ratio.CreateAttr(key, "100")
	}

	spacing := charPr.CreateElement("hh:spacing")
	for _, key := range []string{"hangul", "latin", "hanja", "japanese", "other", "symbol", "user"} {
		spacing.CreateAttr(key, "0")
	}

	relSz := charPr.CreateElement("hh:relSz")
	for _, key := range []string{"hangul", "latin", "hanja", "japanese", "other", "symbol", "user"} {
		relSz.CreateAttr(key, "100")
	}

	offset := charPr.CreateElement("hh:offset")
	for _, key := range []string{"hangul", "latin", "hanja", "japanese", "other", "symbol", "user"} {
		offset.CreateAttr(key, "0")
	}

	underline := charPr.CreateElement("hh:underline")
	underline.CreateAttr("type", "NONE")
	underline.CreateAttr("shape", "SOLID")
	underline.CreateAttr("color", "#000000")

	strikeout := charPr.CreateElement("hh:strikeout")
	strikeout.CreateAttr("shape", "NONE")
	strikeout.CreateAttr("color", "#000000")

	outline := charPr.CreateElement("hh:outline")
	outline.CreateAttr("type", "NONE")

	shadow := charPr.CreateElement("hh:shadow")
	shadow.CreateAttr("type", "NONE")
	shadow.CreateAttr("color", "#B2B2B2")
	shadow.CreateAttr("offsetX", "10")
	shadow.CreateAttr("offsetY", "10")

	return charPr
}

func defaultParaPrElement() *etree.Element {
	paraPr := etree.NewElement("hh:paraPr")
	paraPr.CreateAttr("id", "0")
	paraPr.CreateAttr("tabPrIDRef", "0")
	paraPr.CreateAttr("condense", "0")
	paraPr.CreateAttr("fontLineHeight", "0")
	paraPr.CreateAttr("snapToGrid", "1")
	paraPr.CreateAttr("suppressLineNumbers", "0")
	paraPr.CreateAttr("checked", "0")

	align := paraPr.CreateElement("hh:align")
	align.CreateAttr("horizontal", "JUSTIFY")
	align.CreateAttr("vertical", "BASELINE")

	heading := paraPr.CreateElement("hh:heading")
	heading.CreateAttr("type", "NONE")
	heading.CreateAttr("idRef", "0")
	heading.CreateAttr("level", "0")

	breakSetting := paraPr.CreateElement("hh:breakSetting")
	breakSetting.CreateAttr("breakLatinWord", "KEEP_WORD")
	breakSetting.CreateAttr("breakNonLatinWord", "KEEP_WORD")
	breakSetting.CreateAttr("widowOrphan", "0")
	breakSetting.CreateAttr("keepWithNext", "0")
	breakSetting.CreateAttr("keepLines", "0")
	breakSetting.CreateAttr("pageBreakBefore", "0")
	breakSetting.CreateAttr("lineWrap", "BREAK")

	autoSpacing := paraPr.CreateElement("hh:autoSpacing")
	autoSpacing.CreateAttr("eAsianEng", "0")
	autoSpacing.CreateAttr("eAsianNum", "0")

	switchElement := paraPr.CreateElement("hp:switch")
	caseElement := switchElement.CreateElement("hp:case")
	caseElement.CreateAttr("hp:required-namespace", "http://www.hancom.co.kr/hwpml/2016/HwpUnitChar")
	appendParaPrSpacing(caseElement, 0, 0, 0, 0, 0, 160, false)

	defaultElement := switchElement.CreateElement("hp:default")
	appendParaPrSpacing(defaultElement, 0, 0, 0, 0, 0, 160, true)

	border := paraPr.CreateElement("hh:border")
	border.CreateAttr("borderFillIDRef", "1")
	border.CreateAttr("offsetLeft", "0")
	border.CreateAttr("offsetRight", "0")
	border.CreateAttr("offsetTop", "0")
	border.CreateAttr("offsetBottom", "0")
	border.CreateAttr("connect", "0")
	border.CreateAttr("ignoreMargin", "0")
	return paraPr
}

func ensureStyledCharPr(charProperties *etree.Element, baseID string, spec TextStyleSpec) (string, error) {
	base := findCharPrByID(charProperties, baseID)
	if base == nil {
		base = findCharPrByID(charProperties, "0")
	}
	if base == nil {
		return "", fmt.Errorf("base charPr not found: %s", baseID)
	}

	nextID := strconv.Itoa(nextCharPrID(charProperties))
	cloned := base.Copy()
	setElementAttr(cloned, "id", nextID)
	applyTextStyleToCharPr(cloned, spec)
	charProperties.AddChild(cloned)
	setElementAttr(charProperties, "itemCnt", strconv.Itoa(len(childElementsByTag(charProperties, "hh:charPr"))))
	return nextID, nil
}

func ensureStyledParaPr(paragraphProperties *etree.Element, baseID string, apply func(*etree.Element) error) (string, error) {
	base := findParaPrByID(paragraphProperties, baseID)
	if base == nil {
		base = findParaPrByID(paragraphProperties, "0")
	}
	if base == nil {
		return "", fmt.Errorf("base paraPr not found: %s", baseID)
	}

	nextID := strconv.Itoa(nextParaPrID(paragraphProperties))
	cloned := base.Copy()
	setElementAttr(cloned, "id", nextID)
	if err := apply(cloned); err != nil {
		return "", err
	}
	paragraphProperties.AddChild(cloned)
	setElementAttr(paragraphProperties, "itemCnt", strconv.Itoa(len(childElementsByTag(paragraphProperties, "hh:paraPr"))))
	return nextID, nil
}

func findCharPrByID(charProperties *etree.Element, id string) *etree.Element {
	for _, child := range childElementsByTag(charProperties, "hh:charPr") {
		if strings.TrimSpace(child.SelectAttrValue("id", "")) == strings.TrimSpace(id) {
			return child
		}
	}
	return nil
}

func findParaPrByID(paragraphProperties *etree.Element, id string) *etree.Element {
	for _, child := range childElementsByTag(paragraphProperties, "hh:paraPr") {
		if strings.TrimSpace(child.SelectAttrValue("id", "")) == strings.TrimSpace(id) {
			return child
		}
	}
	return nil
}

func nextCharPrID(charProperties *etree.Element) int {
	maxID := -1
	for _, child := range childElementsByTag(charProperties, "hh:charPr") {
		value, err := strconv.Atoi(strings.TrimSpace(child.SelectAttrValue("id", "")))
		if err == nil && value > maxID {
			maxID = value
		}
	}
	return maxID + 1
}

func nextParaPrID(paragraphProperties *etree.Element) int {
	maxID := -1
	for _, child := range childElementsByTag(paragraphProperties, "hh:paraPr") {
		value, err := strconv.Atoi(strings.TrimSpace(child.SelectAttrValue("id", "")))
		if err == nil && value > maxID {
			maxID = value
		}
	}
	return maxID + 1
}

func applyTextStyleToCharPr(charPr *etree.Element, spec TextStyleSpec) {
	if spec.TextColor != "" {
		setElementAttr(charPr, "textColor", spec.TextColor)
	}

	if spec.Bold != nil {
		toggleMarkerElement(charPr, "hh:bold", *spec.Bold, "hh:underline")
	}
	if spec.Italic != nil {
		toggleMarkerElement(charPr, "hh:italic", *spec.Italic, "hh:underline")
	}
	if spec.Underline != nil {
		underline := firstChildByTag(charPr, "hh:underline")
		if underline == nil {
			underline = etree.NewElement("hh:underline")
			insertChildBeforeTag(charPr, underline, "hh:strikeout")
		}

		if *spec.Underline {
			setElementAttr(underline, "type", "BOTTOM")
		} else {
			setElementAttr(underline, "type", "NONE")
		}
		setElementAttr(underline, "shape", "SOLID")
		color := strings.TrimSpace(underline.SelectAttrValue("color", ""))
		if spec.TextColor != "" {
			color = spec.TextColor
		}
		if color == "" {
			color = "#000000"
		}
		setElementAttr(underline, "color", color)
	}
}

func paragraphLayoutHasChanges(spec ParagraphLayoutSpec) bool {
	return strings.TrimSpace(spec.Align) != "" ||
		spec.IndentMM != nil ||
		spec.LeftMarginMM != nil ||
		spec.RightMarginMM != nil ||
		spec.SpaceBeforeMM != nil ||
		spec.SpaceAfterMM != nil ||
		spec.LineSpacingPercent != nil
}

func applyParagraphLayoutToParaPr(paraPr *etree.Element, spec ParagraphLayoutSpec) {
	if value := normalizeParagraphAlign(spec.Align); value != "" {
		align := firstChildByTag(paraPr, "hh:align")
		if align == nil {
			align = etree.NewElement("hh:align")
			paraPr.InsertChildAt(0, align)
		}
		setElementAttr(align, "horizontal", value)
		if align.SelectAttr("vertical") == nil {
			align.CreateAttr("vertical", "BASELINE")
		}
	}

	caseValue := marginValueFromMM(spec.IndentMM)
	leftValue := marginValueFromMM(spec.LeftMarginMM)
	rightValue := marginValueFromMM(spec.RightMarginMM)
	prevValue := marginValueFromMM(spec.SpaceBeforeMM)
	nextValue := marginValueFromMM(spec.SpaceAfterMM)
	lineSpacing := 0
	if spec.LineSpacingPercent != nil {
		lineSpacing = *spec.LineSpacingPercent
	}

	updateParaPrSpacing(paraPr, caseValue, leftValue, rightValue, prevValue, nextValue, lineSpacing, spec)
}

func applyParagraphListToParaPr(refList, paraPr *etree.Element, spec ParagraphListSpec) error {
	heading := firstChildByTag(paraPr, "hh:heading")
	if heading == nil {
		heading = etree.NewElement("hh:heading")
		align := firstChildByTag(paraPr, "hh:align")
		insertIndex := 0
		if align != nil {
			for index, child := range paraPr.ChildElements() {
				if child == align {
					insertIndex = index + 1
					break
				}
			}
		}
		paraPr.InsertChildAt(insertIndex, heading)
	}

	kind := strings.ToUpper(strings.TrimSpace(spec.Kind))
	if kind == "NONE" {
		setElementAttr(heading, "type", "NONE")
		setElementAttr(heading, "idRef", "0")
		setElementAttr(heading, "level", "0")
		return nil
	}

	if kind == "BULLET" {
		idRef := ensureDefaultBullet(refList)
		setElementAttr(heading, "type", "BULLET")
		setElementAttr(heading, "idRef", idRef)
		setElementAttr(heading, "level", strconv.Itoa(spec.Level))
		setElementAttr(paraPr, "condense", "0")
		updateParaPrSpacing(paraPr, 0, maxInt(500*(spec.Level+1), 500), 0, 0, 0, 130, ParagraphLayoutSpec{
			LeftMarginMM:       ptrFloatFromHWPUnit(maxInt(500*(spec.Level+1), 500)),
			LineSpacingPercent: ptrInt(130),
		})
		return nil
	}

	numberingID := ensureDefaultNumbering(refList)
	if spec.StartNumber != nil && *spec.StartNumber > 1 {
		numberingID = cloneNumberingWithStart(refList, numberingID, *spec.StartNumber)
	}
	setElementAttr(heading, "type", "NUMBER")
	setElementAttr(heading, "idRef", numberingID)
	setElementAttr(heading, "level", strconv.Itoa(spec.Level))
	setElementAttr(paraPr, "condense", "20")
	updateParaPrSpacing(paraPr, 0, maxInt(1000*(spec.Level+1), 1000), 0, 0, 0, 160, ParagraphLayoutSpec{
		LeftMarginMM:       ptrFloatFromHWPUnit(maxInt(1000*(spec.Level+1), 1000)),
		LineSpacingPercent: ptrInt(160),
	})
	return nil
}

func normalizeParagraphAlign(value string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	switch normalized {
	case "":
		return ""
	case "LEFT", "RIGHT", "CENTER", "JUSTIFY", "DISTRIBUTE", "DISTRIBUTE_SPACE":
		return normalized
	default:
		return ""
	}
}

func updateParaPrSpacing(paraPr *etree.Element, indent, left, right, prev, next, lineSpacing int, spec ParagraphLayoutSpec) {
	switchElement := firstChildByTag(paraPr, "hp:switch")
	if switchElement == nil {
		switchElement = etree.NewElement("hp:switch")
		border := firstChildByTag(paraPr, "hh:border")
		if border != nil {
			insertIndex := len(paraPr.Child)
			for index, child := range paraPr.Child {
				element, ok := child.(*etree.Element)
				if ok && element == border {
					insertIndex = index
					break
				}
			}
			paraPr.InsertChildAt(insertIndex, switchElement)
		} else {
			paraPr.AddChild(switchElement)
		}
	}

	caseElement := firstChildByTag(switchElement, "hp:case")
	if caseElement == nil {
		caseElement = etree.NewElement("hp:case")
		caseElement.CreateAttr("hp:required-namespace", "http://www.hancom.co.kr/hwpml/2016/HwpUnitChar")
		switchElement.InsertChildAt(0, caseElement)
	}
	if caseElement.SelectAttr("hp:required-namespace") == nil {
		caseElement.CreateAttr("hp:required-namespace", "http://www.hancom.co.kr/hwpml/2016/HwpUnitChar")
	}

	defaultElement := firstChildByTag(switchElement, "hp:default")
	if defaultElement == nil {
		defaultElement = etree.NewElement("hp:default")
		switchElement.AddChild(defaultElement)
	}

	applyParaPrSpacingBranch(caseElement, indent, left, right, prev, next, lineSpacing, spec, false)
	applyParaPrSpacingBranch(defaultElement, indent*2, left*2, right*2, prev*2, next*2, lineSpacing, spec, true)
}

func applyParaPrSpacingBranch(parent *etree.Element, indent, left, right, prev, next, lineSpacing int, spec ParagraphLayoutSpec, legacy bool) {
	margin := firstChildByTag(parent, "hh:margin")
	if margin == nil {
		margin = etree.NewElement("hh:margin")
		parent.AddChild(margin)
	}
	setMarginChildValue(margin, "hc:intent", indent, spec.IndentMM != nil)
	setMarginChildValue(margin, "hc:left", left, spec.LeftMarginMM != nil)
	setMarginChildValue(margin, "hc:right", right, spec.RightMarginMM != nil)
	setMarginChildValue(margin, "hc:prev", prev, spec.SpaceBeforeMM != nil)
	setMarginChildValue(margin, "hc:next", next, spec.SpaceAfterMM != nil)

	lineSpacingElement := firstChildByTag(parent, "hh:lineSpacing")
	if lineSpacingElement == nil {
		lineSpacingElement = etree.NewElement("hh:lineSpacing")
		parent.AddChild(lineSpacingElement)
	}
	if spec.LineSpacingPercent != nil {
		setElementAttr(lineSpacingElement, "type", "PERCENT")
		setElementAttr(lineSpacingElement, "value", strconv.Itoa(lineSpacing))
		setElementAttr(lineSpacingElement, "unit", "HWPUNIT")
	}
	_ = legacy
}

func setMarginChildValue(margin *etree.Element, tag string, value int, changed bool) {
	child := firstChildByTag(margin, tag)
	if child == nil {
		child = etree.NewElement(tag)
		margin.AddChild(child)
	}
	if !changed && child.SelectAttr("value") != nil {
		return
	}
	setElementAttr(child, "value", strconv.Itoa(maxInt(value, 0)))
	setElementAttr(child, "unit", "HWPUNIT")
}

func appendParaPrSpacing(parent *etree.Element, indent, left, right, prev, next, lineSpacing int, legacy bool) {
	margin := parent.CreateElement("hh:margin")
	for _, item := range []struct {
		tag   string
		value int
	}{
		{tag: "hc:intent", value: indent},
		{tag: "hc:left", value: left},
		{tag: "hc:right", value: right},
		{tag: "hc:prev", value: prev},
		{tag: "hc:next", value: next},
	} {
		child := margin.CreateElement(item.tag)
		child.CreateAttr("value", strconv.Itoa(item.value))
		child.CreateAttr("unit", "HWPUNIT")
	}

	lineSpacingElement := parent.CreateElement("hh:lineSpacing")
	lineSpacingElement.CreateAttr("type", "PERCENT")
	lineSpacingElement.CreateAttr("value", strconv.Itoa(lineSpacing))
	lineSpacingElement.CreateAttr("unit", "HWPUNIT")
	_ = legacy
}

func marginValueFromMM(value *float64) int {
	if value == nil {
		return 0
	}
	return mmToHWPUnit(*value)
}

func ptrFloatFromHWPUnit(value int) *float64 {
	converted := float64(value) * 25.4 / 7200.0
	return &converted
}

func ptrInt(value int) *int {
	return &value
}

func ensureDefaultNumbering(refList *etree.Element) string {
	numberings := firstChildByTag(refList, "hh:numberings")
	if numberings == nil {
		numberings = etree.NewElement("hh:numberings")
		refList.AddChild(numberings)
	}
	for _, numbering := range childElementsByTag(numberings, "hh:numbering") {
		if id := strings.TrimSpace(numbering.SelectAttrValue("id", "")); id != "" {
			setElementAttr(numberings, "itemCnt", strconv.Itoa(len(childElementsByTag(numberings, "hh:numbering"))))
			return id
		}
	}

	numbering := etree.NewElement("hh:numbering")
	numbering.CreateAttr("id", "1")
	numbering.CreateAttr("start", "1")
	for level := 1; level <= 7; level++ {
		paraHead := numbering.CreateElement("hh:paraHead")
		paraHead.CreateAttr("start", "1")
		paraHead.CreateAttr("level", strconv.Itoa(level))
		paraHead.CreateAttr("align", "LEFT")
		paraHead.CreateAttr("useInstWidth", "1")
		paraHead.CreateAttr("autoIndent", "1")
		paraHead.CreateAttr("widthAdjust", "0")
		paraHead.CreateAttr("textOffsetType", "PERCENT")
		paraHead.CreateAttr("textOffset", "50")
		paraHead.CreateAttr("numFormat", "DIGIT")
		paraHead.CreateAttr("charPrIDRef", "4294967295")
		paraHead.CreateAttr("checkable", "0")
		paraHead.SetText(numberingFormatForLevel(level))
	}
	numberings.AddChild(numbering)
	setElementAttr(numberings, "itemCnt", strconv.Itoa(len(childElementsByTag(numberings, "hh:numbering"))))
	return "1"
}

func ensureDefaultBullet(refList *etree.Element) string {
	bullets := firstChildByTag(refList, "hh:bullets")
	if bullets == nil {
		bullets = etree.NewElement("hh:bullets")
		refList.AddChild(bullets)
	}
	for _, bullet := range childElementsByTag(bullets, "hh:bullet") {
		if id := strings.TrimSpace(bullet.SelectAttrValue("id", "")); id != "" {
			setElementAttr(bullets, "itemCnt", strconv.Itoa(len(childElementsByTag(bullets, "hh:bullet"))))
			return id
		}
	}

	bullet := etree.NewElement("hh:bullet")
	bullet.CreateAttr("id", "1")
	bullet.CreateAttr("char", "•")
	bullet.CreateAttr("useImage", "0")
	paraHead := bullet.CreateElement("hh:paraHead")
	paraHead.CreateAttr("level", "0")
	paraHead.CreateAttr("align", "LEFT")
	paraHead.CreateAttr("useInstWidth", "0")
	paraHead.CreateAttr("autoIndent", "1")
	paraHead.CreateAttr("widthAdjust", "0")
	paraHead.CreateAttr("textOffsetType", "PERCENT")
	paraHead.CreateAttr("textOffset", "0")
	paraHead.CreateAttr("numFormat", "DIGIT")
	paraHead.CreateAttr("charPrIDRef", "4294967295")
	paraHead.CreateAttr("checkable", "0")
	bullets.AddChild(bullet)
	setElementAttr(bullets, "itemCnt", strconv.Itoa(len(childElementsByTag(bullets, "hh:bullet"))))
	return "1"
}

func cloneNumberingWithStart(refList *etree.Element, baseID string, start int) string {
	numberings := firstChildByTag(refList, "hh:numberings")
	if numberings == nil {
		return ensureDefaultNumbering(refList)
	}

	base := (*etree.Element)(nil)
	for _, numbering := range childElementsByTag(numberings, "hh:numbering") {
		if strings.TrimSpace(numbering.SelectAttrValue("id", "")) == strings.TrimSpace(baseID) {
			base = numbering
			break
		}
	}
	if base == nil {
		return ensureDefaultNumbering(refList)
	}

	nextID := 1
	for _, numbering := range childElementsByTag(numberings, "hh:numbering") {
		value, err := strconv.Atoi(strings.TrimSpace(numbering.SelectAttrValue("id", "")))
		if err == nil && value >= nextID {
			nextID = value + 1
		}
	}

	cloned := base.Copy()
	setElementAttr(cloned, "id", strconv.Itoa(nextID))
	setElementAttr(cloned, "start", strconv.Itoa(start))
	for _, paraHead := range childElementsByTag(cloned, "hh:paraHead") {
		setElementAttr(paraHead, "start", strconv.Itoa(start))
	}
	numberings.AddChild(cloned)
	setElementAttr(numberings, "itemCnt", strconv.Itoa(len(childElementsByTag(numberings, "hh:numbering"))))
	return strconv.Itoa(nextID)
}

func numberingFormatForLevel(level int) string {
	parts := make([]string, 0, level)
	for index := 1; index <= level; index++ {
		parts = append(parts, fmt.Sprintf("^%d", index))
	}
	return strings.Join(parts, ".") + "."
}

func toggleMarkerElement(root *etree.Element, tag string, enabled bool, beforeTag string) {
	child := firstChildByTag(root, tag)
	if enabled {
		if child == nil {
			child = etree.NewElement(tag)
			insertChildBeforeTag(root, child, beforeTag)
		}
		return
	}
	if child != nil {
		root.RemoveChild(child)
	}
}

func insertChildBeforeTag(root, child *etree.Element, beforeTag string) {
	if root == nil || child == nil {
		return
	}
	for index, existing := range root.Child {
		element, ok := existing.(*etree.Element)
		if !ok {
			continue
		}
		if tagMatches(element.Tag, beforeTag) {
			root.InsertChildAt(index, child)
			return
		}
	}
	root.AddChild(child)
}

func textStyleHasChanges(spec TextStyleSpec) bool {
	return spec.Bold != nil || spec.Italic != nil || spec.Underline != nil || strings.TrimSpace(spec.TextColor) != ""
}

type charPrState struct {
	CharPrIDRef string
	Bold        bool
	Italic      bool
	Underline   bool
	TextColor   string
}

func buildCharPrStateMap(charProperties *etree.Element) map[string]charPrState {
	states := map[string]charPrState{
		"0": {
			CharPrIDRef: "0",
			TextColor:   "#000000",
		},
	}
	if charProperties == nil {
		return states
	}

	for _, charPr := range childElementsByTag(charProperties, "hh:charPr") {
		id := strings.TrimSpace(charPr.SelectAttrValue("id", ""))
		if id == "" {
			continue
		}
		underline := false
		if underlineElement := firstChildByTag(charPr, "hh:underline"); underlineElement != nil {
			underline = !strings.EqualFold(strings.TrimSpace(underlineElement.SelectAttrValue("type", "NONE")), "NONE")
		}
		textColor := strings.TrimSpace(charPr.SelectAttrValue("textColor", ""))
		if textColor == "" {
			textColor = "#000000"
		}
		states[id] = charPrState{
			CharPrIDRef: id,
			Bold:        firstChildByTag(charPr, "hh:bold") != nil,
			Italic:      firstChildByTag(charPr, "hh:italic") != nil,
			Underline:   underline,
			TextColor:   strings.ToUpper(textColor),
		}
	}
	return states
}

func resolveRunStyleState(run *etree.Element, states map[string]charPrState) charPrState {
	charPrIDRef := fallbackString(strings.TrimSpace(run.SelectAttrValue("charPrIDRef", "")), "0")
	if state, ok := states[charPrIDRef]; ok {
		return state
	}
	return charPrState{
		CharPrIDRef: charPrIDRef,
		TextColor:   "#000000",
	}
}

func runStyleMatchesFilter(state charPrState, filter RunStyleFilter) bool {
	if filter.Bold != nil && state.Bold != *filter.Bold {
		return false
	}
	if filter.Italic != nil && state.Italic != *filter.Italic {
		return false
	}
	if filter.Underline != nil && state.Underline != *filter.Underline {
		return false
	}
	if filter.TextColor != "" && !strings.EqualFold(state.TextColor, filter.TextColor) {
		return false
	}
	return true
}

func runStyleFilterHasConditions(filter RunStyleFilter) bool {
	return filter.Bold != nil || filter.Italic != nil || filter.Underline != nil || strings.TrimSpace(filter.TextColor) != ""
}

func normalizeObjectTypeFilter(types []string) map[string]struct{} {
	if len(types) == 0 {
		return nil
	}

	filter := make(map[string]struct{}, len(types))
	for _, entry := range types {
		value := strings.ToLower(strings.TrimSpace(entry))
		if value == "" {
			continue
		}
		filter[value] = struct{}{}
	}
	if len(filter) == 0 {
		return nil
	}
	return filter
}

func collectObjectMatches(root *etree.Element, paragraphIndex, runIndex int, path string, typeFilter map[string]struct{}, nextIndex *int, matches *[]ObjectMatch) {
	if root == nil {
		return
	}

	if objectType, ok := classifyObjectElement(root); ok && objectTypeAllowed(objectType, typeFilter) {
		match := ObjectMatch{
			Index:     *nextIndex,
			Type:      objectType,
			Paragraph: paragraphIndex,
			Run:       runIndex,
			Path:      path,
			Tag:       root.Tag,
			ID:        objectElementID(root),
			Ref:       objectElementRef(root, objectType),
			Text:      objectElementText(root, objectType),
		}
		if objectType == "table" {
			match.Rows, match.Cols = tableDimensions(root)
		}
		*matches = append(*matches, match)
		*nextIndex = *nextIndex + 1
	}

	counts := make(map[string]int)
	for _, child := range root.ChildElements() {
		childIndex := counts[child.Tag]
		counts[child.Tag] = childIndex + 1
		childPath := fmt.Sprintf("%s/%s[%d]", path, child.Tag, childIndex)
		collectObjectMatches(child, paragraphIndex, runIndex, childPath, typeFilter, nextIndex, matches)
	}
}

func findObjectElementByTypeAndIndex(root *etree.Element, objectType string, targetIndex int) *etree.Element {
	current := 0
	for _, paragraph := range editableParagraphs(root) {
		for _, run := range childElementsByTag(paragraph, "hp:run") {
			if element := findObjectElementRecursive(run, objectType, targetIndex, &current); element != nil {
				return element
			}
		}
	}
	return nil
}

func findObjectElementRecursive(root *etree.Element, objectType string, targetIndex int, current *int) *etree.Element {
	if root == nil {
		return nil
	}

	if classified, ok := classifyObjectElement(root); ok && classified == objectType {
		if *current == targetIndex {
			return root
		}
		*current = *current + 1
	}

	for _, child := range root.ChildElements() {
		if element := findObjectElementRecursive(child, objectType, targetIndex, current); element != nil {
			return element
		}
	}
	return nil
}

func classifyObjectElement(element *etree.Element) (string, bool) {
	switch {
	case tagMatches(element.Tag, "hp:tbl"):
		return "table", true
	case tagMatches(element.Tag, "hp:pic"):
		return "image", true
	case tagMatches(element.Tag, "hp:equation"):
		return "equation", true
	case tagMatches(element.Tag, "hp:line"):
		return "line", true
	case tagMatches(element.Tag, "hp:ellipse"):
		return "ellipse", true
	case tagMatches(element.Tag, "hp:rect"):
		if firstChildByTag(element, "hp:drawText") != nil {
			return "textbox", true
		}
		return "rectangle", true
	default:
		return "", false
	}
}

func objectTypeAllowed(objectType string, typeFilter map[string]struct{}) bool {
	if len(typeFilter) == 0 {
		return true
	}
	_, ok := typeFilter[objectType]
	return ok
}

func objectElementID(element *etree.Element) string {
	for _, key := range []string{"id", "instid", "itemIDRef"} {
		if value := strings.TrimSpace(element.SelectAttrValue(key, "")); value != "" {
			return value
		}
	}
	return ""
}

func objectElementRef(element *etree.Element, objectType string) string {
	if objectType != "image" {
		return ""
	}

	image := firstChildByTag(element, "hc:img")
	if image == nil {
		return ""
	}
	return strings.TrimSpace(image.SelectAttrValue("binaryItemIDRef", ""))
}

func objectElementText(element *etree.Element, objectType string) string {
	var text string
	switch objectType {
	case "equation":
		text = elementPlainText(firstChildByTag(element, "hp:script"))
	case "image":
		text = ""
	default:
		text = elementPlainText(element)
	}
	return strings.TrimSpace(text)
}

func normalizePositionAlign(value string, allowed []string) string {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return ""
	}
	for _, candidate := range allowed {
		if normalized == candidate {
			return normalized
		}
	}
	return ""
}

func collectTagMatches(root *etree.Element, paragraphIndex, runIndex int, path, tag string, nextIndex *int, matches *[]TagMatch) {
	if root == nil {
		return
	}

	if tagMatches(root.Tag, tag) {
		*matches = append(*matches, TagMatch{
			Index:     *nextIndex,
			Paragraph: paragraphIndex,
			Run:       runIndex,
			Path:      path,
			Tag:       root.Tag,
			Text:      strings.TrimSpace(elementPlainText(root)),
		})
		*nextIndex = *nextIndex + 1
	}

	counts := make(map[string]int)
	for _, child := range root.ChildElements() {
		childIndex := counts[child.Tag]
		counts[child.Tag] = childIndex + 1
		childPath := fmt.Sprintf("%s/%s[%d]", path, child.Tag, childIndex)
		collectTagMatches(child, paragraphIndex, runIndex, childPath, tag, nextIndex, matches)
	}
}

func collectAttributeMatches(root *etree.Element, paragraphIndex, runIndex int, path string, filter AttributeFilter, nextIndex *int, matches *[]AttributeMatch) {
	if root == nil {
		return
	}

	if attributeTagAllowed(root.Tag, filter.Tag) {
		for _, attr := range root.Attr {
			if !attributeNameMatches(attr.Key, filter.Attr) {
				continue
			}
			if !attributeValueMatches(attr.Value, filter.Value) {
				continue
			}
			*matches = append(*matches, AttributeMatch{
				Index:     *nextIndex,
				Paragraph: paragraphIndex,
				Run:       runIndex,
				Path:      path,
				Tag:       root.Tag,
				Attr:      attr.Key,
				Value:     attr.Value,
				Text:      strings.TrimSpace(elementPlainText(root)),
			})
			*nextIndex = *nextIndex + 1
		}
	}

	counts := make(map[string]int)
	for _, child := range root.ChildElements() {
		childIndex := counts[child.Tag]
		counts[child.Tag] = childIndex + 1
		childPath := fmt.Sprintf("%s/%s[%d]", path, child.Tag, childIndex)
		collectAttributeMatches(child, paragraphIndex, runIndex, childPath, filter, nextIndex, matches)
	}
}

type searchContext struct {
	Paragraph int
	Run       int
	Path      string
}

func collectSearchContexts(root *etree.Element, context searchContext, paragraphIndexes map[*etree.Element]int, contexts map[*etree.Element]searchContext) {
	if root == nil {
		return
	}
	contexts[root] = context

	counts := make(map[string]int)
	currentParagraph := context.Paragraph
	currentRun := context.Run
	for _, child := range root.ChildElements() {
		childIndex := counts[child.Tag]
		counts[child.Tag] = childIndex + 1

		childContext := searchContext{
			Paragraph: currentParagraph,
			Run:       currentRun,
			Path:      fmt.Sprintf("%s/%s[%d]", context.Path, child.Tag, childIndex),
		}
		if currentParagraph == -1 && tagMatches(child.Tag, "hp:p") {
			if paragraphIndex, ok := paragraphIndexes[child]; ok {
				childContext.Paragraph = paragraphIndex
			}
			childContext.Run = -1
		}
		if childContext.Run == -1 && childContext.Paragraph >= 0 && tagMatches(root.Tag, "hp:p") && tagMatches(child.Tag, "hp:run") {
			childContext.Run = len(childElementsBeforeTag(root, child, "hp:run"))
		}
		collectSearchContexts(child, childContext, paragraphIndexes, contexts)
	}
}

func editableParagraphIndexMap(root *etree.Element) map[*etree.Element]int {
	indexes := make(map[*etree.Element]int)
	for index, paragraph := range editableParagraphs(root) {
		indexes[paragraph] = index
	}
	return indexes
}

func childElementsBeforeTag(root, target *etree.Element, tag string) []*etree.Element {
	if root == nil || target == nil {
		return nil
	}

	result := make([]*etree.Element, 0)
	for _, child := range root.ChildElements() {
		if child == target {
			break
		}
		if tagMatches(child.Tag, tag) {
			result = append(result, child)
		}
	}
	return result
}

func attributeTagAllowed(actualTag, expectedTag string) bool {
	expectedTag = strings.TrimSpace(expectedTag)
	if expectedTag == "" {
		return true
	}
	return tagMatches(actualTag, expectedTag)
}

func attributeNameMatches(actualAttr, expectedAttr string) bool {
	expectedAttr = strings.TrimSpace(expectedAttr)
	if expectedAttr == "" {
		return false
	}
	return actualAttr == expectedAttr || localTag(actualAttr) == localTag(expectedAttr)
}

func attributeValueMatches(actualValue, expectedValue string) bool {
	expectedValue = strings.TrimSpace(expectedValue)
	if expectedValue == "" {
		return true
	}
	return actualValue == expectedValue
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

func removeHeaderFooter(targetDir, tag string) (Report, error) {
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

	replaceRunControl(run, tag, nil)

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
	if ctrl != nil {
		run.AddChild(ctrl)
	}
}

func newColumnControl(run *etree.Element, count, gap int) *etree.Element {
	existing := (*etree.Element)(nil)
	for _, child := range run.ChildElements() {
		if !tagMatches(child.Tag, "hp:ctrl") {
			continue
		}
		for _, nested := range child.ChildElements() {
			if tagMatches(nested.Tag, "hp:colPr") {
				existing = nested
				break
			}
		}
		if existing != nil {
			break
		}
	}

	ctrl := etree.NewElement("hp:ctrl")
	colPr := ctrl.CreateElement("hp:colPr")
	colPr.CreateAttr("id", "")
	colPr.CreateAttr("type", "NEWSPAPER")
	colPr.CreateAttr("layout", "LEFT")
	colPr.CreateAttr("colCount", strconv.Itoa(count))
	colPr.CreateAttr("sameSz", "1")
	colPr.CreateAttr("sameGap", strconv.Itoa(gap))

	if existing != nil {
		for _, child := range existing.ChildElements() {
			colPr.AddChild(child.Copy())
		}
	}
	if firstChildByTag(colPr, "hp:colLine") == nil {
		colLine := colPr.CreateElement("hp:colLine")
		colLine.CreateAttr("type", "NONE")
		colLine.CreateAttr("width", "0.1 mm")
		colLine.CreateAttr("color", "#000000")
	}
	return ctrl
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

func pageLayoutHasChanges(spec PageLayoutSpec) bool {
	return spec.Orientation != "" ||
		spec.WidthMM != nil ||
		spec.HeightMM != nil ||
		spec.LeftMarginMM != nil ||
		spec.RightMarginMM != nil ||
		spec.TopMarginMM != nil ||
		spec.BottomMarginMM != nil ||
		spec.HeaderMarginMM != nil ||
		spec.FooterMarginMM != nil ||
		spec.GutterMarginMM != nil ||
		spec.GutterType != "" ||
		pageLayoutHasBorderChanges(spec)
}

func pageLayoutHasBorderChanges(spec PageLayoutSpec) bool {
	return spec.BorderFillIDRef != nil ||
		spec.BorderTextBorder != "" ||
		spec.BorderFillArea != "" ||
		spec.BorderHeaderInside != nil ||
		spec.BorderFooterInside != nil ||
		spec.BorderOffsetLeftMM != nil ||
		spec.BorderOffsetRightMM != nil ||
		spec.BorderOffsetTopMM != nil ||
		spec.BorderOffsetBottomMM != nil
}

func validatePageLayoutSpec(spec PageLayoutSpec) error {
	if spec.Orientation != "" && !isAllowedPageLayoutOrientation(strings.ToUpper(spec.Orientation)) {
		return fmt.Errorf("orientation must be PORTRAIT or LANDSCAPE")
	}
	if err := validatePositiveMM(spec.WidthMM, "width"); err != nil {
		return err
	}
	if err := validatePositiveMM(spec.HeightMM, "height"); err != nil {
		return err
	}
	for _, item := range []struct {
		name  string
		value *float64
	}{
		{name: "left margin", value: spec.LeftMarginMM},
		{name: "right margin", value: spec.RightMarginMM},
		{name: "top margin", value: spec.TopMarginMM},
		{name: "bottom margin", value: spec.BottomMarginMM},
		{name: "header margin", value: spec.HeaderMarginMM},
		{name: "footer margin", value: spec.FooterMarginMM},
		{name: "gutter margin", value: spec.GutterMarginMM},
		{name: "border offset left", value: spec.BorderOffsetLeftMM},
		{name: "border offset right", value: spec.BorderOffsetRightMM},
		{name: "border offset top", value: spec.BorderOffsetTopMM},
		{name: "border offset bottom", value: spec.BorderOffsetBottomMM},
	} {
		if err := validateNonNegativeMM(item.value, item.name); err != nil {
			return err
		}
	}
	if spec.BorderFillIDRef != nil && *spec.BorderFillIDRef < 0 {
		return fmt.Errorf("border fill id must be zero or greater")
	}
	return nil
}

func validatePositiveMM(value *float64, name string) error {
	if value != nil && *value <= 0 {
		return fmt.Errorf("%s must be positive", name)
	}
	return nil
}

func validateNonNegativeMM(value *float64, name string) error {
	if value != nil && *value < 0 {
		return fmt.Errorf("%s must be zero or greater", name)
	}
	return nil
}

func ensureSectionPagePr(sectionProperty *etree.Element) *etree.Element {
	pagePr := firstChildByTag(sectionProperty, "hp:pagePr")
	if pagePr != nil {
		return pagePr
	}

	pagePr = etree.NewElement("hp:pagePr")
	pagePr.CreateAttr("landscape", "WIDELY")
	pagePr.CreateAttr("width", "59528")
	pagePr.CreateAttr("height", "84186")
	pagePr.CreateAttr("gutterType", "LEFT_ONLY")
	pagePr.CreateElement("hp:margin")
	insertChildBeforeTag(sectionProperty, pagePr, "hp:footNotePr")
	return pagePr
}

func applyPageLayoutToPagePr(pagePr *etree.Element, spec PageLayoutSpec) {
	width := attrIntValue(pagePr, "width", 59528)
	height := attrIntValue(pagePr, "height", 84186)

	if spec.WidthMM != nil {
		width = mmToHWPUnit(*spec.WidthMM)
	}
	if spec.HeightMM != nil {
		height = mmToHWPUnit(*spec.HeightMM)
	}

	orientation := strings.ToUpper(strings.TrimSpace(spec.Orientation))
	if orientation != "" {
		setElementAttr(pagePr, "landscape", "WIDELY")
		if width > 0 && height > 0 {
			if orientation == "PORTRAIT" && width > height {
				width, height = height, width
			}
			if orientation == "LANDSCAPE" && width < height {
				width, height = height, width
			}
		}
	}

	if width > 0 {
		setElementAttr(pagePr, "width", strconv.Itoa(width))
	}
	if height > 0 {
		setElementAttr(pagePr, "height", strconv.Itoa(height))
	}
	if strings.TrimSpace(spec.GutterType) != "" {
		setElementAttr(pagePr, "gutterType", strings.ToUpper(strings.TrimSpace(spec.GutterType)))
	}

	margin := firstChildByTag(pagePr, "hp:margin")
	if margin == nil {
		margin = pagePr.CreateElement("hp:margin")
	}
	setOptionalMMAttr(margin, "left", spec.LeftMarginMM)
	setOptionalMMAttr(margin, "right", spec.RightMarginMM)
	setOptionalMMAttr(margin, "top", spec.TopMarginMM)
	setOptionalMMAttr(margin, "bottom", spec.BottomMarginMM)
	setOptionalMMAttr(margin, "header", spec.HeaderMarginMM)
	setOptionalMMAttr(margin, "footer", spec.FooterMarginMM)
	setOptionalMMAttr(margin, "gutter", spec.GutterMarginMM)
}

func setOptionalMMAttr(root *etree.Element, key string, value *float64) {
	if value == nil {
		return
	}
	setElementAttr(root, key, strconv.Itoa(mmToHWPUnit(*value)))
}

func attrIntValue(root *etree.Element, key string, fallback int) int {
	if root == nil {
		return fallback
	}
	attr := root.SelectAttr(key)
	if attr == nil {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(attr.Value))
	if err != nil {
		return fallback
	}
	return value
}

func applyPageLayoutToBorderFill(sectionProperty *etree.Element, spec PageLayoutSpec) {
	for _, pageBorderFill := range ensureSectionPageBorderFills(sectionProperty) {
		if spec.BorderFillIDRef != nil {
			setElementAttr(pageBorderFill, "borderFillIDRef", strconv.Itoa(*spec.BorderFillIDRef))
		}
		if strings.TrimSpace(spec.BorderTextBorder) != "" {
			setElementAttr(pageBorderFill, "textBorder", strings.ToUpper(strings.TrimSpace(spec.BorderTextBorder)))
		}
		if strings.TrimSpace(spec.BorderFillArea) != "" {
			setElementAttr(pageBorderFill, "fillArea", strings.ToUpper(strings.TrimSpace(spec.BorderFillArea)))
		}
		if spec.BorderHeaderInside != nil {
			setElementAttr(pageBorderFill, "headerInside", boolToIntString(*spec.BorderHeaderInside))
		}
		if spec.BorderFooterInside != nil {
			setElementAttr(pageBorderFill, "footerInside", boolToIntString(*spec.BorderFooterInside))
		}

		offset := firstChildByTag(pageBorderFill, "hp:offset")
		if offset == nil {
			offset = pageBorderFill.CreateElement("hp:offset")
		}
		setOptionalMMAttr(offset, "left", spec.BorderOffsetLeftMM)
		setOptionalMMAttr(offset, "right", spec.BorderOffsetRightMM)
		setOptionalMMAttr(offset, "top", spec.BorderOffsetTopMM)
		setOptionalMMAttr(offset, "bottom", spec.BorderOffsetBottomMM)
	}
}

func ensureSectionPageBorderFills(sectionProperty *etree.Element) []*etree.Element {
	pageBorderFills := childElementsByTag(sectionProperty, "hp:pageBorderFill")
	if len(pageBorderFills) > 0 {
		return pageBorderFills
	}

	for _, pageType := range []string{"BOTH", "EVEN", "ODD"} {
		pageBorderFill := etree.NewElement("hp:pageBorderFill")
		pageBorderFill.CreateAttr("type", pageType)
		pageBorderFill.CreateAttr("borderFillIDRef", "1")
		pageBorderFill.CreateAttr("textBorder", "PAPER")
		pageBorderFill.CreateAttr("headerInside", "0")
		pageBorderFill.CreateAttr("footerInside", "0")
		pageBorderFill.CreateAttr("fillArea", "PAPER")

		offset := pageBorderFill.CreateElement("hp:offset")
		offset.CreateAttr("left", "1417")
		offset.CreateAttr("right", "1417")
		offset.CreateAttr("top", "1417")
		offset.CreateAttr("bottom", "1417")
		sectionProperty.AddChild(pageBorderFill)
		pageBorderFills = append(pageBorderFills, pageBorderFill)
	}

	return pageBorderFills
}

func boolToIntString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func isAllowedPageLayoutOrientation(value string) bool {
	return value == "PORTRAIT" || value == "LANDSCAPE"
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

func RecordHistory(targetDir string, spec HistoryEntrySpec) error {
	if strings.TrimSpace(spec.Command) == "" {
		return fmt.Errorf("history command is required")
	}

	if spec.Timestamp.IsZero() {
		spec.Timestamp = time.Now()
	}
	if strings.TrimSpace(spec.Author) == "" {
		spec.Author = "hwpxctl"
	}
	if strings.TrimSpace(spec.Summary) == "" {
		spec.Summary = spec.Command
	}

	contentPath := filepath.Join(targetDir, "Contents", "content.hpf")
	contentDoc, err := loadXML(contentPath)
	if err != nil {
		return err
	}

	contentRoot := contentDoc.Root()
	if contentRoot == nil {
		return fmt.Errorf("content.hpf has no root")
	}

	manifestChanged, err := ensureHistoryManifestItem(contentRoot)
	if err != nil {
		return err
	}
	if manifestChanged {
		if err := saveXML(contentDoc, contentPath); err != nil {
			return err
		}
	}

	historyPath := filepath.Join(targetDir, "Contents", "history.xml")
	historyDoc, err := loadOrCreateHistoryXML(historyPath)
	if err != nil {
		return err
	}

	historyRoot := historyDoc.Root()
	if historyRoot == nil {
		return fmt.Errorf("history.xml has no root")
	}

	historyRoot.AddChild(newHistoryEntryElement(nextHistoryEntryID(historyRoot), spec))
	return saveXML(historyDoc, historyPath)
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

func loadOrCreateHistoryXML(path string) (*etree.Document, error) {
	if _, err := os.Stat(path); err == nil {
		return loadXML(path)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)

	root := doc.CreateElement("hh:history")
	root.CreateAttr("xmlns:hh", "http://www.hancom.co.kr/hwpml/2011/head")
	root.CreateAttr("version", "1.0")
	return doc, nil
}

func ensureHistoryManifestItem(root *etree.Element) (bool, error) {
	manifest := firstChildByTag(root, "opf:manifest")
	if manifest == nil {
		return false, fmt.Errorf("content.hpf is missing opf:manifest")
	}

	for _, item := range childElementsByTag(manifest, "opf:item") {
		if item.SelectAttrValue("id", "") == "history" || item.SelectAttrValue("href", "") == "Contents/history.xml" {
			return false, nil
		}
	}

	item := etree.NewElement("opf:item")
	item.CreateAttr("id", "history")
	item.CreateAttr("href", "Contents/history.xml")
	item.CreateAttr("media-type", "application/xml")
	manifest.AddChild(item)
	return true, nil
}

func newHistoryEntryElement(entryID int, spec HistoryEntrySpec) *etree.Element {
	entry := etree.NewElement("hh:historyEntry")
	entry.CreateAttr("id", strconv.Itoa(entryID))
	entry.CreateAttr("command", strings.TrimSpace(spec.Command))
	entry.CreateAttr("author", strings.TrimSpace(spec.Author))
	entry.CreateAttr("createdAt", spec.Timestamp.UTC().Format(time.RFC3339))

	summary := entry.CreateElement("hh:summary")
	summary.SetText(spec.Summary)
	return entry
}

func nextHistoryEntryID(root *etree.Element) int {
	maxID := 0
	for _, child := range childElementsByTag(root, "hh:historyEntry") {
		value, err := strconv.Atoi(child.SelectAttrValue("id", "0"))
		if err != nil {
			continue
		}
		if value > maxID {
			maxID = value
		}
	}
	return maxID + 1
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

func clearSubListParagraphs(subList *etree.Element) {
	for _, child := range append([]*etree.Element{}, subList.ChildElements()...) {
		if tagMatches(child.Tag, "hp:p") {
			subList.RemoveChild(child)
		}
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

func setElementAttr(root *etree.Element, key, value string) {
	if root == nil {
		return
	}
	if attr := root.SelectAttr(key); attr != nil {
		attr.Value = value
		return
	}
	root.CreateAttr(key, value)
}

func mapKeysSorted(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, leftErr := strconv.Atoi(keys[i])
		right, rightErr := strconv.Atoi(keys[j])
		if leftErr == nil && rightErr == nil {
			return left < right
		}
		return keys[i] < keys[j]
	})
	return keys
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
	return newTableParagraphElementWithWidth(counter, spec, defaultTableWidth)
}

func newTableParagraphElementWithWidth(counter *idCounter, spec TableSpec, tableWidth int) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")
	run.AddChild(newTableElement(counter, spec, tableWidth))
	return paragraph
}

func newTableElement(counter *idCounter, spec TableSpec, tableWidth int) *etree.Element {
	if tableWidth <= 0 {
		tableWidth = defaultTableWidth
	}
	table := etree.NewElement("hp:tbl")
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
	size.CreateAttr("width", strconv.Itoa(tableWidth))
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

	baseWidth := tableWidth / spec.Cols
	remainder := tableWidth % spec.Cols

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
	return table
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

func newLineParagraphElement(counter *idCounter, shapeID string, width, height int, lineColor string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")

	line := run.CreateElement("hp:line")
	line.CreateAttr("id", shapeID)
	line.CreateAttr("zOrder", "0")
	line.CreateAttr("numberingType", "NONE")
	line.CreateAttr("lock", "0")
	line.CreateAttr("dropcapstyle", "None")
	line.CreateAttr("href", "")
	line.CreateAttr("groupLevel", "0")
	line.CreateAttr("instid", shapeID)
	line.CreateAttr("ratio", "0")

	offset := line.CreateElement("hp:offset")
	offset.CreateAttr("x", "0")
	offset.CreateAttr("y", "0")

	orgSize := line.CreateElement("hp:orgSz")
	orgSize.CreateAttr("width", strconv.Itoa(width))
	orgSize.CreateAttr("height", strconv.Itoa(height))

	currentSize := line.CreateElement("hp:curSz")
	currentSize.CreateAttr("width", strconv.Itoa(width))
	currentSize.CreateAttr("height", strconv.Itoa(height))

	flip := line.CreateElement("hp:flip")
	flip.CreateAttr("horizontal", "0")
	flip.CreateAttr("vertical", "0")

	rotation := line.CreateElement("hp:rotationInfo")
	rotation.CreateAttr("angle", "0")
	rotation.CreateAttr("centerX", strconv.Itoa(width/2))
	rotation.CreateAttr("centerY", strconv.Itoa(height/2))
	rotation.CreateAttr("rotateimage", "1")

	renderingInfo := line.CreateElement("hp:renderingInfo")
	renderingInfo.AddChild(newMatrixElement("hc:transMatrix"))
	renderingInfo.AddChild(newMatrixElement("hc:scaMatrix"))
	renderingInfo.AddChild(newMatrixElement("hc:rotMatrix"))

	lineShape := line.CreateElement("hp:lineShape")
	lineShape.CreateAttr("color", lineColor)
	lineShape.CreateAttr("width", "283")
	lineShape.CreateAttr("style", "SOLID")
	lineShape.CreateAttr("endCap", "FLAT")
	lineShape.CreateAttr("headStyle", "NORMAL")
	lineShape.CreateAttr("tailStyle", "NORMAL")
	lineShape.CreateAttr("outlineStyle", "NORMAL")

	shadow := line.CreateElement("hp:shadow")
	shadow.CreateAttr("type", "NONE")
	shadow.CreateAttr("color", "#B2B2B2")
	shadow.CreateAttr("offsetX", "0")
	shadow.CreateAttr("offsetY", "0")
	shadow.CreateAttr("alpha", "0")

	startPoint := line.CreateElement("hc:startPt")
	startPoint.CreateAttr("x", "0")
	startPoint.CreateAttr("y", "0")

	endPoint := line.CreateElement("hc:endPt")
	endPoint.CreateAttr("x", strconv.Itoa(width))
	endPoint.CreateAttr("y", strconv.Itoa(height))

	size := line.CreateElement("hp:sz")
	size.CreateAttr("width", strconv.Itoa(width))
	size.CreateAttr("widthRelTo", "ABSOLUTE")
	size.CreateAttr("height", strconv.Itoa(height))
	size.CreateAttr("heightRelTo", "ABSOLUTE")
	size.CreateAttr("protect", "0")

	position := line.CreateElement("hp:pos")
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

	line.AddChild(newMarginElement("hp:outMargin"))

	run.CreateElement("hp:t")
	paragraph.AddChild(newPictureLineSegElement(width, height))
	return paragraph
}

func newEllipseParagraphElement(counter *idCounter, shapeID string, width, height int, lineColor, fillColor string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	paragraph.CreateAttr("id", counter.Next())
	paragraph.CreateAttr("paraPrIDRef", "0")
	paragraph.CreateAttr("styleIDRef", "0")
	paragraph.CreateAttr("pageBreak", "0")
	paragraph.CreateAttr("columnBreak", "0")
	paragraph.CreateAttr("merged", "0")

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")

	ellipse := run.CreateElement("hp:ellipse")
	ellipse.CreateAttr("id", shapeID)
	ellipse.CreateAttr("zOrder", "0")
	ellipse.CreateAttr("numberingType", "NONE")
	ellipse.CreateAttr("lock", "0")
	ellipse.CreateAttr("dropcapstyle", "None")
	ellipse.CreateAttr("href", "")
	ellipse.CreateAttr("groupLevel", "0")
	ellipse.CreateAttr("instid", shapeID)
	ellipse.CreateAttr("intervalDirty", "false")
	ellipse.CreateAttr("hasArcPr", "false")
	ellipse.CreateAttr("arcType", "Normal")

	offset := ellipse.CreateElement("hp:offset")
	offset.CreateAttr("x", "0")
	offset.CreateAttr("y", "0")

	orgSize := ellipse.CreateElement("hp:orgSz")
	orgSize.CreateAttr("width", strconv.Itoa(width))
	orgSize.CreateAttr("height", strconv.Itoa(height))

	currentSize := ellipse.CreateElement("hp:curSz")
	currentSize.CreateAttr("width", strconv.Itoa(width))
	currentSize.CreateAttr("height", strconv.Itoa(height))

	flip := ellipse.CreateElement("hp:flip")
	flip.CreateAttr("horizontal", "0")
	flip.CreateAttr("vertical", "0")

	rotation := ellipse.CreateElement("hp:rotationInfo")
	rotation.CreateAttr("angle", "0")
	rotation.CreateAttr("centerX", strconv.Itoa(width/2))
	rotation.CreateAttr("centerY", strconv.Itoa(height/2))
	rotation.CreateAttr("rotateimage", "1")

	renderingInfo := ellipse.CreateElement("hp:renderingInfo")
	renderingInfo.AddChild(newMatrixElement("hc:transMatrix"))
	renderingInfo.AddChild(newMatrixElement("hc:scaMatrix"))
	renderingInfo.AddChild(newMatrixElement("hc:rotMatrix"))

	lineShape := ellipse.CreateElement("hp:lineShape")
	lineShape.CreateAttr("color", lineColor)
	lineShape.CreateAttr("width", "283")
	lineShape.CreateAttr("style", "SOLID")
	lineShape.CreateAttr("endCap", "FLAT")
	lineShape.CreateAttr("headStyle", "NORMAL")
	lineShape.CreateAttr("tailStyle", "NORMAL")
	lineShape.CreateAttr("outlineStyle", "NORMAL")

	fillBrush := ellipse.CreateElement("hp:fillBrush")
	winBrush := fillBrush.CreateElement("hc:winBrush")
	winBrush.CreateAttr("faceColor", fillColor)
	winBrush.CreateAttr("hatchColor", "#FFFFFF")

	shadow := ellipse.CreateElement("hp:shadow")
	shadow.CreateAttr("type", "NONE")
	shadow.CreateAttr("color", "#B2B2B2")
	shadow.CreateAttr("offsetX", "0")
	shadow.CreateAttr("offsetY", "0")
	shadow.CreateAttr("alpha", "0")

	appendPoint(ellipse, "hc:center", width/2, height/2)
	appendPoint(ellipse, "hc:ax1", width, height/2)
	appendPoint(ellipse, "hc:ax2", width/2, height)
	appendPoint(ellipse, "hc:start1", width, height/2)
	appendPoint(ellipse, "hc:end1", width, height/2)
	appendPoint(ellipse, "hc:start2", width/2, height)
	appendPoint(ellipse, "hc:end2", width/2, height)

	size := ellipse.CreateElement("hp:sz")
	size.CreateAttr("width", strconv.Itoa(width))
	size.CreateAttr("widthRelTo", "ABSOLUTE")
	size.CreateAttr("height", strconv.Itoa(height))
	size.CreateAttr("heightRelTo", "ABSOLUTE")
	size.CreateAttr("protect", "0")

	position := ellipse.CreateElement("hp:pos")
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

	ellipse.AddChild(newMarginElement("hp:outMargin"))

	run.CreateElement("hp:t")
	paragraph.AddChild(newPictureLineSegElement(width, height))
	return paragraph
}

func newTextBoxParagraphElement(counter *idCounter, shapeID string, width, height int, lineColor, fillColor string, texts []string) *etree.Element {
	paragraph := newRectangleParagraphElement(counter, shapeID, width, height, lineColor, fillColor)

	run := firstChildByTag(paragraph, "hp:run")
	if run == nil {
		return paragraph
	}
	rect := firstChildByTag(run, "hp:rect")
	if rect == nil {
		return paragraph
	}

	const textMargin = 283
	textWidth := maxInt(width-(textMargin*2), 0)
	textHeight := maxInt(height-(textMargin*2), 0)

	drawText := rect.CreateElement("hp:drawText")
	drawText.CreateAttr("lastWidth", strconv.Itoa(textWidth))
	drawText.CreateAttr("name", "")
	drawText.CreateAttr("editable", "0")

	textMarginElement := drawText.CreateElement("hp:textMargin")
	textMarginElement.CreateAttr("left", strconv.Itoa(textMargin))
	textMarginElement.CreateAttr("right", strconv.Itoa(textMargin))
	textMarginElement.CreateAttr("top", strconv.Itoa(textMargin))
	textMarginElement.CreateAttr("bottom", strconv.Itoa(textMargin))

	subList := drawText.CreateElement("hp:subList")
	subList.CreateAttr("id", "")
	subList.CreateAttr("textDirection", "HORIZONTAL")
	subList.CreateAttr("lineWrap", "BREAK")
	subList.CreateAttr("vertAlign", "TOP")
	subList.CreateAttr("linkListIDRef", "0")
	subList.CreateAttr("linkListNextIDRef", "0")
	subList.CreateAttr("textWidth", strconv.Itoa(textWidth))
	subList.CreateAttr("textHeight", strconv.Itoa(textHeight))
	subList.CreateAttr("hasTextRef", "0")
	subList.CreateAttr("hasNumRef", "0")

	for _, text := range texts {
		subList.AddChild(newTextBoxInnerParagraphElement(counter, text, textWidth))
	}

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

func newTextBoxInnerParagraphElement(counter *idCounter, text string, width int) *etree.Element {
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

	paragraph.AddChild(newShapeTextLineSegElement(width))
	return paragraph
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

func newShapeTextLineSegElement(width int) *etree.Element {
	lineSegArray := etree.NewElement("hp:linesegarray")
	lineSeg := lineSegArray.CreateElement("hp:lineseg")
	lineSeg.CreateAttr("textpos", "0")
	lineSeg.CreateAttr("vertpos", "0")
	lineSeg.CreateAttr("vertsize", "1200")
	lineSeg.CreateAttr("textheight", "1200")
	lineSeg.CreateAttr("baseline", "1020")
	lineSeg.CreateAttr("spacing", "720")
	lineSeg.CreateAttr("horzpos", "0")
	lineSeg.CreateAttr("horzsize", strconv.Itoa(maxInt(width, 1200)))
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
