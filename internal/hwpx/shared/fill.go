package shared

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/beevik/etree"
)

func PlanFillTemplate(targetDir string, selector SectionSelector, replacements []FillTemplateReplacement) ([]FillTemplateChange, []FillTemplateMiss, error) {
	targets, err := resolveSectionTargets(targetDir, selector)
	if err != nil {
		return nil, nil, err
	}
	changes, misses := planFillTemplateTargets(targets, selector, replacements)
	return changes, misses, nil
}

func FillTemplate(targetDir string, selector SectionSelector, replacements []FillTemplateReplacement) (Report, []FillTemplateChange, []FillTemplateMiss, error) {
	targets, err := resolveSectionTargets(targetDir, selector)
	if err != nil {
		return Report{}, nil, nil, err
	}

	changes, misses := planFillTemplateTargets(targets, selector, replacements)
	if len(changes) == 0 {
		report, err := Validate(targetDir)
		if err != nil {
			return Report{}, nil, nil, err
		}
		return report, changes, misses, nil
	}

	if err := applyFillTemplateChanges(targetDir, targets, changes); err != nil {
		return Report{}, nil, nil, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, nil, nil, err
	}
	return report, changes, misses, nil
}

func planFillTemplateTargets(targets []sectionTarget, selector SectionSelector, replacements []FillTemplateReplacement) ([]FillTemplateChange, []FillTemplateMiss) {
	var changes []FillTemplateChange
	var misses []FillTemplateMiss

	for _, replacement := range replacements {
		mode := normalizeFillMode(replacement)
		var planned []FillTemplateChange
		switch {
		case strings.TrimSpace(replacement.Placeholder) != "":
			planned = planPlaceholderChanges(targets, replacement, mode)
		case strings.TrimSpace(replacement.NearText) != "":
			planned = planNearTextChanges(targets, replacement, mode)
		case strings.TrimSpace(replacement.Anchor) != "":
			planned = planAnchorChanges(targets, replacement, mode)
		}
		changes = append(changes, planned...)
		if miss := buildFillTemplateMiss(targets, selector, replacement, mode, len(planned)); miss != nil {
			misses = append(misses, *miss)
		}
	}

	changes = dedupeFillTemplateChanges(changes)
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].SectionIndex != changes[j].SectionIndex {
			return changes[i].SectionIndex < changes[j].SectionIndex
		}
		if changes[i].TableIndex != nil && changes[j].TableIndex != nil && *changes[i].TableIndex != *changes[j].TableIndex {
			return *changes[i].TableIndex < *changes[j].TableIndex
		}
		leftParagraph := -1
		rightParagraph := -1
		if changes[i].ParagraphIndex != nil {
			leftParagraph = *changes[i].ParagraphIndex
		}
		if changes[j].ParagraphIndex != nil {
			rightParagraph = *changes[j].ParagraphIndex
		}
		return leftParagraph < rightParagraph
	})

	return changes, misses
}

func normalizeFillMode(replacement FillTemplateReplacement) string {
	mode := strings.ToLower(strings.TrimSpace(replacement.Mode))
	if mode != "" {
		return mode
	}
	if strings.TrimSpace(replacement.Placeholder) != "" {
		return "replace"
	}
	if strings.TrimSpace(replacement.NearText) != "" {
		if len(replacement.Values) > 0 {
			return "paragraph-next-repeat"
		}
		return "paragraph-next"
	}
	if len(replacement.Grid) > 0 {
		return "table-right-grid"
	}
	if len(replacement.Values) > 0 {
		return "table-down-repeat"
	}
	return "table-right"
}

