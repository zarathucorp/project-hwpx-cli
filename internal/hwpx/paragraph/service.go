package paragraph

import "github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"

func Add(targetDir string, texts []string) (shared.Report, int, error) {
	return shared.AddParagraphs(targetDir, texts)
}

func AddRunText(targetDir string, paragraphIndex int, runIndex *int, text string) (shared.Report, int, string, error) {
	return shared.AddRunText(targetDir, paragraphIndex, runIndex, text)
}

func SetRunText(targetDir string, selector shared.SectionSelector, paragraphIndex, runIndex int, text string) (shared.Report, string, string, error) {
	return shared.SetRunText(targetDir, selector, paragraphIndex, runIndex, text)
}

func FindRunsByStyle(targetDir string, selector shared.SectionSelector, filter shared.RunStyleFilter) ([]shared.RunStyleMatch, error) {
	return shared.FindRunsByStyle(targetDir, selector, filter)
}

func ReplaceRunsByStyle(targetDir string, selector shared.SectionSelector, filter shared.RunStyleFilter, text string) (shared.Report, []shared.RunTextReplacement, error) {
	return shared.ReplaceRunsByStyle(targetDir, selector, filter, text)
}

func SetText(targetDir string, selector shared.SectionSelector, paragraphIndex int, text string) (shared.Report, string, error) {
	return shared.SetParagraphText(targetDir, selector, paragraphIndex, text)
}

func SetLayout(targetDir string, paragraphIndex int, spec shared.ParagraphLayoutSpec) (shared.Report, string, error) {
	return shared.SetParagraphLayout(targetDir, paragraphIndex, spec)
}

func SetList(targetDir string, paragraphIndex int, spec shared.ParagraphListSpec) (shared.Report, string, error) {
	return shared.SetParagraphList(targetDir, paragraphIndex, spec)
}

func ApplyTextStyle(targetDir string, paragraphIndex int, runIndex *int, spec shared.TextStyleSpec) (shared.Report, []string, int, error) {
	return shared.ApplyTextStyle(targetDir, paragraphIndex, runIndex, spec)
}

func Delete(targetDir string, paragraphIndex int) (shared.Report, string, error) {
	return shared.DeleteParagraph(targetDir, paragraphIndex)
}
