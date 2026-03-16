package layout

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func SetHeaderText(targetDir string, spec shared.HeaderFooterSpec) (shared.Report, error) {
	return shared.SetHeaderText(targetDir, spec)
}

func SetFooterText(targetDir string, spec shared.HeaderFooterSpec) (shared.Report, error) {
	return shared.SetFooterText(targetDir, spec)
}

func SetPageNumber(targetDir string, spec shared.PageNumberSpec) (shared.Report, error) {
	return shared.SetPageNumber(targetDir, spec)
}