func planPlaceholderChanges(targets []sectionTarget, replacement FillTemplateReplacement, mode string) []FillTemplateChange {
	selector := strings.TrimSpace(replacement.Placeholder)
	if selector == "" {
		return nil
	}
	matchMode := normalizeFillTemplateMatchMode(replacement.MatchMode)

	var changes []FillTemplateChange
	matchIndex := 0
	for _, target := range targets {
		paragraphs := findElementsByTag(target.Root, "hp:p")
		for paragraphIndex, paragraph := range paragraphs {
			text := paragraphPlainText(paragraph)
			if !fillTemplateTextMatches(text, selector, matchMode) {
				continue
			}
			if !fillTemplateOccurrenceMatches(replacement.Occurrence, matchIndex) {
				matchIndex++
				continue
			}
			matchIndex++

			updated := strings.ReplaceAll(text, selector, replacement.Value)
			if updated == text {
				continue
			}

			paragraphIndexCopy := paragraphIndex
			change := FillTemplateChange{
				Kind:           "placeholder",
				Mode:           mode,
				SectionIndex:   target.Index,
				SectionPath:    target.Path,
				ParagraphIndex: &paragraphIndexCopy,
				Selector:       selector,
				PreviousText:   text,
				Text:           updated,
			}
			if tableIndex, cell := locateParagraphTableContext(target.Root, paragraph); tableIndex != nil {
				change.TableIndex = tableIndex
				change.Cell = cell
			}
			changes = append(changes, change)
			if replacement.Occurrence != nil {
				return changes
			}
		}
	}
	return changes
}

func planAnchorChanges(targets []sectionTarget, replacement FillTemplateReplacement, mode string) []FillTemplateChange {
	if mode != "table-right" &&
		mode != "table-down" &&
		mode != "table-left" &&
		mode != "table-up" &&
		mode != "table-right-grid" &&
		mode != "table-down-grid" &&
		mode != "table-right-repeat" &&
		mode != "table-down-repeat" &&
		mode != "table-left-repeat" &&
		mode != "table-up-repeat" {
		return nil
	}

	anchor := strings.TrimSpace(replacement.Anchor)
	if anchor == "" {
		return nil
	}
	tableLabelSelector := strings.TrimSpace(replacement.TableLabel)
	tableIndexSelector := replacement.TableIndex
	matchMode := normalizeFillTemplateMatchMode(replacement.MatchMode)

	var changes []FillTemplateChange
	matchIndex := 0
	for _, target := range targets {
		tables := findElementsByTag(target.Root, "hp:tbl")
		tableLabels := deriveFillTemplateTableLabels(target.Root)
		for tableIndex, table := range tables {
			if tableIndexSelector != nil && tableIndex != *tableIndexSelector {
				continue
			}
			tableLabel := strings.TrimSpace(tableLabels[table])
			if tableLabelSelector != "" && !fillTemplateTableLabelMatches(tableLabel, tableLabelSelector, matchMode) {
				continue
			}
			repeatPlanned := false
			for _, rowElement := range childElementsByTag(table, "hp:tr") {
				for _, cell := range childElementsByTag(rowElement, "hp:tc") {
					cellText := strings.TrimSpace(paragraphPlainText(cell))
					if !fillTemplateTextMatches(cellText, anchor, matchMode) {
						continue
					}
					if !fillTemplateOccurrenceMatches(replacement.Occurrence, matchIndex) {
						matchIndex++
						continue
					}
					matchIndex++

					row, col := tableCellAddress(cell)
					spanRow, spanCol := tableCellSpan(cell)
					if spanRow <= 0 {
						spanRow = 1
					}
					if spanCol <= 0 {
						spanCol = 1
					}
					targetRow := row
					targetCol := col
					switch mode {
					case "table-right", "table-right-repeat":
						targetCol = col + spanCol
					case "table-down", "table-down-repeat":
						targetRow = row + spanRow
					case "table-right-grid":
						targetCol = col + spanCol
					case "table-down-grid":
						targetRow = row + spanRow
					case "table-left", "table-left-repeat":
						targetCol = col - 1
					case "table-up", "table-up-repeat":
						targetRow = row - 1
					}
					targetEntry, err := tableCellEntry(table, targetRow, targetCol)
					if err != nil {
						continue
					}

					tableIndexCopy := tableIndex
					if isGridMode(mode) {
						changes = append(changes, planGridAnchorChanges(target, table, tableIndexCopy, tableLabel, targetEntry, replacement, mode)...)
						if replacement.Occurrence != nil {
							return changes
						}
						repeatPlanned = true
						break
					}
					if isRepeatMode(mode) {
						changes = append(changes, planRepeatedAnchorChanges(target, table, tableIndexCopy, tableLabel, targetEntry, replacement, mode)...)
						repeatPlanned = true
						break
					}
					change := FillTemplateChange{
						Kind:         "anchor",
						Mode:         mode,
						SectionIndex: target.Index,
						SectionPath:  target.Path,
						TableIndex:   &tableIndexCopy,
						Cell: &TableCellCoordinate{
							Row: targetEntry.anchor[0],
							Col: targetEntry.anchor[1],
						},
						TableLabel:   tableLabel,
						Selector:     anchor,
						PreviousText: strings.TrimSpace(paragraphPlainText(targetEntry.cell)),
						Text:         replacement.Value,
					}
					changes = append(changes, change)
					if replacement.Occurrence != nil {
						return changes
					}
				}
				if repeatPlanned {
					break
				}
			}
			if repeatPlanned {
				continue
			}
		}
	}
	return changes
}

