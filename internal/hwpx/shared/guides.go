package shared

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx/core"
)

func RemoveGuides(targetDir string, selector SectionSelector, reason string) (Report, []core.TemplateTextCandidate, error) {
	analysis, err := core.AnalyzeTemplate(targetDir)
	if err != nil {
		return Report{}, nil, err
	}

	candidates := filterGuideCandidates(analysis.Guides, selector, reason)
	if len(candidates) == 0 {
		report, err := Validate(targetDir)
		if err != nil {
			return Report{}, nil, err
		}
		return report, candidates, nil
	}

	targets, err := resolveSectionTargets(targetDir, selector)
	if err != nil {
		return Report{}, nil, err
	}

	targetByIndex := make(map[int]sectionTarget, len(targets))
	for _, target := range targets {
		targetByIndex[target.Index] = target
	}

	grouped := groupGuideCandidatesBySection(candidates)
	for _, sectionIndex := range sortedGuideSections(grouped) {
		target, ok := targetByIndex[sectionIndex]
		if !ok {
			return Report{}, nil, fmt.Errorf("section target not found: %d", sectionIndex)
		}

		paragraphs := findElementsByTag(target.Root, "hp:p")
		for _, candidate := range grouped[sectionIndex] {
			if candidate.ParagraphIndex < 0 || candidate.ParagraphIndex >= len(paragraphs) {
				return Report{}, nil, fmt.Errorf("guide paragraph index out of range: %d", candidate.ParagraphIndex)
			}

			paragraph := paragraphs[candidate.ParagraphIndex]
			if hasSectionProperty(paragraph) {
				continue
			}

			replaceParagraphText(paragraph, "")
		}

		if err := saveXML(target.Doc, filepath.Join(targetDir, filepath.FromSlash(target.Path))); err != nil {
			return Report{}, nil, err
		}
	}

	report, err := Validate(targetDir)
	if err != nil {
		return Report{}, nil, err
	}
	return report, candidates, nil
}

func filterGuideCandidates(candidates []core.TemplateTextCandidate, selector SectionSelector, reason string) []core.TemplateTextCandidate {
	filtered := make([]core.TemplateTextCandidate, 0, len(candidates))
	normalizedReason := strings.TrimSpace(reason)

	for _, candidate := range candidates {
		if selector.Section != nil && candidate.SectionIndex != *selector.Section {
			continue
		}
		if normalizedReason != "" && !strings.EqualFold(candidate.Reason, normalizedReason) {
			continue
		}
		filtered = append(filtered, candidate)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].SectionIndex != filtered[j].SectionIndex {
			return filtered[i].SectionIndex < filtered[j].SectionIndex
		}
		return filtered[i].ParagraphIndex < filtered[j].ParagraphIndex
	})
	return filtered
}

func groupGuideCandidatesBySection(candidates []core.TemplateTextCandidate) map[int][]core.TemplateTextCandidate {
	grouped := make(map[int][]core.TemplateTextCandidate)
	for _, candidate := range candidates {
		grouped[candidate.SectionIndex] = append(grouped[candidate.SectionIndex], candidate)
	}
	return grouped
}

func sortedGuideSections(grouped map[int][]core.TemplateTextCandidate) []int {
	sections := make([]int, 0, len(grouped))
	for sectionIndex := range grouped {
		sections = append(sections, sectionIndex)
	}
	sort.Ints(sections)
	return sections
}
