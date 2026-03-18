package section

import "github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"

func Add(targetDir string) (shared.Report, int, string, error) {
	return shared.AddSection(targetDir)
}

func Delete(targetDir string, sectionIndex int) (shared.Report, string, error) {
	return shared.DeleteSection(targetDir, sectionIndex)
}
