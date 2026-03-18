package history

import "github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"

func Record(targetDir string, spec shared.HistoryEntrySpec) error {
	return shared.RecordHistory(targetDir, spec)
}