func planGridAnchorChanges(target sectionTarget, table *etree.Element, tableIndex int, tableLabel string, firstTarget tableGridEntry, replacement FillTemplateReplacement, mode string) []FillTemplateChange {
	if len(replacement.Grid) == 0 {
		return nil
	}

	var changes []FillTemplateChange
	seenAnchors := map[[2]int]struct{}{}
	for rowOffset, values := range replacement.Grid {
		for colOffset, value := range values {
			targetRow := firstTarget.anchor[0] + rowOffset
			targetCol := firstTarget.anchor[1] + colOffset
			entry, err := tableCellEntry(table, targetRow, targetCol)
			if err != nil {
				return changes
			}
			if _, ok := seenAnchors[entry.anchor]; ok {
				continue
			}
			seenAnchors[entry.anchor] = struct{}{}

			tableIndexCopy := tableIndex
			changes = append(changes, FillTemplateChange{
				Kind:         "anchor",
				Mode:         mode,
				SectionIndex: target.Index,
				SectionPath:  target.Path,
				TableIndex:   &tableIndexCopy,
				Cell: &TableCellCoordinate{
					Row: entry.anchor[0],
					Col: entry.anchor[1],
				},
				TableLabel:   tableLabel,
				Selector:     strings.TrimSpace(replacement.Anchor),
				PreviousText: strings.TrimSpace(paragraphPlainText(entry.cell)),
				Text:         value,
			})
		}
	}
	return changes
}

func planRepeatedAnchorChanges(target sectionTarget, table *etree.Element, tableIndex int, tableLabel string, firstTarget tableGridEntry, replacement FillTemplateReplacement, mode string) []FillTemplateChange {
	values := replacement.Values
	if len(values) == 0 {
		return nil
	}

	var changes []FillTemplateChange
	seenAnchors := map[[2]int]struct{}{}
	valueIndex := 0
	for offset := 0; valueIndex < len(values); offset++ {
		targetRow := firstTarget.anchor[0]
		targetCol := firstTarget.anchor[1]
		switch mode {
		case "table-down-repeat":
			targetRow = firstTarget.anchor[0] + offset
		case "table-right-repeat":
			targetCol = firstTarget.anchor[1] + offset
		case "table-up-repeat":
			targetRow = firstTarget.anchor[0] - offset
		case "table-left-repeat":
			targetCol = firstTarget.anchor[1] - offset
		}

		entry, err := tableCellEntry(table, targetRow, targetCol)
		if err != nil {
			break
		}
		if _, ok := seenAnchors[entry.anchor]; ok {
			continue
		}
		seenAnchors[entry.anchor] = struct{}{}

		tableIndexCopy := tableIndex
		changes = append(changes, FillTemplateChange{
			Kind:         "anchor",
			Mode:         mode,
			SectionIndex: target.Index,
			SectionPath:  target.Path,
			TableIndex:   &tableIndexCopy,
			Cell: &TableCellCoordinate{
				Row: entry.anchor[0],
				Col: entry.anchor[1],
			},
			TableLabel:   tableLabel,
			Selector:     strings.TrimSpace(replacement.Anchor),
			PreviousText: strings.TrimSpace(paragraphPlainText(entry.cell)),
			Text:         values[valueIndex],
		})
		valueIndex++
	}
	return changes
}

