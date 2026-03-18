package document

import "github.com/zarathucop/project-hwpx-cli/internal/hwpx/shared"

func CreateEditableDocument(outputDir string) (shared.Report, error) {
	return shared.CreateEditableDocument(outputDir)
}
