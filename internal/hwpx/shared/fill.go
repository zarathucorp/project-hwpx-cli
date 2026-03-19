package shared

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/beevik/etree"
)

func PlanFillTemplate(targetDir string, selector SectionSelector, replacements []FillTemplateReplacement) ([]FillTemplateChange, error) {
	targets, err := resolveSectionTargets(targetDir, selector)
	if err != nil {
		return nil, err
	}
	return planFillTemplateTargets(targets, replacements), nil
}

func FillTemplate(targetDir string, selector SectionSelector, replacements []FillTemplateReplacement) (Report, []FillTemplateChange, error) {
	targets, err := resolveSectionTargets(targetDir, selector)
	if err != nil {
		return Report{}, nil, err
	}

	changes := planFillTemplateTargets(targets, replacements)
	if len(changes) == 0 {
		report, err := Validate(targetDir)
		if err != nil {
			return Report{}, nil, err
		}
		return report, changes, nil
	}

	if err := applyFillTemplateChanges(targetDir, targets, changes); err != nil {
		return Report{}, nil, err
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, nil, err
	}
	return report, changes, nil
}

func planFillTemplateTargets(targets []sectionTarget, replacements []FillTemplateReplacement) []FillTemplateChange {
	var changes []FillTemplateChange

	for _, replacement := range replacements {
		mode := normalizeFillMode(replacement)
		switch {
		case strings.TrimSpace(replacement.Placeholder) != "":
			changes = append(changes, planPlaceholderChanges(targets, replacement, mode)...)
		case strings.TrimSpace(replacement.NearText) != "":
			changes = append(changes, planNearTextChanges(targets, replacement, mode)...)
		case strings.TrimSpace(replacement.Anchor) != "":
			changes = append(changes, planAnchorChanges(targets, replacement, mode)...)
		}
	}

	sort.Slice(changes, func(i, j int) bool {
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

	return dedupeFillTemplateChanges(changes)
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
		return "paragraph-next"
	}
	return "table-right"
}

func planPlaceholderChanges(targets []sectionTarget, replacement FillTemplateReplacement, mode string) []FillTemplateChange {
	selector := strings.TrimSpace(replacement.Placeholder)
	if selector == "" {
		return nil
	}

	var changes []FillTemplateChange
	for _, target := range targets {
		paragraphs := findElementsByTag(target.Root, "hp:p")
		for paragraphIndex, paragraph := range paragraphs {
			text := paragraphPlainText(paragraph)
			if !strings.Contains(text, selector) {
				continue
			}

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
		}
	}
	return changes
}

func planAnchorChanges(targets []sectionTarget, replacement FillTemplateReplacement, mode string) []FillTemplateChange {
	if mode != "table-right" {
		return nil
	}

	anchor := strings.TrimSpace(replacement.Anchor)
	if anchor == "" {
		return nil
	}

	var changes []FillTemplateChange
	for _, target := range targets {
		tables := findElementsByTag(target.Root, "hp:tbl")
		for tableIndex, table := range tables {
			for _, rowElement := range childElementsByTag(table, "hp:tr") {
				for _, cell := range childElementsByTag(rowElement, "hp:tc") {
					cellText := strings.TrimSpace(paragraphPlainText(cell))
					if !strings.Contains(cellText, anchor) {
						continue
					}

					row, col := tableCellAddress(cell)
					_, spanCol := tableCellSpan(cell)
					if spanCol <= 0 {
						spanCol = 1
					}
					targetEntry, err := tableCellEntry(table, row, col+spanCol)
					if err != nil {
						continue
					}

					tableIndexCopy := tableIndex
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
						Selector:     anchor,
						PreviousText: strings.TrimSpace(paragraphPlainText(targetEntry.cell)),
						Text:         replacement.Value,
					}
					changes = append(changes, change)
				}
			}
		}
	}
	return changes
}

func planNearTextChanges(targets []sectionTarget, replacement FillTemplateReplacement, mode string) []FillTemplateChange {
	selector := strings.TrimSpace(replacement.NearText)
	if selector == "" {
		return nil
	}

	var changes []FillTemplateChange
	for _, target := range targets {
		paragraphs := findElementsByTag(target.Root, "hp:p")
		for paragraphIndex, paragraph := range paragraphs {
			text := strings.TrimSpace(paragraphPlainText(paragraph))
			if text == "" || !strings.Contains(text, selector) {
				continue
			}

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
			}
		}
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
	builder.WriteString(change.Kind)
	builder.WriteString("|")
	builder.WriteString(change.Mode)
	builder.WriteString("|")
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

func writeTableCellText(root *etree.Element, cell *etree.Element, text string) error {
	subList := firstChildByTag(cell, "hp:subList")
	if subList == nil {
		return fmt.Errorf("table cell does not contain hp:subList")
	}

	clearSubListParagraphs(subList)
	counter := newIDCounter(root)
	for _, paragraphText := range normalizeParagraphTexts(text) {
		subList.AddChild(newCellParagraphElement(counter, paragraphText))
	}
	return nil
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