func planNearTextChanges(targets []sectionTarget, replacement FillTemplateReplacement, mode string) []FillTemplateChange {
	selector := strings.TrimSpace(replacement.NearText)
	if selector == "" {
		return nil
	}
	matchMode := normalizeFillTemplateMatchMode(replacement.MatchMode)

	var changes []FillTemplateChange
	matchIndex := 0
	for _, target := range targets {
		paragraphs := findElementsByTag(target.Root, "hp:p")
		for paragraphIndex, paragraph := range paragraphs {
			text := strings.TrimSpace(paragraphPlainText(paragraph))
			if text == "" || !fillTemplateTextMatches(text, selector, matchMode) {
				continue
			}
			if !fillTemplateOccurrenceMatches(replacement.Occurrence, matchIndex) {
				matchIndex++
				continue
			}
			matchIndex++

			switch mode {
			case "paragraph-replace":
				if change := planParagraphReplaceChange(target, paragraphs, paragraphIndex, selector, replacement.Value, "near-text", mode); change != nil {
					changes = append(changes, *change)
				}
			case "paragraph-next":
				nextIndex := nextFillableParagraphIndex(target.Root, paragraphs, paragraphIndex)
				if nextIndex < 0 {
					continue
				}
				if change := planParagraphReplaceChange(target, paragraphs, nextIndex, selector, replacement.Value, "near-text", mode); change != nil {
					changes = append(changes, *change)
				}
			case "paragraph-next-repeat":
				changes = append(changes, planRepeatedParagraphChanges(target, paragraphs, nextFillableParagraphIndex(target.Root, paragraphs, paragraphIndex), selector, replacement.Values, mode)...)
			case "paragraph-replace-repeat":
				changes = append(changes, planRepeatedParagraphChanges(target, paragraphs, paragraphIndex, selector, replacement.Values, mode)...)
			}
			if replacement.Occurrence != nil {
				return changes
			}
		}
	}
	return changes
}

func planRepeatedParagraphChanges(target sectionTarget, paragraphs []*etree.Element, startIndex int, selector string, values []string, mode string) []FillTemplateChange {
	if startIndex < 0 || len(values) == 0 {
		return nil
	}

	var changes []FillTemplateChange
	currentIndex := startIndex
	for valueIndex := 0; valueIndex < len(values) && currentIndex >= 0; valueIndex++ {
		if change := planParagraphReplaceChange(target, paragraphs, currentIndex, selector, values[valueIndex], "near-text", mode); change != nil {
			changes = append(changes, *change)
		}
		currentIndex = nextFillableParagraphIndex(target.Root, paragraphs, currentIndex)
	}
	return changes
}

func dedupeFillTemplateChanges(changes []FillTemplateChange) []FillTemplateChange {
	result := make([]FillTemplateChange, 0, len(changes))
	seen := make(map[string]int, len(changes))
	for _, change := range changes {
		key := fillTemplateChangeKey(change)
		if index, ok := seen[key]; ok {
			result[index] = change
			continue
		}
		seen[key] = len(result)
		result = append(result, change)
	}
	return result
}

func fillTemplateChangeKey(change FillTemplateChange) string {
	var builder strings.Builder
	builder.WriteString(change.SectionPath)
	builder.WriteString("|")
	if change.ParagraphIndex != nil {
		builder.WriteString(fmt.Sprintf("p:%d", *change.ParagraphIndex))
	}
	builder.WriteString("|")
	if change.TableIndex != nil {
		builder.WriteString(fmt.Sprintf("t:%d", *change.TableIndex))
	}
	builder.WriteString("|")
	if change.Cell != nil {
		builder.WriteString(fmt.Sprintf("c:%d,%d", change.Cell.Row, change.Cell.Col))
	}
	return builder.String()
}

