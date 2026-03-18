package search

import "github.com/zarathucop/project-hwpx-cli/internal/hwpx/shared"

func FindByTag(targetDir string, filter shared.TagFilter) ([]shared.TagMatch, error) {
	return shared.FindByTag(targetDir, filter)
}

func FindByAttr(targetDir string, filter shared.AttributeFilter) ([]shared.AttributeMatch, error) {
	return shared.FindByAttr(targetDir, filter)
}

func FindByXPath(targetDir string, filter shared.XPathFilter) ([]shared.XPathMatch, error) {
	return shared.FindByXPath(targetDir, filter)
}
