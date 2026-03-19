package table

import "github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"

func Add(targetDir string, selector shared.SectionSelector, spec shared.TableSpec) (shared.Report, int, error) {
	return shared.AddTable(targetDir, selector, spec)
}

func AddNested(targetDir string, selector shared.SectionSelector, tableIndex, row, col int, spec shared.TableSpec) (shared.Report, error) {
	return shared.AddNestedTable(targetDir, selector, tableIndex, row, col, spec)
}

func SetCell(targetDir string, selector shared.SectionSelector, tableIndex, row, col int, spec shared.TableCellStyleSpec) (shared.Report, error) {
	return shared.SetTableCell(targetDir, selector, tableIndex, row, col, spec)
}

func SetCellText(targetDir string, selector shared.SectionSelector, tableIndex, row, col int, text string) (shared.Report, error) {
	return shared.SetTableCellText(targetDir, selector, tableIndex, row, col, text)
}

func SetCellContent(targetDir string, selector shared.SectionSelector, tableIndex, row, col int, spec shared.TableCellTextSpec) (shared.Report, string, []string, int, error) {
	return shared.SetTableCellContent(targetDir, selector, tableIndex, row, col, spec)
}

func SetCellParagraphLayout(targetDir string, selector shared.SectionSelector, tableIndex, row, col, paragraphIndex int, spec shared.ParagraphLayoutSpec) (shared.Report, string, error) {
	return shared.SetTableCellParagraphLayout(targetDir, selector, tableIndex, row, col, paragraphIndex, spec)
}

func SetCellTextStyle(targetDir string, selector shared.SectionSelector, tableIndex, row, col, paragraphIndex int, runIndex *int, spec shared.TextStyleSpec) (shared.Report, []string, int, error) {
	return shared.SetTableCellTextStyle(targetDir, selector, tableIndex, row, col, paragraphIndex, runIndex, spec)
}

func MergeCells(targetDir string, selector shared.SectionSelector, tableIndex, startRow, startCol, endRow, endCol int) (shared.Report, error) {
	return shared.MergeTableCells(targetDir, selector, tableIndex, startRow, startCol, endRow, endCol)
}

func SplitCell(targetDir string, selector shared.SectionSelector, tableIndex, row, col int) (shared.Report, error) {
	return shared.SplitTableCell(targetDir, selector, tableIndex, row, col)
}

func NormalizeBorders(targetDir string, selector shared.SectionSelector, tableIndex int) (shared.Report, error) {
	return shared.NormalizeTableBorders(targetDir, selector, tableIndex)
}
