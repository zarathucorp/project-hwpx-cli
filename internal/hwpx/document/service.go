package document

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func CreateEditableDocument(outputDir string) (shared.Report, error) {
	return shared.CreateEditableDocument(outputDir)
}