func applyFillTemplateChanges(targetDir string, targets []sectionTarget, changes []FillTemplateChange) error {
	targetBySection := make(map[int]sectionTarget, len(targets))
	for _, target := range targets {
		targetBySection[target.Index] = target
	}

	changedSections := make(map[int]struct{})
	for _, change := range changes {
		target, ok := targetBySection[change.SectionIndex]
		if !ok {
			return fmt.Errorf("section target not found: %d", change.SectionIndex)
		}

		switch change.Kind {
		case "placeholder":
			fallthrough
		case "near-text":
			if change.ParagraphIndex == nil {
				continue
			}
			paragraphs := findElementsByTag(target.Root, "hp:p")
			if *change.ParagraphIndex < 0 || *change.ParagraphIndex >= len(paragraphs) {
				return fmt.Errorf("paragraph index out of range: %d", *change.ParagraphIndex)
			}
			replaceParagraphText(paragraphs[*change.ParagraphIndex], change.Text)
		case "anchor":
			if change.TableIndex == nil || change.Cell == nil {
				continue
			}
			tables := findElementsByTag(target.Root, "hp:tbl")
			if *change.TableIndex < 0 || *change.TableIndex >= len(tables) {
				return fmt.Errorf("table index out of range: %d", *change.TableIndex)
			}
			entry, err := tableCellEntry(tables[*change.TableIndex], change.Cell.Row, change.Cell.Col)
			if err != nil {
				return err
			}
			if err := writeTableCellText(target.Root, entry.cell, change.Text); err != nil {
				return err
			}
		}

		changedSections[change.SectionIndex] = struct{}{}
		targetBySection[change.SectionIndex] = target
	}

	for _, target := range targets {
		if _, ok := changedSections[target.Index]; !ok {
			continue
		}
		if err := saveXML(target.Doc, filepath.Join(targetDir, filepath.FromSlash(target.Path))); err != nil {
			return err
		}
	}
	return nil
}

func planParagraphReplaceChange(target sectionTarget, paragraphs []*etree.Element, paragraphIndex int, selector, value, kind, mode string) *FillTemplateChange {
	if paragraphIndex < 0 || paragraphIndex >= len(paragraphs) {
		return nil
	}

	paragraph := paragraphs[paragraphIndex]
	previousText := paragraphPlainText(paragraph)
	if previousText == value {
		return nil
	}

	paragraphIndexCopy := paragraphIndex
	change := &FillTemplateChange{
		Kind:           kind,
		Mode:           mode,
		SectionIndex:   target.Index,
		SectionPath:    target.Path,
		ParagraphIndex: &paragraphIndexCopy,
		Selector:       selector,
		PreviousText:   previousText,
		Text:           value,
	}
	if tableIndex, cell := locateParagraphTableContext(target.Root, paragraph); tableIndex != nil {
		change.TableIndex = tableIndex
		change.Cell = cell
	}
	return change
}

func nextFillableParagraphIndex(root *etree.Element, paragraphs []*etree.Element, startIndex int) int {
	for index := startIndex + 1; index < len(paragraphs); index++ {
		paragraph := paragraphs[index]
		if hasSectionProperty(paragraph) {
			continue
		}
		if tableIndex, _ := locateParagraphTableContext(root, paragraph); tableIndex != nil {
			continue
		}
		return index
	}
	return -1
}

func isRepeatMode(mode string) bool {
	return mode == "table-down-repeat" ||
		mode == "table-right-repeat" ||
		mode == "table-up-repeat" ||
		mode == "table-left-repeat"
}

func isGridMode(mode string) bool {
	return mode == "table-right-grid" || mode == "table-down-grid"
}

func fillTemplateTableLabelMatches(actual, expected, matchMode string) bool {
	return fillTemplateTextMatches(actual, expected, matchMode)
}

