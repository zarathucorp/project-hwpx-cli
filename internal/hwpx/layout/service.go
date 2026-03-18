package layout

import "github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"

func SetHeaderText(targetDir string, spec shared.HeaderFooterSpec) (shared.Report, error) {
	return shared.SetHeaderText(targetDir, spec)
}

func SetFooterText(targetDir string, spec shared.HeaderFooterSpec) (shared.Report, error) {
	return shared.SetFooterText(targetDir, spec)
}

func RemoveHeader(targetDir string) (shared.Report, error) {
	return shared.RemoveHeader(targetDir)
}

func RemoveFooter(targetDir string) (shared.Report, error) {
	return shared.RemoveFooter(targetDir)
}

func SetPageNumber(targetDir string, spec shared.PageNumberSpec) (shared.Report, error) {
	return shared.SetPageNumber(targetDir, spec)
}

func SetColumns(targetDir string, spec shared.ColumnSpec) (shared.Report, error) {
	return shared.SetColumns(targetDir, spec)
}

func SetPageLayout(targetDir string, spec shared.PageLayoutSpec) (shared.Report, error) {
	return shared.SetPageLayout(targetDir, spec)
}
