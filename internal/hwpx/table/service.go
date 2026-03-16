package table

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func Add(targetDir string, spec shared.TableSpec) (shared.Report, int, error) {
	return shared.AddTable(targetDir, spec)
}

func SetCellText(targetDir string, tableIndex, row, col int, text string) (shared.Report, error) {
	return shared.SetTableCellText(targetDir, tableIndex, row, col, text)
}

func MergeCells(targetDir string, tableIndex, startRow, startCol, endRow, endCol int) (shared.Report, error) {
	return shared.MergeTableCells(targetDir, tableIndex, startRow, startCol, endRow, endCol)
}

func SplitCell(targetDir string, tableIndex, row, col int) (shared.Report, error) {
	return shared.SplitTableCell(targetDir, tableIndex, row, col)
}
