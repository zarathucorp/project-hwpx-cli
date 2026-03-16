package shared

import (
	"github.com/beevik/etree"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx/core"
)

type Report = core.Report

func Validate(targetPath string) (Report, error) {
	return core.Validate(targetPath)
}

func Unpack(filePath, outputDir string) error {
	return core.Unpack(filePath, outputDir)
}

const (
	defaultTableWidth   = 42520
	defaultCellHeight   = 2400
	defaultImageWidth   = 22677
	defaultEquationVer  = "Equation Version 60"
	defaultEquationFont = "HancomEQN"
	defaultSectionPath  = "Contents/section0.xml"
	templateGlob        = "example/*.hwpx"
	pageToken           = "{{PAGE}}"
	totalPageToken      = "{{TOTAL_PAGE}}"
)

type TableSpec struct {
	Rows  int
	Cols  int
	Cells [][]string
}

type tableGridEntry struct {
	cell   *etree.Element
	row    int
	col    int
	anchor [2]int
	span   [2]int
}

type ImageEmbed struct {
	ItemID     string
	BinaryPath string
}

type ImagePlacement struct {
	ItemID      string
	BinaryPath  string
	PixelWidth  int
	PixelHeight int
	Width       int
	Height      int
}

type HeaderFooterSpec struct {
	Text          []string
	ApplyPageType string
}

type PageNumberSpec struct {
	Position   string
	FormatType string
	SideChar   string
	StartPage  int
}

type NoteSpec struct {
	AnchorText string
	Text       []string
}

type MemoSpec struct {
	AnchorText string
	Text       []string
	Author     string
}

type BookmarkSpec struct {
	Name string
	Text string
}

type HyperlinkSpec struct {
	Target string
	Text   string
}

type HeadingSpec struct {
	Kind         string
	Level        int
	Text         string
	BookmarkName string
}

type TOCSpec struct {
	Title    string
	MaxLevel int
}

type CrossReferenceSpec struct {
	BookmarkName string
	Text         string
}

type EquationSpec struct {
	Script string
}

type RectangleSpec struct {
	WidthMM   float64
	HeightMM  float64
	LineColor string
	FillColor string
}

type styleRef struct {
	ID          string
	Name        string
	ParaPrIDRef string
	CharPrIDRef string
}

type headingEntry struct {
	Level        int
	Text         string
	BookmarkName string
}
