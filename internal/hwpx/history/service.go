package history

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func Record(targetDir string, spec shared.HistoryEntrySpec) error {
	return shared.RecordHistory(targetDir, spec)
}
