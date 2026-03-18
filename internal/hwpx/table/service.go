package table

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func Add(targetDir string, spec shared.TableSpec) (shared.Report, int, error) {
	return shared.AddTable(targetDir, spec)
}

func AddNested(targetDir string, tableIndex, row, col int, spec shared.TableSpec) (shared.Report, error) {
	return shared.AddNestedTable(targetDir, tableIndex, row, col, spec)
}

func SetCell(targetDir string, tableIndex, row, col int, spec shared.TableCellStyleSpec) (shared.Report, error) {
	return shared.SetTableCell(targetDir, tableIndex, row, col, spec)
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
