package reference

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func AddBookmark(targetDir string, spec shared.BookmarkSpec) (shared.Report, error) {
	return shared.AddBookmark(targetDir, spec)
}

func AddHyperlink(targetDir string, spec shared.HyperlinkSpec) (shared.Report, string, error) {
	return shared.AddHyperlink(targetDir, spec)
}

func AddHeading(targetDir string, spec shared.HeadingSpec) (shared.Report, string, error) {
	return shared.AddHeading(targetDir, spec)
}

func InsertTOC(targetDir string, spec shared.TOCSpec) (shared.Report, int, error) {
	return shared.InsertTOC(targetDir, spec)
}

func AddCrossReference(targetDir string, spec shared.CrossReferenceSpec) (shared.Report, string, string, error) {
	return shared.AddCrossReference(targetDir, spec)
}
