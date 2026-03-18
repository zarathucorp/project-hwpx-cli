package shared

import (
	"time"

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

type ObjectPositionSpec struct {
	Type        string
	Index       int
	TreatAsChar *bool
	XMM         *float64
	YMM         *float64
	HorzAlign   string
	VertAlign   string
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

type ColumnSpec struct {
	Count int
	GapMM float64
}

type PageLayoutSpec struct {
	Orientation          string
	WidthMM              *float64
	HeightMM             *float64
	LeftMarginMM         *float64
	RightMarginMM        *float64
	TopMarginMM          *float64
	BottomMarginMM       *float64
	HeaderMarginMM       *float64
	FooterMarginMM       *float64
	GutterMarginMM       *float64
	GutterType           string
	BorderFillIDRef      *int
	BorderTextBorder     string
	BorderFillArea       string
	BorderHeaderInside   *bool
	BorderFooterInside   *bool
	BorderOffsetLeftMM   *float64
	BorderOffsetRightMM  *float64
	BorderOffsetTopMM    *float64
	BorderOffsetBottomMM *float64
}

type ParagraphLayoutSpec struct {
	Align              string
	IndentMM           *float64
	LeftMarginMM       *float64
	RightMarginMM      *float64
	SpaceBeforeMM      *float64
	SpaceAfterMM       *float64
	LineSpacingPercent *int
}

type ParagraphListSpec struct {
	Kind        string
	Level       int
	StartNumber *int
}

type TextStyleSpec struct {
	Bold      *bool
	Italic    *bool
	Underline *bool
	TextColor string
}

type RunStyleFilter struct {
	Bold      *bool
	Italic    *bool
	Underline *bool
	TextColor string
}

type RunStyleMatch struct {
	Paragraph   int    `json:"paragraph"`
	Run         int    `json:"run"`
	Text        string `json:"text"`
	CharPrIDRef string `json:"charPrIdRef"`
	Bold        bool   `json:"bold"`
	Italic      bool   `json:"italic"`
	Underline   bool   `json:"underline"`
	TextColor   string `json:"textColor"`
}

type RunTextReplacement struct {
	Paragraph    int    `json:"paragraph"`
	Run          int    `json:"run"`
	PreviousText string `json:"previousText"`
	Text         string `json:"text"`
	CharPrIDRef  string `json:"charPrIdRef"`
}

type ObjectFilter struct {
	Types []string
}

type ObjectMatch struct {
	Index     int    `json:"index"`
	Type      string `json:"type"`
	Paragraph int    `json:"paragraph"`
	Run       int    `json:"run"`
	Path      string `json:"path"`
	Tag       string `json:"tag"`
	ID        string `json:"id,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Text      string `json:"text,omitempty"`
	Rows      int    `json:"rows,omitempty"`
	Cols      int    `json:"cols,omitempty"`
}

type TagFilter struct {
	Tag string
}

type TagMatch struct {
	Index     int    `json:"index"`
	Paragraph int    `json:"paragraph"`
	Run       int    `json:"run"`
	Path      string `json:"path"`
	Tag       string `json:"tag"`
	Text      string `json:"text,omitempty"`
}

type AttributeFilter struct {
	Attr  string
	Value string
	Tag   string
}

type AttributeMatch struct {
	Index     int    `json:"index"`
	Paragraph int    `json:"paragraph"`
	Run       int    `json:"run"`
	Path      string `json:"path"`
	Tag       string `json:"tag"`
	Attr      string `json:"attr"`
	Value     string `json:"value"`
	Text      string `json:"text,omitempty"`
}

type XPathFilter struct {
	Expr string
}

type XPathMatch struct {
	Index     int    `json:"index"`
	Paragraph int    `json:"paragraph"`
	Run       int    `json:"run"`
	Path      string `json:"path"`
	Tag       string `json:"tag"`
	Text      string `json:"text,omitempty"`
}

type HistoryEntrySpec struct {
	Command   string
	Author    string
	Summary   string
	Timestamp time.Time
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

type LineSpec struct {
	WidthMM   float64
	HeightMM  float64
	LineColor string
}

type EllipseSpec struct {
	WidthMM   float64
	HeightMM  float64
	LineColor string
	FillColor string
}

type TextBoxSpec struct {
	WidthMM   float64
	HeightMM  float64
	Text      []string
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
