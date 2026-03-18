package hwpx

import (
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/document"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/history"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/layout"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/media"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/note"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/object"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/paragraph"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/reference"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/search"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/section"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/table"
)

func CreateEditableDocument(outputDir string) (Report, error) {
	return document.CreateEditableDocument(outputDir)
}

func RecordHistory(targetDir string, spec HistoryEntrySpec) error {
	return history.Record(targetDir, spec)
}

func AddParagraphs(targetDir string, texts []string) (Report, int, error) {
	return paragraph.Add(targetDir, texts)
}

func AddRunText(targetDir string, paragraphIndex int, runIndex *int, text string) (Report, int, string, error) {
	return paragraph.AddRunText(targetDir, paragraphIndex, runIndex, text)
}

func SetRunText(targetDir string, paragraphIndex, runIndex int, text string) (Report, string, string, error) {
	return paragraph.SetRunText(targetDir, paragraphIndex, runIndex, text)
}

func FindRunsByStyle(targetDir string, filter RunStyleFilter) ([]RunStyleMatch, error) {
	return paragraph.FindRunsByStyle(targetDir, filter)
}

func ReplaceRunsByStyle(targetDir string, filter RunStyleFilter, text string) (Report, []RunTextReplacement, error) {
	return paragraph.ReplaceRunsByStyle(targetDir, filter, text)
}

func SetParagraphText(targetDir string, paragraphIndex int, text string) (Report, string, error) {
	return paragraph.SetText(targetDir, paragraphIndex, text)
}

func SetParagraphLayout(targetDir string, paragraphIndex int, spec ParagraphLayoutSpec) (Report, string, error) {
	return paragraph.SetLayout(targetDir, paragraphIndex, spec)
}

func SetParagraphList(targetDir string, paragraphIndex int, spec ParagraphListSpec) (Report, string, error) {
	return paragraph.SetList(targetDir, paragraphIndex, spec)
}

func ApplyTextStyle(targetDir string, paragraphIndex int, runIndex *int, spec TextStyleSpec) (Report, []string, int, error) {
	return paragraph.ApplyTextStyle(targetDir, paragraphIndex, runIndex, spec)
}

func DeleteParagraph(targetDir string, paragraphIndex int) (Report, string, error) {
	return paragraph.Delete(targetDir, paragraphIndex)
}

func AddSection(targetDir string) (Report, int, string, error) {
	return section.Add(targetDir)
}

func DeleteSection(targetDir string, sectionIndex int) (Report, string, error) {
	return section.Delete(targetDir, sectionIndex)
}

func AddTable(targetDir string, spec TableSpec) (Report, int, error) {
	return table.Add(targetDir, spec)
}

func AddNestedTable(targetDir string, tableIndex, row, col int, spec TableSpec) (Report, error) {
	return table.AddNested(targetDir, tableIndex, row, col, spec)
}

func SetTableCellText(targetDir string, tableIndex, row, col int, text string) (Report, error) {
	return table.SetCellText(targetDir, tableIndex, row, col, text)
}

func MergeTableCells(targetDir string, tableIndex, startRow, startCol, endRow, endCol int) (Report, error) {
	return table.MergeCells(targetDir, tableIndex, startRow, startCol, endRow, endCol)
}

func SplitTableCell(targetDir string, tableIndex, row, col int) (Report, error) {
	return table.SplitCell(targetDir, tableIndex, row, col)
}

func EmbedImage(targetDir, imagePath string) (Report, ImageEmbed, error) {
	return media.EmbedImage(targetDir, imagePath)
}

func InsertImage(targetDir, imagePath string, widthMM float64) (Report, ImagePlacement, error) {
	return media.InsertImage(targetDir, imagePath, widthMM)
}

func SetHeaderText(targetDir string, spec HeaderFooterSpec) (Report, error) {
	return layout.SetHeaderText(targetDir, spec)
}

func SetFooterText(targetDir string, spec HeaderFooterSpec) (Report, error) {
	return layout.SetFooterText(targetDir, spec)
}

func RemoveHeader(targetDir string) (Report, error) {
	return layout.RemoveHeader(targetDir)
}

func RemoveFooter(targetDir string) (Report, error) {
	return layout.RemoveFooter(targetDir)
}

func SetPageNumber(targetDir string, spec PageNumberSpec) (Report, error) {
	return layout.SetPageNumber(targetDir, spec)
}

func SetColumns(targetDir string, spec ColumnSpec) (Report, error) {
	return layout.SetColumns(targetDir, spec)
}

func AddFootnote(targetDir string, spec NoteSpec) (Report, int, error) {
	return note.AddFootnote(targetDir, spec)
}

func AddEndnote(targetDir string, spec NoteSpec) (Report, int, error) {
	return note.AddEndnote(targetDir, spec)
}

func AddBookmark(targetDir string, spec BookmarkSpec) (Report, error) {
	return reference.AddBookmark(targetDir, spec)
}

func AddHyperlink(targetDir string, spec HyperlinkSpec) (Report, string, error) {
	return reference.AddHyperlink(targetDir, spec)
}

func AddHeading(targetDir string, spec HeadingSpec) (Report, string, error) {
	return reference.AddHeading(targetDir, spec)
}

func InsertTOC(targetDir string, spec TOCSpec) (Report, int, error) {
	return reference.InsertTOC(targetDir, spec)
}

func AddCrossReference(targetDir string, spec CrossReferenceSpec) (Report, string, string, error) {
	return reference.AddCrossReference(targetDir, spec)
}

func AddEquation(targetDir string, spec EquationSpec) (Report, string, error) {
	return object.AddEquation(targetDir, spec)
}

func AddMemo(targetDir string, spec MemoSpec) (Report, string, string, int, error) {
	return note.AddMemo(targetDir, spec)
}

func AddRectangle(targetDir string, spec RectangleSpec) (Report, string, int, int, error) {
	return object.AddRectangle(targetDir, spec)
}

func AddLine(targetDir string, spec LineSpec) (Report, string, int, int, error) {
	return object.AddLine(targetDir, spec)
}

func AddEllipse(targetDir string, spec EllipseSpec) (Report, string, int, int, error) {
	return object.AddEllipse(targetDir, spec)
}

func AddTextBox(targetDir string, spec TextBoxSpec) (Report, string, int, int, error) {
	return object.AddTextBox(targetDir, spec)
}

func SetObjectPosition(targetDir string, spec ObjectPositionSpec) (Report, string, error) {
	return object.SetPosition(targetDir, spec)
}

func FindObjects(targetDir string, filter ObjectFilter) ([]ObjectMatch, error) {
	return object.FindObjects(targetDir, filter)
}

func FindByTag(targetDir string, filter TagFilter) ([]TagMatch, error) {
	return search.FindByTag(targetDir, filter)
}

func FindByAttr(targetDir string, filter AttributeFilter) ([]AttributeMatch, error) {
	return search.FindByAttr(targetDir, filter)
}

func FindByXPath(targetDir string, filter XPathFilter) ([]XPathMatch, error) {
	return search.FindByXPath(targetDir, filter)
}
