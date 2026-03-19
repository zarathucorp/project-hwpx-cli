package core

type ManifestItem struct {
	ID        string `json:"id"`
	Href      string `json:"href"`
	MediaType string `json:"mediaType"`
}

type AnalysisCell struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type TemplateCell struct {
	Row            int    `json:"row"`
	Col            int    `json:"col"`
	RowSpan        int    `json:"rowSpan,omitempty"`
	ColSpan        int    `json:"colSpan,omitempty"`
	ParagraphCount int    `json:"paragraphCount"`
	Text           string `json:"text,omitempty"`
}

type TemplateParagraph struct {
	SectionIndex   int           `json:"sectionIndex"`
	SectionPath    string        `json:"sectionPath"`
	ParagraphIndex int           `json:"paragraphIndex"`
	TableIndex     *int          `json:"tableIndex,omitempty"`
	Cell           *AnalysisCell `json:"cell,omitempty"`
	StyleIDRef     string        `json:"styleIdRef,omitempty"`
	StyleName      string        `json:"styleName,omitempty"`
	StyleSummary   string        `json:"styleSummary,omitempty"`
	Text           string        `json:"text"`
}

type TemplateSection struct {
	SectionIndex    int    `json:"sectionIndex"`
	SectionPath     string `json:"sectionPath"`
	ParagraphCount  int    `json:"paragraphCount"`
	TableCount      int    `json:"tableCount"`
	MergedCellCount int    `json:"mergedCellCount"`
	HasHeader       bool   `json:"hasHeader"`
	HasFooter       bool   `json:"hasFooter"`
	HasPageNumber   bool   `json:"hasPageNumber"`
	TextPreview     string `json:"textPreview,omitempty"`
}

type TemplateTable struct {
	SectionIndex     int            `json:"sectionIndex"`
	SectionPath      string         `json:"sectionPath"`
	TableIndex       int            `json:"tableIndex"`
	ParentTableIndex *int           `json:"parentTableIndex,omitempty"`
	NestedDepth      int            `json:"nestedDepth,omitempty"`
	Rows             int            `json:"rows"`
	Cols             int            `json:"cols"`
	MergedCellCount  int            `json:"mergedCellCount"`
	ParagraphCount   int            `json:"paragraphCount"`
	LabelText        string         `json:"labelText,omitempty"`
	TextPreview      string         `json:"textPreview,omitempty"`
	Cells            []TemplateCell `json:"cells,omitempty"`
}

type TemplateTextCandidate struct {
	SectionIndex   int           `json:"sectionIndex"`
	SectionPath    string        `json:"sectionPath"`
	ParagraphIndex int           `json:"paragraphIndex"`
	TableIndex     *int          `json:"tableIndex,omitempty"`
	Cell           *AnalysisCell `json:"cell,omitempty"`
	StyleSummary   string        `json:"styleSummary,omitempty"`
	Text           string        `json:"text"`
	Reason         string        `json:"reason"`
}

type TargetQuery struct {
	Anchor      string `json:"anchor,omitempty"`
	NearText    string `json:"nearText,omitempty"`
	TableLabel  string `json:"tableLabel,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

type TemplateTargetMatch struct {
	Kind           string        `json:"kind"`
	QueryType      string        `json:"queryType"`
	SectionIndex   int           `json:"sectionIndex"`
	SectionPath    string        `json:"sectionPath"`
	ParagraphIndex *int          `json:"paragraphIndex,omitempty"`
	TableIndex     *int          `json:"tableIndex,omitempty"`
	Cell           *AnalysisCell `json:"cell,omitempty"`
	StyleSummary   string        `json:"styleSummary,omitempty"`
	LabelText      string        `json:"labelText,omitempty"`
	Text           string        `json:"text,omitempty"`
	Reason         string        `json:"reason,omitempty"`
	RowSpan        int           `json:"rowSpan,omitempty"`
	ColSpan        int           `json:"colSpan,omitempty"`
}

type TemplateAnalysis struct {
	SectionCount     int                     `json:"sectionCount"`
	TableCount       int                     `json:"tableCount"`
	ParagraphCount   int                     `json:"paragraphCount"`
	PlaceholderCount int                     `json:"placeholderCount"`
	GuideCount       int                     `json:"guideCount"`
	Sections         []TemplateSection       `json:"sections"`
	Tables           []TemplateTable         `json:"tables"`
	Paragraphs       []TemplateParagraph     `json:"paragraphs"`
	Placeholders     []TemplateTextCandidate `json:"placeholders"`
	Guides           []TemplateTextCandidate `json:"guides"`
}

type RoundtripSnapshot struct {
	Valid              bool     `json:"valid"`
	RenderSafe         bool     `json:"renderSafe"`
	RiskHints          []string `json:"riskHints,omitempty"`
	SectionPaths       []string `json:"sectionPaths,omitempty"`
	SectionCount       int      `json:"sectionCount"`
	TableCount         int      `json:"tableCount"`
	ParagraphCount     int      `json:"paragraphCount"`
	PlaceholderCount   int      `json:"placeholderCount"`
	GuideCount         int      `json:"guideCount"`
	ObjectCount        int      `json:"objectCount"`
	ControlCount       int      `json:"controlCount"`
	HeaderCount        int      `json:"headerCount"`
	FooterCount        int      `json:"footerCount"`
	TextLength         int      `json:"textLength"`
	LineCount          int      `json:"lineCount"`
	TextDigest         string   `json:"textDigest,omitempty"`
	ParagraphDigest    string   `json:"paragraphDigest,omitempty"`
	TableDigest        string   `json:"tableDigest,omitempty"`
	ObjectDigest       string   `json:"objectDigest,omitempty"`
	HeaderFooterDigest string   `json:"headerFooterDigest,omitempty"`
	ControlDigest      string   `json:"controlDigest,omitempty"`
}

type RoundtripIssue struct {
	Code           string        `json:"code"`
	Severity       string        `json:"severity"`
	Message        string        `json:"message"`
	SectionIndex   *int          `json:"sectionIndex,omitempty"`
	SectionPath    string        `json:"sectionPath,omitempty"`
	ParagraphIndex *int          `json:"paragraphIndex,omitempty"`
	TableIndex     *int          `json:"tableIndex,omitempty"`
	Cell           *AnalysisCell `json:"cell,omitempty"`
	Before         string        `json:"before,omitempty"`
	After          string        `json:"after,omitempty"`
}

type RoundtripCheckReport struct {
	Passed bool              `json:"passed"`
	Before RoundtripSnapshot `json:"before"`
	After  RoundtripSnapshot `json:"after"`
	Issues []RoundtripIssue  `json:"issues,omitempty"`
}

type Summary struct {
	Entries     []string          `json:"entries"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Version     map[string]string `json:"version,omitempty"`
	Manifest    []ManifestItem    `json:"manifest,omitempty"`
	Spine       []string          `json:"spine,omitempty"`
	SectionPath []string          `json:"sectionPaths,omitempty"`
	BinaryPath  []string          `json:"binaryPaths,omitempty"`
}

type Report struct {
	Valid       bool           `json:"valid"`
	RenderSafe  bool           `json:"renderSafe"`
	Errors      []string       `json:"errors"`
	Warnings    []string       `json:"warnings"`
	RiskHints   []string       `json:"riskHints,omitempty"`
	RiskSignals map[string]int `json:"riskSignals,omitempty"`
	Summary     Summary        `json:"summary"`
}
