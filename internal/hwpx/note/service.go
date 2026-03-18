package note

import "github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"

func AddFootnote(targetDir string, spec shared.NoteSpec) (shared.Report, int, error) {
	return shared.AddFootnote(targetDir, spec)
}

func AddEndnote(targetDir string, spec shared.NoteSpec) (shared.Report, int, error) {
	return shared.AddEndnote(targetDir, spec)
}

func AddMemo(targetDir string, spec shared.MemoSpec) (shared.Report, string, string, int, error) {
	return shared.AddMemo(targetDir, spec)
}
