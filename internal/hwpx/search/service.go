package search

import "github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"

func FindByTag(targetDir string, selector shared.SectionSelector, filter shared.TagFilter) ([]shared.TagMatch, error) {
	return shared.FindByTag(targetDir, selector, filter)
}

func FindByAttr(targetDir string, selector shared.SectionSelector, filter shared.AttributeFilter) ([]shared.AttributeMatch, error) {
	return shared.FindByAttr(targetDir, selector, filter)
}

func FindByXPath(targetDir string, selector shared.SectionSelector, filter shared.XPathFilter) ([]shared.XPathMatch, error) {
	return shared.FindByXPath(targetDir, selector, filter)
}
