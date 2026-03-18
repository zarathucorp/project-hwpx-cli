package paragraph

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func Add(targetDir string, texts []string) (shared.Report, int, error) {
	return shared.AddParagraphs(targetDir, texts)
}

func SetText(targetDir string, paragraphIndex int, text string) (shared.Report, string, error) {
	return shared.SetParagraphText(targetDir, paragraphIndex, text)
}

func ApplyTextStyle(targetDir string, paragraphIndex int, runIndex *int, spec shared.TextStyleSpec) (shared.Report, []string, int, error) {
	return shared.ApplyTextStyle(targetDir, paragraphIndex, runIndex, spec)
}

func Delete(targetDir string, paragraphIndex int) (shared.Report, string, error) {
	return shared.DeleteParagraph(targetDir, paragraphIndex)
}
