package hwpx

type ManifestItem struct {
	ID        string `json:"id"`
	Href      string `json:"href"`
	MediaType string `json:"mediaType"`
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
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Summary  Summary  `json:"summary"`
}