func buildFillTemplateMiss(targets []sectionTarget, selector SectionSelector, replacement FillTemplateReplacement, mode string, matchedCount int) *FillTemplateMiss {
	requested := 1
	if len(replacement.Values) > 0 {
		requested = len(replacement.Values)
	}
	if len(replacement.Grid) > 0 {
		requested = fillTemplateGridSize(replacement.Grid)
	}
	sectionScoped := selector.Section != nil || !selector.AllSections

	kind := "anchor"
	selectorText := strings.TrimSpace(replacement.Anchor)
	reason := "anchor-not-found"
	switch {
	case strings.TrimSpace(replacement.Placeholder) != "":
		kind = "placeholder"
		selectorText = strings.TrimSpace(replacement.Placeholder)
		reason = "placeholder-not-found"
	case strings.TrimSpace(replacement.NearText) != "":
		kind = "near-text"
		selectorText = strings.TrimSpace(replacement.NearText)
		reason = "near-text-not-found"
	}

	if matchedCount == 0 {
		if kind == "anchor" {
			switch {
			case replacement.Occurrence != nil:
				reason = "occurrence-not-found"
			case replacement.TableIndex != nil && !fillTemplateHasTableIndex(targets, *replacement.TableIndex):
				reason = "table-index-not-found"
			case strings.TrimSpace(replacement.TableLabel) != "" && !fillTemplateHasMatchingTableLabel(targets, replacement.TableLabel):
				reason = "table-label-not-found"
			}
		} else if replacement.Occurrence != nil {
			reason = "occurrence-not-found"
		}
		return &FillTemplateMiss{
			Kind:          kind,
			Mode:          mode,
			Selector:      selectorText,
			TableLabel:    strings.TrimSpace(replacement.TableLabel),
			TableIndex:    replacement.TableIndex,
			Occurrence:    replacement.Occurrence,
			Reason:        reason,
			Requested:     requested,
			Matched:       0,
			Partial:       false,
			SectionScoped: sectionScoped,
		}
	}
	if matchedCount < requested {
		return &FillTemplateMiss{
			Kind:          kind,
			Mode:          mode,
			Selector:      selectorText,
			TableLabel:    strings.TrimSpace(replacement.TableLabel),
			TableIndex:    replacement.TableIndex,
			Occurrence:    replacement.Occurrence,
			Reason:        "insufficient-target-capacity",
			Requested:     requested,
			Matched:       matchedCount,
			Partial:       true,
			SectionScoped: sectionScoped,
		}
	}
	return nil
}

func fillTemplateOccurrenceMatches(occurrence *int, matchIndex int) bool {
	if occurrence == nil {
		return true
	}
	return *occurrence == matchIndex+1
}

func fillTemplateHasMatchingTableLabel(targets []sectionTarget, tableLabel string) bool {
	for _, target := range targets {
		for _, actual := range deriveFillTemplateTableLabels(target.Root) {
			if fillTemplateTableLabelMatches(actual, tableLabel, "contains") {
				return true
			}
		}
	}
	return false
}

func normalizeFillTemplateMatchMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "exact":
		return "exact"
	default:
		return "contains"
	}
}

func fillTemplateTextMatches(actual, expected, matchMode string) bool {
	actual = strings.TrimSpace(strings.ToLower(actual))
	expected = strings.TrimSpace(strings.ToLower(expected))
	if actual == "" || expected == "" {
		return false
	}
	if matchMode == "exact" {
		return actual == expected
	}
	return strings.Contains(actual, expected)
}

func fillTemplateHasTableIndex(targets []sectionTarget, tableIndex int) bool {
	if tableIndex < 0 {
		return false
	}
	for _, target := range targets {
		if tableIndex < len(findElementsByTag(target.Root, "hp:tbl")) {
			return true
		}
	}
	return false
}

func fillTemplateGridSize(grid [][]string) int {
	total := 0
	for _, row := range grid {
		total += len(row)
	}
	return total
}

func deriveFillTemplateTableLabels(root *etree.Element) map[*etree.Element]string {
	labels := map[*etree.Element]string{}
	lastText := ""

	for _, paragraph := range findElementsByTag(root, "hp:p") {
		if hasSectionProperty(paragraph) {
			continue
		}

		directText := strings.TrimSpace(fillTemplateParagraphDirectText(paragraph))
		labelText := directText
		if labelText == "" {
			labelText = lastText
		}

		for _, table := range fillTemplateParagraphTables(paragraph) {
			if strings.TrimSpace(labelText) == "" {
				continue
			}
			labels[table] = labelText
		}

		if directText != "" {
			lastText = directText
		}
	}

	return labels
}

