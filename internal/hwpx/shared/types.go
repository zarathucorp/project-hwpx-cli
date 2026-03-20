package shared

import (
	"time"

	"github.com/beevik/etree"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx/core"
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
	Rows           int
	Cols           int
	Cells          [][]string
	WidthMM        *float64
	HeightMM       *float64
	ColWidthsMM    []float64
	RowHeightsMM   []float64
	MarginLeftMM   *float64
	MarginRightMM  *float64
	MarginTopMM    *float64
	MarginBottomMM *float64
}

type TableCellStyleSpec struct {
	Text                *string
	VertAlign           string
	MarginLeftMM        *float64
	MarginRightMM       *float64
	MarginTopMM         *float64
	MarginBottomMM      *float64
	BorderStyle         string
	BorderColor         string
	BorderWidthMM       *float64
	BorderLeftStyle     string
	BorderRightStyle    string
	BorderTopStyle      string
	BorderBottomStyle   string
	BorderLeftColor     string
	BorderRightColor    string
	BorderTopColor      string
	BorderBottomColor   string
	BorderLeftWidthMM   *float64
	BorderRightWidthMM  *float64
	BorderTopWidthMM    *float64
	BorderBottomWidthMM *float64
	FillColor           string
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
	Bold       *bool
	Italic     *bool
	Underline  *bool
	TextColor  string
	FontName   string
	FontSizePt *float64
}

type TableCellTextSpec struct {
	Text            string
	ParagraphLayout ParagraphLayoutSpec
	TextStyle       TextStyleSpec
}

type SectionSelector struct {
	Section     *int
	AllSections bool
}