func fillTemplateParagraphDirectText(paragraph *etree.Element) string {
	var builder strings.Builder
	for _, run := range childElementsByTag(paragraph, "hp:run") {
		fillTemplateAppendInlineText(&builder, run)
	}
	return strings.TrimSpace(builder.String())
}

func fillTemplateAppendInlineText(builder *strings.Builder, element *etree.Element) {
	for _, child := range element.Child {
		switch current := child.(type) {
		case *etree.CharData:
			builder.WriteString(current.Data)
		case *etree.Element:
			if fillTemplateIsObjectElement(current) {
				continue
			}
			switch localTag(current.Tag) {
			case "lineBreak":
				builder.WriteByte('\n')
			case "tab":
				builder.WriteByte('\t')
			default:
				fillTemplateAppendInlineText(builder, current)
			}
		}
	}
}

func fillTemplateIsObjectElement(element *etree.Element) bool {
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
	case tagMatches(element.Tag, "hp:drawText"):
		return true
	default:
		return false
	}
}

func fillTemplateParagraphTables(paragraph *etree.Element) []*etree.Element {
	var tables []*etree.Element
	for _, run := range childElementsByTag(paragraph, "hp:run") {
		for _, child := range run.ChildElements() {
			if tagMatches(child.Tag, "hp:tbl") {
				tables = append(tables, child)
			}
		}
	}
	return tables
}

func writeTableCellText(root *etree.Element, cell *etree.Element, text string) error {
	subList := firstChildByTag(cell, "hp:subList")
	if subList == nil {
		return fmt.Errorf("table cell does not contain hp:subList")
	}

	templateParagraphs := childElementsByTag(subList, "hp:p")
	clearSubListParagraphs(subList)
	counter := newIDCounter(root)
	for _, paragraphText := range normalizeParagraphTexts(text) {
		if len(templateParagraphs) == 0 {
			subList.AddChild(newCellParagraphElement(counter, paragraphText))
			continue
		}
		subList.AddChild(newFillTemplateParagraphElement(counter, templateParagraphs[0], paragraphText))
	}
	return nil
}

func newFillTemplateParagraphElement(counter *idCounter, templateParagraph *etree.Element, text string) *etree.Element {
	paragraph := etree.NewElement("hp:p")
	copyParagraphAttrs(templateParagraph, paragraph)
	paragraph.RemoveAttr("id")
	paragraph.CreateAttr("id", counter.Next())

	run := paragraph.CreateElement("hp:run")
	if templateRun := firstChildByTag(templateParagraph, "hp:run"); templateRun != nil {
		copyCharAttr(templateRun, run)
	}
	if strings.TrimSpace(run.SelectAttrValue("charPrIDRef", "")) == "" {
		run.CreateAttr("charPrIDRef", firstRunCharPrIDRef(templateParagraph))
	}

	textElement := run.CreateElement("hp:t")
	if text != "" {
		textElement.SetText(text)
	}
	paragraph.AddChild(newHeaderFooterLineSegElement(text))
	return paragraph
}

func locateParagraphTableContext(root *etree.Element, paragraph *etree.Element) (*int, *TableCellCoordinate) {
	tables := findElementsByTag(root, "hp:tbl")
	tableIndexByElement := make(map[*etree.Element]int, len(tables))
	for index, table := range tables {
		tableIndexByElement[table] = index
	}

	for current := paragraph.Parent(); current != nil; current = current.Parent() {
		if tagMatches(current.Tag, "hp:tc") {
			row, col := tableCellAddress(current)
			cell := &TableCellCoordinate{Row: row, Col: col}
			for ancestor := current.Parent(); ancestor != nil; ancestor = ancestor.Parent() {
				if !tagMatches(ancestor.Tag, "hp:tbl") {
					continue
				}
				if index, ok := tableIndexByElement[ancestor]; ok {
					tableIndex := index
					return &tableIndex, cell
				}
			}
			return nil, cell
		}
		if tagMatches(current.Tag, "hp:tbl") {
			if index, ok := tableIndexByElement[current]; ok {
				tableIndex := index
				return &tableIndex, nil
			}
		}
	}

	return nil, nil
}