type FillTemplateReplacement struct {
	SourceIndex   *int                `json:"sourceIndex,omitempty" yaml:"sourceIndex,omitempty"`
	Placeholder   string              `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
	Anchor        string              `json:"anchor,omitempty" yaml:"anchor,omitempty"`
	NearText      string              `json:"nearText,omitempty" yaml:"nearText,omitempty"`
	TableLabel    string              `json:"tableLabel,omitempty" yaml:"tableLabel,omitempty"`
	TableIndex    *int                `json:"tableIndex,omitempty" yaml:"tableIndex,omitempty"`
	Occurrence    *int                `json:"occurrence,omitempty" yaml:"occurrence,omitempty"`
	MatchMode     string              `json:"matchMode,omitempty" yaml:"matchMode,omitempty"`
	Required      bool                `json:"required,omitempty" yaml:"required,omitempty"`
	Unique        bool                `json:"unique,omitempty" yaml:"unique,omitempty"`
	FallbackValue string              `json:"fallbackValue,omitempty" yaml:"fallbackValue,omitempty"`
	Expand        bool                `json:"expand,omitempty" yaml:"expand,omitempty"`
	Fields        []string            `json:"fields,omitempty" yaml:"fields,omitempty"`
	Records       []map[string]string `json:"records,omitempty" yaml:"records,omitempty"`
	Value         string              `json:"value" yaml:"value"`
	Values        []string            `json:"values,omitempty" yaml:"values,omitempty"`
	Grid          [][]string          `json:"grid,omitempty" yaml:"grid,omitempty"`
	Mode          string              `json:"mode,omitempty" yaml:"mode,omitempty"`
}

type FillTemplateChange struct {
	ResolutionIndex *int                 `json:"resolutionIndex,omitempty"`
	Kind            string               `json:"kind"`
	Mode            string               `json:"mode"`
	SectionIndex    int                  `json:"sectionIndex"`
	SectionPath     string               `json:"sectionPath"`
	ParagraphIndex  *int                 `json:"paragraphIndex,omitempty"`
	TableIndex      *int                 `json:"tableIndex,omitempty"`
	Cell            *TableCellCoordinate `json:"cell,omitempty"`
	TableLabel      string               `json:"tableLabel,omitempty"`
	Selector        string               `json:"selector"`
	PreviousText    string               `json:"previousText,omitempty"`
	Expand          bool                 `json:"expand,omitempty"`
	Text            string               `json:"text"`
}

type FillTemplateMiss struct {
	ResolutionIndex *int   `json:"resolutionIndex,omitempty"`
	Kind            string `json:"kind"`
	Mode            string `json:"mode"`
	Selector        string `json:"selector"`
	TableLabel      string `json:"tableLabel,omitempty"`
	TableIndex      *int   `json:"tableIndex,omitempty"`
	Occurrence      *int   `json:"occurrence,omitempty"`
	Required        bool   `json:"required,omitempty"`
	Reason          string `json:"reason"`
	Requested       int    `json:"requested"`
	Matched         int    `json:"matched"`
	Partial         bool   `json:"partial"`
	SectionScoped   bool   `json:"sectionScoped"`
}

type TableCellCoordinate struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type RunStyleFilter struct {
	Bold       *bool
	Italic     *bool
	Underline  *bool
	TextColor  string
	FontName   string
	FontSizePt *float64
}

type RunStyleMatch struct {
	SectionIndex   int                  `json:"sectionIndex"`
	SectionPath    string               `json:"sectionPath,omitempty"`
	ParagraphIndex int                  `json:"paragraphIndex"`
	Paragraph      int                  `json:"paragraph"`
	Run            int                  `json:"run"`
	TableIndex     *int                 `json:"tableIndex,omitempty"`
	Cell           *TableCellCoordinate `json:"cell,omitempty"`
	Text           string               `json:"text"`
	CharPrIDRef    string               `json:"charPrIdRef"`
	Bold           bool                 `json:"bold"`
	Italic         bool                 `json:"italic"`
	Underline      bool                 `json:"underline"`
	TextColor      string               `json:"textColor"`
	FontName       string               `json:"fontName,omitempty"`
	FontSizePt     float64              `json:"fontSizePt,omitempty"`
}

type RunTextReplacement struct {
	SectionIndex   int                  `json:"sectionIndex"`
	SectionPath    string               `json:"sectionPath,omitempty"`
	ParagraphIndex int                  `json:"paragraphIndex"`
	Paragraph      int                  `json:"paragraph"`
	Run            int                  `json:"run"`
	TableIndex     *int                 `json:"tableIndex,omitempty"`
	Cell           *TableCellCoordinate `json:"cell,omitempty"`
	PreviousText   string               `json:"previousText"`
	Text           string               `json:"text"`
	CharPrIDRef    string               `json:"charPrIdRef"`
}

type ObjectFilter struct {
	Types []string
}

type ObjectMatch struct {
	Index          int                  `json:"index"`
	SectionIndex   int                  `json:"sectionIndex"`
	SectionPath    string               `json:"sectionPath,omitempty"`
	ParagraphIndex int                  `json:"paragraphIndex"`
	Paragraph      int                  `json:"paragraph"`
	Run            int                  `json:"run"`
	TableIndex     *int                 `json:"tableIndex,omitempty"`
	Cell           *TableCellCoordinate `json:"cell,omitempty"`
	Path           string               `json:"path"`
	Type           string               `json:"type"`
	Tag            string               `json:"tag"`
	ID             string               `json:"id,omitempty"`
	Ref            string               `json:"ref,omitempty"`
	Text           string               `json:"text,omitempty"`
	Rows           int                  `json:"rows,omitempty"`
	Cols           int                  `json:"cols,omitempty"`
}

type TagFilter struct {
	Tag string
}

type TagMatch struct {
	Index          int                  `json:"index"`
	SectionIndex   int                  `json:"sectionIndex"`
	SectionPath    string               `json:"sectionPath,omitempty"`
	ParagraphIndex int                  `json:"paragraphIndex"`
	Paragraph      int                  `json:"paragraph"`
	Run            int                  `json:"run"`
	TableIndex     *int                 `json:"tableIndex,omitempty"`
	Cell           *TableCellCoordinate `json:"cell,omitempty"`
	Path           string               `json:"path"`
	Tag            string               `json:"tag"`
	Text           string               `json:"text,omitempty"`
}

type AttributeFilter struct {
	Attr  string
	Value string
	Tag   string
}

type AttributeMatch struct {
	Index          int                  `json:"index"`
	SectionIndex   int                  `json:"sectionIndex"`
	SectionPath    string               `json:"sectionPath,omitempty"`
	ParagraphIndex int                  `json:"paragraphIndex"`
	Paragraph      int                  `json:"paragraph"`
	Run            int                  `json:"run"`
	TableIndex     *int                 `json:"tableIndex,omitempty"`
	Cell           *TableCellCoordinate `json:"cell,omitempty"`
	Path           string               `json:"path"`
	Tag            string               `json:"tag"`
	Attr           string               `json:"attr"`
	Value          string               `json:"value"`
	Text           string               `json:"text,omitempty"`
}

type XPathFilter struct {
	Expr string
}

type XPathMatch struct {
	Index          int                  `json:"index"`
	SectionIndex   int                  `json:"sectionIndex"`
	SectionPath    string               `json:"sectionPath,omitempty"`
	ParagraphIndex int                  `json:"paragraphIndex"`
	Paragraph      int                  `json:"paragraph"`
	Run            int                  `json:"run"`
	TableIndex     *int                 `json:"tableIndex,omitempty"`
	Cell           *TableCellCoordinate `json:"cell,omitempty"`
	Path           string               `json:"path"`
	Tag            string               `json:"tag"`
	Text           string               `json:"text,omitempty"`
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
