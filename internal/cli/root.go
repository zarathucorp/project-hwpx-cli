package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

const (
	version             = "0.1.0"
	schemaVersion       = "hwpxctl/v1"
	defaultErrorKind    = "command_failed"
	defaultFormatEnvVar = "HWPXCTL_FORMAT"
)

type outputFormat string

const (
	formatDefault outputFormat = ""
	formatText    outputFormat = "text"
	formatJSON    outputFormat = "json"
)

type commandError struct {
	message string
	code    int
	kind    string
	data    any
	silent  bool
}

type commandOptions struct {
	input          string
	output         string
	format         outputFormat
	formatExplicit bool
}

type responseEnvelope struct {
	SchemaVersion string         `json:"schemaVersion"`
	Command       string         `json:"command"`
	Success       bool           `json:"success"`
	Data          any            `json:"data,omitempty"`
	Error         *responseError `json:"error,omitempty"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type inspectResult struct {
	InputPath string      `json:"inputPath"`
	Report    hwpx.Report `json:"report"`
}

type validateResult struct {
	InputPath string      `json:"inputPath"`
	Report    hwpx.Report `json:"report"`
}

type textResult struct {
	InputPath      string `json:"inputPath"`
	OutputPath     string `json:"outputPath,omitempty"`
	Text           string `json:"text,omitempty"`
	LineCount      int    `json:"lineCount"`
	CharacterCount int    `json:"characterCount"`
}

type exportResult struct {
	InputPath      string `json:"inputPath"`
	OutputPath     string `json:"outputPath,omitempty"`
	Format         string `json:"format"`
	Content        string `json:"content,omitempty"`
	LineCount      int    `json:"lineCount"`
	CharacterCount int    `json:"characterCount"`
	BlockCount     int    `json:"blockCount"`
}

type packResult struct {
	InputPath  string      `json:"inputPath"`
	OutputPath string      `json:"outputPath"`
	Report     hwpx.Report `json:"report"`
}

type unpackResult struct {
	InputPath  string      `json:"inputPath"`
	OutputPath string      `json:"outputPath"`
	Report     hwpx.Report `json:"report"`
}

type createResult struct {
	OutputPath string      `json:"outputPath"`
	Report     hwpx.Report `json:"report"`
}

type paragraphEditResult struct {
	InputPath       string      `json:"inputPath"`
	AddedParagraphs int         `json:"addedParagraphs"`
	Report          hwpx.Report `json:"report"`
}

type runTextAddResult struct {
	InputPath   string      `json:"inputPath"`
	Paragraph   int         `json:"paragraph"`
	Run         int         `json:"run"`
	Text        string      `json:"text"`
	CharPrIDRef string      `json:"charPrIdRef"`
	Report      hwpx.Report `json:"report"`
}

type runTextUpdateResult struct {
	InputPath    string      `json:"inputPath"`
	Paragraph    int         `json:"paragraph"`
	Run          int         `json:"run"`
	Text         string      `json:"text"`
	PreviousText string      `json:"previousText,omitempty"`
	CharPrIDRef  string      `json:"charPrIdRef"`
	Report       hwpx.Report `json:"report"`
}

type runStyleSearchResult struct {
	InputPath string               `json:"inputPath"`
	Count     int                  `json:"count"`
	Matches   []hwpx.RunStyleMatch `json:"matches"`
}

type runStyleReplaceResult struct {
	InputPath    string                    `json:"inputPath"`
	Count        int                       `json:"count"`
	Text         string                    `json:"text"`
	Replacements []hwpx.RunTextReplacement `json:"replacements"`
	Report       hwpx.Report               `json:"report"`
}

type objectSearchResult struct {
	InputPath string             `json:"inputPath"`
	Count     int                `json:"count"`
	Matches   []hwpx.ObjectMatch `json:"matches"`
}

type tagSearchResult struct {
	InputPath string          `json:"inputPath"`
	Count     int             `json:"count"`
	Matches   []hwpx.TagMatch `json:"matches"`
}

type attributeSearchResult struct {
	InputPath string                `json:"inputPath"`
	Count     int                   `json:"count"`
	Matches   []hwpx.AttributeMatch `json:"matches"`
}

type xpathSearchResult struct {
	InputPath string            `json:"inputPath"`
	Count     int               `json:"count"`
	Matches   []hwpx.XPathMatch `json:"matches"`
}

type paragraphUpdateResult struct {
	InputPath    string      `json:"inputPath"`
	Paragraph    int         `json:"paragraph"`
	PreviousText string      `json:"previousText,omitempty"`
	RemovedText  string      `json:"removedText,omitempty"`
	Deleted      bool        `json:"deleted"`
	Report       hwpx.Report `json:"report"`
}

type paragraphLayoutResult struct {
	InputPath          string      `json:"inputPath"`
	Paragraph          int         `json:"paragraph"`
	ParaPrIDRef        string      `json:"paraPrIdRef"`
	Align              string      `json:"align,omitempty"`
	IndentMM           *float64    `json:"indentMm,omitempty"`
	LeftMarginMM       *float64    `json:"leftMarginMm,omitempty"`
	RightMarginMM      *float64    `json:"rightMarginMm,omitempty"`
	SpaceBeforeMM      *float64    `json:"spaceBeforeMm,omitempty"`
	SpaceAfterMM       *float64    `json:"spaceAfterMm,omitempty"`
	LineSpacingPercent *int        `json:"lineSpacingPercent,omitempty"`
	Report             hwpx.Report `json:"report"`
}

type paragraphListResult struct {
	InputPath   string      `json:"inputPath"`
	Paragraph   int         `json:"paragraph"`
	Kind        string      `json:"kind"`
	Level       int         `json:"level"`
	StartNumber *int        `json:"startNumber,omitempty"`
	ParaPrIDRef string      `json:"paraPrIdRef"`
	Report      hwpx.Report `json:"report"`
}

type textStyleResult struct {
	InputPath   string      `json:"inputPath"`
	Paragraph   int         `json:"paragraph"`
	Run         *int        `json:"run,omitempty"`
	AppliedRuns int         `json:"appliedRuns"`
	CharPrIDs   []string    `json:"charPrIds"`
	Bold        *bool       `json:"bold,omitempty"`
	Italic      *bool       `json:"italic,omitempty"`
	Underline   *bool       `json:"underline,omitempty"`
	TextColor   string      `json:"textColor,omitempty"`
	Report      hwpx.Report `json:"report"`
}

type sectionEditResult struct {
	InputPath   string      `json:"inputPath"`
	Section     int         `json:"section"`
	SectionPath string      `json:"sectionPath"`
	Deleted     bool        `json:"deleted"`
	RemovedPath string      `json:"removedPath,omitempty"`
	Report      hwpx.Report `json:"report"`
}

type tableAddResult struct {
	InputPath  string      `json:"inputPath"`
	TableIndex int         `json:"tableIndex"`
	Rows       int         `json:"rows"`
	Cols       int         `json:"cols"`
	Report     hwpx.Report `json:"report"`
}

type nestedTableAddResult struct {
	InputPath  string      `json:"inputPath"`
	TableIndex int         `json:"tableIndex"`
	Row        int         `json:"row"`
	Col        int         `json:"col"`
	Rows       int         `json:"rows"`
	Cols       int         `json:"cols"`
	Report     hwpx.Report `json:"report"`
}

type tableCellEditResult struct {
	InputPath  string      `json:"inputPath"`
	TableIndex int         `json:"tableIndex"`
	Row        int         `json:"row"`
	Col        int         `json:"col"`
	Report     hwpx.Report `json:"report"`
}

type tableMergeResult struct {
	InputPath  string      `json:"inputPath"`
	TableIndex int         `json:"tableIndex"`
	StartRow   int         `json:"startRow"`
	StartCol   int         `json:"startCol"`
	EndRow     int         `json:"endRow"`
	EndCol     int         `json:"endCol"`
	Report     hwpx.Report `json:"report"`
}

type imageEmbedResult struct {
	InputPath  string      `json:"inputPath"`
	ImagePath  string      `json:"imagePath"`
	ItemID     string      `json:"itemId"`
	BinaryPath string      `json:"binaryPath"`
	Report     hwpx.Report `json:"report"`
}

type imageInsertResult struct {
	InputPath    string      `json:"inputPath"`
	ImagePath    string      `json:"imagePath"`
	ItemID       string      `json:"itemId"`
	BinaryPath   string      `json:"binaryPath"`
	PixelWidth   int         `json:"pixelWidth"`
	PixelHeight  int         `json:"pixelHeight"`
	PlacedWidth  int         `json:"placedWidth"`
	PlacedHeight int         `json:"placedHeight"`
	Report       hwpx.Report `json:"report"`
}

type printPDFResult struct {
	InputPath  string `json:"inputPath"`
	OutputPath string `json:"outputPath"`
}

type headerFooterResult struct {
	InputPath     string      `json:"inputPath"`
	Kind          string      `json:"kind"`
	ApplyPageType string      `json:"applyPageType"`
	Report        hwpx.Report `json:"report"`
}

type pageNumberResult struct {
	InputPath  string      `json:"inputPath"`
	Position   string      `json:"position"`
	FormatType string      `json:"formatType"`
	SideChar   string      `json:"sideChar"`
	StartPage  int         `json:"startPage"`
	Report     hwpx.Report `json:"report"`
}

type columnsResult struct {
	InputPath string      `json:"inputPath"`
	Count     int         `json:"count"`
	GapMM     float64     `json:"gapMm"`
	Report    hwpx.Report `json:"report"`
}

type noteResult struct {
	InputPath string      `json:"inputPath"`
	Kind      string      `json:"kind"`
	Number    int         `json:"number"`
	Report    hwpx.Report `json:"report"`
}

type memoResult struct {
	InputPath string      `json:"inputPath"`
	MemoID    string      `json:"memoId"`
	FieldID   string      `json:"fieldId"`
	Number    int         `json:"number"`
	Author    string      `json:"author"`
	Report    hwpx.Report `json:"report"`
}

type bookmarkResult struct {
	InputPath string      `json:"inputPath"`
	Name      string      `json:"name"`
	Report    hwpx.Report `json:"report"`
}

type hyperlinkResult struct {
	InputPath string      `json:"inputPath"`
	Target    string      `json:"target"`
	FieldID   string      `json:"fieldId"`
	Report    hwpx.Report `json:"report"`
}

type headingResult struct {
	InputPath    string      `json:"inputPath"`
	Kind         string      `json:"kind"`
	Level        int         `json:"level"`
	Text         string      `json:"text"`
	BookmarkName string      `json:"bookmarkName"`
	Report       hwpx.Report `json:"report"`
}

type tocResult struct {
	InputPath  string      `json:"inputPath"`
	Title      string      `json:"title"`
	MaxLevel   int         `json:"maxLevel"`
	EntryCount int         `json:"entryCount"`
	Report     hwpx.Report `json:"report"`
}

type crossReferenceResult struct {
	InputPath    string      `json:"inputPath"`
	BookmarkName string      `json:"bookmarkName"`
	Text         string      `json:"text"`
	FieldID      string      `json:"fieldId"`
	Report       hwpx.Report `json:"report"`
}

type equationResult struct {
	InputPath string      `json:"inputPath"`
	Script    string      `json:"script"`
	ItemID    string      `json:"itemId"`
	Report    hwpx.Report `json:"report"`
}

type rectangleResult struct {
	InputPath string      `json:"inputPath"`
	ShapeID   string      `json:"shapeId"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	Report    hwpx.Report `json:"report"`
}

type lineResult struct {
	InputPath string      `json:"inputPath"`
	ShapeID   string      `json:"shapeId"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	Report    hwpx.Report `json:"report"`
}

type ellipseResult struct {
	InputPath string      `json:"inputPath"`
	ShapeID   string      `json:"shapeId"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	Report    hwpx.Report `json:"report"`
}

type textBoxResult struct {
	InputPath string      `json:"inputPath"`
	ShapeID   string      `json:"shapeId"`
	Text      []string    `json:"text"`
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	Report    hwpx.Report `json:"report"`
}

type objectPositionResult struct {
	InputPath   string      `json:"inputPath"`
	Type        string      `json:"type"`
	Index       int         `json:"index"`
	ObjectID    string      `json:"objectId"`
	TreatAsChar *bool       `json:"treatAsChar,omitempty"`
	XMM         *float64    `json:"xMm,omitempty"`
	YMM         *float64    `json:"yMm,omitempty"`
	HorzAlign   string      `json:"horzAlign,omitempty"`
	VertAlign   string      `json:"vertAlign,omitempty"`
	Report      hwpx.Report `json:"report"`
}

type schemaDoc struct {
	SchemaVersion string            `json:"schemaVersion"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Environment   []environmentSpec `json:"environment"`
	Commands      []commandSpec     `json:"commands"`
	Response      responseSpec      `json:"responseEnvelope"`
}

type environmentSpec struct {
	Name        string   `json:"name"`
	Values      []string `json:"values"`
	Default     string   `json:"default"`
	Description string   `json:"description"`
}

type commandSpec struct {
	Name        string       `json:"name"`
	Summary     string       `json:"summary"`
	Arguments   []argument   `json:"arguments,omitempty"`
	Options     []optionSpec `json:"options,omitempty"`
	JSONCapable bool         `json:"jsonCapable"`
	Examples    []string     `json:"examples,omitempty"`
}

type argument struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type optionSpec struct {
	Name        string   `json:"name"`
	Values      []string `json:"values,omitempty"`
	Required    bool     `json:"required"`
	Description string   `json:"description"`
}

type responseSpec struct {
	Format string          `json:"format"`
	Fields []responseField `json:"fields"`
}

type responseField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

func (e commandError) Error() string {
	return e.message
}

func (e commandError) ExitCode() int {
	return e.code
}

func (e commandError) Silent() bool {
	return e.silent
}

func Run(args []string, stdout, stderr io.Writer) error {
	format, err := resolveRequestedFormat(args)
	if err != nil {
		return writeStructuredError(stdout, detectCommandName(args), format, err)
	}

	root := newRootCommand(stdout, stderr, format)
	root.SetArgs(args)
	err = root.Execute()
	if err != nil {
		return writeStructuredError(stdout, detectCommandName(args), format, normalizeCLIError(err))
	}
	return nil
}

func parseCommandOptions(cmd *cobra.Command, args []string, defaultFormat outputFormat, requireInput bool) (commandOptions, error) {
	opts := commandOptions{format: defaultFormat}
	if opts.format == formatDefault {
		opts.format = formatText
	}

	if len(args) > 1 {
		return commandOptions{}, commandError{
			message: "too many positional arguments",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if len(args) == 1 {
		if err := validatePathArg(args[0]); err != nil {
			return commandOptions{}, err
		}
		opts.input = args[0]
	}

	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag != nil && formatFlag.Changed {
		opts.formatExplicit = true
	}

	outputFlag := cmd.Flags().Lookup("output")
	if outputFlag != nil {
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return commandOptions{}, err
		}
		if output != "" {
			if err := validatePathArg(output); err != nil {
				return commandOptions{}, err
			}
			opts.output = output
		}
	}

	if requireInput && opts.input == "" {
		return commandOptions{}, commandError{
			message: "input path is required",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	return opts, nil
}

type namedCommandOptions struct {
	commandOptions
	values map[string]string
}

func parseNamedCommandOptions(cmd *cobra.Command, args []string, defaultFormat outputFormat, requireInput bool) (namedCommandOptions, error) {
	opts := namedCommandOptions{
		commandOptions: commandOptions{format: defaultFormat},
		values:         map[string]string{},
	}
	if opts.format == formatDefault {
		opts.format = formatText
	}

	if len(args) > 1 {
		return namedCommandOptions{}, commandError{
			message: "too many positional arguments",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if len(args) == 1 {
		if err := validatePathArg(args[0]); err != nil {
			return namedCommandOptions{}, err
		}
		opts.input = args[0]
	}

	formatFlag := cmd.Flags().Lookup("format")
	if formatFlag != nil && formatFlag.Changed {
		opts.formatExplicit = true
	}

	outputFlag := cmd.Flags().Lookup("output")
	if outputFlag != nil {
		output, err := cmd.Flags().GetString("output")
		if err != nil {
			return namedCommandOptions{}, err
		}
		if output != "" {
			if err := validatePathArg(output); err != nil {
				return namedCommandOptions{}, err
			}
			opts.output = output
		}
	}

	var flagErr error
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flagErr != nil {
			return
		}
		if flag.Name == "format" || flag.Name == "output" || flag.Name == "help" {
			return
		}
		if !flag.Changed {
			return
		}

		value, err := cmd.Flags().GetString(flag.Name)
		if err != nil {
			flagErr = err
			return
		}
		opts.values[flag.Name] = value
	})
	if flagErr != nil {
		return namedCommandOptions{}, flagErr
	}
	if err := validateNamedCommandValues(cmd, opts.values); err != nil {
		return namedCommandOptions{}, err
	}

	if requireInput && opts.input == "" {
		return namedCommandOptions{}, commandError{
			message: "input path is required",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	return opts, nil
}

func validateNamedCommandValues(cmd *cobra.Command, values map[string]string) error {
	var validationErr error
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if validationErr != nil || !flag.Changed {
			return
		}
		if flag.Name != "image" {
			return
		}
		if err := validatePathArg(values[flag.Name]); err != nil {
			validationErr = err
		}
	})
	return validationErr
}

func resolveRequestedFormat(args []string) (outputFormat, error) {
	envFormat := strings.TrimSpace(os.Getenv(defaultFormatEnvVar))
	if envFormat == "" {
		envFormat = string(formatText)
	}

	resolved, err := parseOutputFormat(envFormat)
	if err != nil {
		return formatDefault, commandError{
			message: fmt.Sprintf("invalid %s value %q", defaultFormatEnvVar, envFormat),
			code:    1,
			kind:    "invalid_environment",
		}
	}

	for index := 0; index < len(args); index++ {
		if args[index] != "--format" {
			continue
		}
		if index+1 >= len(args) {
			return formatDefault, commandError{
				message: "missing value for --format",
				code:    1,
				kind:    "invalid_arguments",
			}
		}
		return parseOutputFormat(args[index+1])
	}

	return resolved, nil
}

func parseOutputFormat(value string) (outputFormat, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "text":
		return formatText, nil
	case "json":
		return formatJSON, nil
	default:
		return formatDefault, commandError{
			message: fmt.Sprintf("unsupported format: %s", value),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
}

func validatePathArg(value string) error {
	for _, char := range value {
		if char == 0 || char == '\n' || char == '\r' {
			return commandError{
				message: "path arguments must not contain control characters",
				code:    1,
				kind:    "invalid_arguments",
			}
		}
	}
	return nil
}

func writeStructuredError(stdout io.Writer, command string, format outputFormat, err error) error {
	if format != formatJSON {
		return err
	}

	code := 1
	kind := defaultErrorKind
	message := err.Error()
	var data any

	var commandErr commandError
	if errors.As(err, &commandErr) {
		code = commandErr.code
		if commandErr.kind != "" {
			kind = commandErr.kind
		}
		if commandErr.message != "" {
			message = commandErr.message
		}
		data = commandErr.data
	}

	writeErr := writeEnvelope(stdout, responseEnvelope{
		SchemaVersion: schemaVersion,
		Command:       command,
		Success:       false,
		Data:          data,
		Error: &responseError{
			Code:    kind,
			Message: message,
		},
	})
	if writeErr != nil {
		return writeErr
	}

	return commandError{code: code, silent: true}
}

func writeEnvelope(stdout io.Writer, envelope responseEnvelope) error {
	return writeJSON(stdout, envelope)
}

func writeJSON(stdout io.Writer, value any) error {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeInspectText(stdout io.Writer, input string, report hwpx.Report) error {
	_, err := fmt.Fprintf(
		stdout,
		"input: %s\nvalid: %t\nentries: %d\nsections: %d\nwarnings: %d\nerrors: %d\n",
		absolutePath(input),
		report.Valid,
		len(report.Summary.Entries),
		len(report.Summary.SectionPath),
		len(report.Warnings),
		len(report.Errors),
	)
	return err
}

func writeValidateText(stdout io.Writer, input string, report hwpx.Report) error {
	status := "valid"
	if !report.Valid {
		status = "invalid"
	}

	if _, err := fmt.Fprintf(
		stdout,
		"input: %s\nstatus: %s\nwarnings: %d\nerrors: %d\n",
		absolutePath(input),
		status,
		len(report.Warnings),
		len(report.Errors),
	); err != nil {
		return err
	}

	for _, warning := range report.Warnings {
		if _, err := fmt.Fprintf(stdout, "warning: %s\n", warning); err != nil {
			return err
		}
	}
	for _, reportErr := range report.Errors {
		if _, err := fmt.Fprintf(stdout, "error: %s\n", reportErr); err != nil {
			return err
		}
	}

	return nil
}

func writeSchemaText(stdout io.Writer, doc schemaDoc) {
	fmt.Fprintf(stdout, "name: %s\nversion: %s\nschemaVersion: %s\n", doc.Name, doc.Version, doc.SchemaVersion)
	for _, command := range doc.Commands {
		fmt.Fprintf(stdout, "command: %s - %s\n", command.Name, command.Summary)
	}
}

func buildSchemaDoc() schemaDoc {
	return decorateSchemaDoc(schemaDoc{
		SchemaVersion: schemaVersion,
		Name:          "hwpxctl",
		Version:       version,
		Environment: []environmentSpec{
			{
				Name:        defaultFormatEnvVar,
				Values:      []string{"text", "json"},
				Default:     "text",
				Description: "Default output mode when --format is omitted.",
			},
		},
		Commands: []commandSpec{
			{
				Name:        "inspect",
				Summary:     "Inspect HWPX metadata, manifest, spine, and section paths.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to a .hwpx file."},
				},
				Options: []optionSpec{
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl inspect ./file.hwpx --format json",
				},
			},
			{
				Name:        "validate",
				Summary:     "Validate a .hwpx file or unpacked directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to a .hwpx file or unpacked directory."},
				},
				Options: []optionSpec{
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl validate ./file.hwpx --format json",
					"hwpxctl validate ./work/unpacked --format json",
				},
			},
			{
				Name:        "text",
				Summary:     "Extract plain text in spine order.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to a .hwpx file."},
				},
				Options: []optionSpec{
					{Name: "--output", Required: false, Description: "Optional text file destination."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl text ./file.hwpx --format json",
					"hwpxctl text ./file.hwpx --output ./out/file.txt --format json",
				},
			},
			{
				Name:        "export-markdown",
				Summary:     "Export paragraphs and tables to Markdown.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to a .hwpx file or unpacked directory."},
				},
				Options: []optionSpec{
					{Name: "--output", Required: false, Description: "Optional Markdown file destination."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl export-markdown ./file.hwpx --format json",
					"hwpxctl export-markdown ./work/unpacked --output ./out/file.md --format json",
				},
			},
			{
				Name:        "export-html",
				Summary:     "Export paragraphs and tables to HTML.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to a .hwpx file or unpacked directory."},
				},
				Options: []optionSpec{
					{Name: "--output", Required: false, Description: "Optional HTML file destination."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl export-html ./file.hwpx --format json",
					"hwpxctl export-html ./work/unpacked --output ./out/file.html --format json",
				},
			},
			{
				Name:        "unpack",
				Summary:     "Unpack a .hwpx file into a directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to a .hwpx file."},
				},
				Options: []optionSpec{
					{Name: "--output", Required: true, Description: "Destination directory."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl unpack ./file.hwpx --output ./work/unpacked --format json",
				},
			},
			{
				Name:        "pack",
				Summary:     "Pack a validated directory into a .hwpx file.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--output", Required: true, Description: "Destination .hwpx file."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl pack ./work/unpacked --output ./out/file.hwpx --format json",
				},
			},
			{
				Name:        "create",
				Summary:     "Create an editable unpacked HWPX directory.",
				JSONCapable: true,
				Options: []optionSpec{
					{Name: "--output", Required: true, Description: "Destination directory."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl create --output ./work/new-doc --format json",
				},
			},
			{
				Name:        "append-text",
				Summary:     "Append one or more paragraphs to the first section in an unpacked directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--text", Required: true, Description: "Paragraph text. Newlines create multiple paragraphs."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl append-text ./work/doc --text \"첫 문단\n둘째 문단\" --format json",
				},
			},
			{
				Name:        "add-run-text",
				Summary:     "Insert one direct text run into an editable paragraph.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--paragraph", Required: true, Description: "Zero-based paragraph index excluding the first section property paragraph."},
					{Name: "--text", Required: true, Description: "Text to insert as a new run."},
					{Name: "--run", Required: false, Description: "Optional zero-based insertion index. Omit to append after the last direct run."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl add-run-text ./work/doc --paragraph 1 --text \"(검토본)\" --format json",
					"hwpxctl add-run-text ./work/doc --paragraph 1 --run 0 --text \"[머리] \" --format json",
				},
			},
			{
				Name:        "set-run-text",
				Summary:     "Replace the text content of one direct run in an editable paragraph.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--paragraph", Required: true, Description: "Zero-based paragraph index excluding the first section property paragraph."},
					{Name: "--run", Required: true, Description: "Zero-based direct run index inside the paragraph."},
					{Name: "--text", Required: true, Description: "Replacement text for the run."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl set-run-text ./work/doc --paragraph 1 --run 0 --text \"[최종]\" --format json",
				},
			},
			{
				Name:        "find-runs-by-style",
				Summary:     "Find direct runs in the first section that match style conditions.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--bold", Required: false, Description: "Match bold true/false."},
					{Name: "--italic", Required: false, Description: "Match italic true/false."},
					{Name: "--underline", Required: false, Description: "Match underline true/false."},
					{Name: "--text-color", Required: false, Description: "Match text color as #RRGGBB."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl find-runs-by-style ./work/doc --bold true --format json",
					"hwpxctl find-runs-by-style ./work/doc --underline true --text-color \"#C00000\" --format json",
				},
			},
			{
				Name:        "replace-runs-by-style",
				Summary:     "Replace text in direct runs that match style conditions.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--text", Required: true, Description: "Replacement text for every matching run."},
					{Name: "--bold", Required: false, Description: "Match bold true/false."},
					{Name: "--italic", Required: false, Description: "Match italic true/false."},
					{Name: "--underline", Required: false, Description: "Match underline true/false."},
					{Name: "--text-color", Required: false, Description: "Match text color as #RRGGBB."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl replace-runs-by-style ./work/doc --bold true --text \"[강조]\" --format json",
					"hwpxctl replace-runs-by-style ./work/doc --underline true --text-color \"#C00000\" --text \"*검토 메모*\" --format json",
				},
			},
			{
				Name:        "find-objects",
				Summary:     "List high-level objects found in the first section.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--type", Required: false, Description: "Optional comma-separated object types: table,image,equation,rectangle,line,ellipse,textbox."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl find-objects ./work/doc --format json",
					"hwpxctl find-objects ./work/doc --type table,textbox --format json",
				},
			},
			{
				Name:        "find-by-tag",
				Summary:     "Find elements in the first section by XML tag.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--tag", Required: true, Description: "Target XML tag. Accepts hp:tbl or tbl forms."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl find-by-tag ./work/doc --tag hp:tbl --format json",
					"hwpxctl find-by-tag ./work/doc --tag drawText --format json",
				},
			},
			{
				Name:        "find-by-attr",
				Summary:     "Find elements in the first section by XML attribute.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--attr", Required: true, Description: "Target attribute name. Accepts names with or without prefix."},
					{Name: "--value", Required: false, Description: "Optional exact attribute value filter."},
					{Name: "--tag", Required: false, Description: "Optional tag filter to narrow matching elements."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl find-by-attr ./work/doc --attr id --tag tbl --format json",
					"hwpxctl find-by-attr ./work/doc --attr editable --tag drawText --value 0 --format json",
				},
			},
			{
				Name:        "find-by-xpath",
				Summary:     "Find elements in the first section with an etree XPath-like expression.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--expr", Required: true, Description: "XPath-like expression evaluated from the first section root."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl find-by-xpath ./work/doc --expr \".//hp:tbl[@id]\" --format json",
					"hwpxctl find-by-xpath ./work/doc --expr \".//hp:drawText[@editable='0']\" --format json",
				},
			},
			{
				Name:        "set-paragraph-text",
				Summary:     "Replace the text of one editable paragraph in the first section.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--paragraph", Required: true, Description: "Zero-based paragraph index excluding the first section property paragraph."},
					{Name: "--text", Required: true, Description: "Replacement paragraph text."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl set-paragraph-text ./work/doc --paragraph 1 --text \"수정된 문단\" --format json",
				},
			},
			{
				Name:        "set-paragraph-layout",
				Summary:     "Update paragraph alignment, indentation, and spacing in the first section.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--paragraph", Required: true, Description: "Zero-based paragraph index excluding the first section property paragraph."},
					{Name: "--align", Required: false, Description: "Horizontal alignment: LEFT, CENTER, RIGHT, JUSTIFY, DISTRIBUTE."},
					{Name: "--indent-mm", Required: false, Description: "First-line indent in millimeters."},
					{Name: "--left-margin-mm", Required: false, Description: "Left margin in millimeters."},
					{Name: "--right-margin-mm", Required: false, Description: "Right margin in millimeters."},
					{Name: "--space-before-mm", Required: false, Description: "Paragraph spacing before in millimeters."},
					{Name: "--space-after-mm", Required: false, Description: "Paragraph spacing after in millimeters."},
					{Name: "--line-spacing-percent", Required: false, Description: "Line spacing percent. Example: 160."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl set-paragraph-layout ./work/doc --paragraph 1 --align CENTER --space-after-mm 4 --format json",
					"hwpxctl set-paragraph-layout ./work/doc --paragraph 2 --indent-mm 4 --left-margin-mm 8 --line-spacing-percent 180 --format json",
				},
			},
			{
				Name:        "set-paragraph-list",
				Summary:     "Apply bullet or numbering to one editable paragraph in the first section.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--paragraph", Required: true, Description: "Zero-based paragraph index excluding the first section property paragraph."},
					{Name: "--kind", Required: true, Description: "List kind: bullet, number, or none."},
					{Name: "--level", Required: false, Description: "Zero-based nesting level. Defaults to 0."},
					{Name: "--start-number", Required: false, Description: "Optional numbering start value. Number lists only."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl set-paragraph-list ./work/doc --paragraph 1 --kind bullet --format json",
					"hwpxctl set-paragraph-list ./work/doc --paragraph 2 --kind number --level 1 --start-number 3 --format json",
				},
			},
			{
				Name:        "set-text-style",
				Summary:     "Apply text style changes to one run or all runs in an editable paragraph.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--paragraph", Required: true, Description: "Zero-based paragraph index excluding the first section property paragraph."},
					{Name: "--run", Required: false, Description: "Optional zero-based run index inside the paragraph. Omit to update all runs."},
					{Name: "--bold", Required: false, Description: "Set bold on or off with true/false."},
					{Name: "--italic", Required: false, Description: "Set italic on or off with true/false."},
					{Name: "--underline", Required: false, Description: "Set underline on or off with true/false."},
					{Name: "--text-color", Required: false, Description: "Set text color as #RRGGBB."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl set-text-style ./work/doc --paragraph 1 --bold true --underline true --format json",
					"hwpxctl set-text-style ./work/doc --paragraph 1 --run 0 --italic true --text-color \"#C00000\" --format json",
				},
			},
			{
				Name:        "delete-paragraph",
				Summary:     "Delete one editable paragraph from the first section.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--paragraph", Required: true, Description: "Zero-based paragraph index excluding the first section property paragraph."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl delete-paragraph ./work/doc --paragraph 1 --format json",
				},
			},
			{
				Name:        "add-section",
				Summary:     "Append one empty section to an unpacked directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl add-section ./work/doc --format json",
				},
			},
			{
				Name:        "delete-section",
				Summary:     "Delete one section by spine order from an unpacked directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--section", Required: true, Description: "Zero-based section index in spine order."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl delete-section ./work/doc --section 1 --format json",
				},
			},
			{
				Name:        "add-table",
				Summary:     "Append a table to the first section in an unpacked directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--rows", Required: false, Description: "Table row count. Inferred from --cells when omitted."},
					{Name: "--cols", Required: false, Description: "Table column count. Inferred from --cells when omitted."},
					{Name: "--cells", Required: false, Description: "Semicolon/comma matrix. Example: a,b;c,d"},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl add-table ./work/doc --cells \"항목,내용;이름,홍길동\" --format json",
				},
			},
			{
				Name:        "add-nested-table",
				Summary:     "Insert a nested table into an existing table cell in the first section.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--table", Required: true, Description: "Zero-based parent table index."},
					{Name: "--row", Required: true, Description: "Zero-based parent cell row index."},
					{Name: "--col", Required: true, Description: "Zero-based parent cell column index."},
					{Name: "--rows", Required: false, Description: "Nested table row count. Inferred from --cells when omitted."},
					{Name: "--cols", Required: false, Description: "Nested table column count. Inferred from --cells when omitted."},
					{Name: "--cells", Required: false, Description: "Semicolon/comma matrix. Example: a,b;c,d"},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl add-nested-table ./work/doc --table 0 --row 1 --col 1 --cells \"내부1,내부2;내부3,내부4\" --format json",
				},
			},
			{
				Name:        "set-table-cell",
				Summary:     "Update a cell in the first section of an unpacked directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--table", Required: true, Description: "Zero-based table index."},
					{Name: "--row", Required: true, Description: "Zero-based row index."},
					{Name: "--col", Required: true, Description: "Zero-based column index."},
					{Name: "--text", Required: true, Description: "Cell text."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl set-table-cell ./work/doc --table 0 --row 1 --col 1 --text \"수정값\" --format json",
				},
			},
			{
				Name:        "merge-table-cells",
				Summary:     "Merge a rectangular region of cells in the first section table.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--table", Required: true, Description: "Zero-based table index."},
					{Name: "--start-row", Required: true, Description: "Top row of the merge rectangle."},
					{Name: "--start-col", Required: true, Description: "Left column of the merge rectangle."},
					{Name: "--end-row", Required: true, Description: "Bottom row of the merge rectangle."},
					{Name: "--end-col", Required: true, Description: "Right column of the merge rectangle."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl merge-table-cells ./work/doc --table 0 --start-row 0 --start-col 0 --end-row 1 --end-col 1 --format json",
				},
			},
			{
				Name:        "split-table-cell",
				Summary:     "Split a merged cell back into individual cells in the first section table.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--table", Required: true, Description: "Zero-based table index."},
					{Name: "--row", Required: true, Description: "Logical row index."},
					{Name: "--col", Required: true, Description: "Logical column index."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl split-table-cell ./work/doc --table 0 --row 0 --col 0 --format json",
				},
			},
			{
				Name:        "embed-image",
				Summary:     "Embed an image asset into an unpacked directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--image", Required: true, Description: "Path to a PNG/JPG/GIF/BMP/SVG file."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl embed-image ./work/doc --image ./assets/logo.png --format json",
				},
			},
			{
				Name:        "insert-image",
				Summary:     "Embed an image and place a visible picture in the first section of an unpacked directory.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--image", Required: true, Description: "Path to a PNG/JPG/GIF file."},
					{Name: "--width-mm", Required: false, Description: "Optional rendered width in millimeters."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl insert-image ./work/doc --image ./assets/logo.png --width-mm 80 --format json",
				},
			},
			{
				Name:        "set-object-position",
				Summary:     "Update image or shape positioning in the first section.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--type", Required: true, Description: "Target object type: image, rectangle, line, ellipse, textbox."},
					{Name: "--index", Required: true, Description: "Zero-based index within the target object type."},
					{Name: "--treat-as-char", Required: false, Description: "Set inline placement on or off with true/false."},
					{Name: "--x-mm", Required: false, Description: "Horizontal offset in millimeters."},
					{Name: "--y-mm", Required: false, Description: "Vertical offset in millimeters."},
					{Name: "--horz-align", Required: false, Description: "Horizontal alignment: LEFT, CENTER, RIGHT."},
					{Name: "--vert-align", Required: false, Description: "Vertical alignment: TOP, CENTER, BOTTOM."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl set-object-position ./work/doc --type image --index 0 --treat-as-char false --x-mm 10 --y-mm 6 --format json",
					"hwpxctl set-object-position ./work/doc --type textbox --index 0 --horz-align CENTER --vert-align TOP --format json",
				},
			},
			{
				Name:    "set-header",
				Summary: "Set header text in the first section of an unpacked directory.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--text", Required: true, Description: "Header text. Newlines create multiple paragraphs."},
					{Name: "--apply-page-type", Required: false, Description: "Page range selector: BOTH, EVEN, ODD."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl set-header ./work/doc --text \"문서 제목\" --format json",
					"hwpxctl set-header ./work/doc --text \"문서 제목 {{PAGE}} / {{TOTAL_PAGE}}\" --format json",
				},
			},
			{
				Name:    "set-footer",
				Summary: "Set footer text in the first section of an unpacked directory.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--text", Required: true, Description: "Footer text. Newlines create multiple paragraphs."},
					{Name: "--apply-page-type", Required: false, Description: "Page range selector: BOTH, EVEN, ODD."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl set-footer ./work/doc --text \"기관명\" --format json",
					"hwpxctl set-footer ./work/doc --text \"- {{PAGE}} / {{TOTAL_PAGE}} -\" --format json",
				},
			},
			{
				Name:    "remove-header",
				Summary: "Remove header content from the first section of an unpacked directory.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl remove-header ./work/doc --format json",
				},
			},
			{
				Name:    "remove-footer",
				Summary: "Remove footer content from the first section of an unpacked directory.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl remove-footer ./work/doc --format json",
				},
			},
			{
				Name:    "set-page-number",
				Summary: "Set page number display in the first section of an unpacked directory.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--position", Required: false, Description: "Page number position. Example: BOTTOM_CENTER."},
					{Name: "--type", Required: false, Description: "Number format. Example: DIGIT."},
					{Name: "--side-char", Required: false, Description: "Optional wrapper character."},
					{Name: "--start-page", Required: false, Description: "Optional first page number."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl set-page-number ./work/doc --position BOTTOM_CENTER --type DIGIT --start-page 1 --format json",
				},
			},
			{
				Name:    "set-columns",
				Summary: "Set multi-column layout in the first section of an unpacked directory.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--count", Required: true, Description: "Number of columns. Minimum 1."},
					{Name: "--gap-mm", Required: false, Description: "Optional gap between columns in millimeters."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl set-columns ./work/doc --count 2 --gap-mm 8 --format json",
				},
			},
			{
				Name:    "add-footnote",
				Summary: "Append a paragraph with a footnote anchor and body in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--anchor-text", Required: true, Description: "Visible body text that owns the footnote anchor."},
					{Name: "--text", Required: true, Description: "Footnote body text. Newlines create multiple note paragraphs."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-footnote ./work/doc --anchor-text \"본문 설명\" --text \"각주 내용\" --format json",
				},
			},
			{
				Name:    "add-endnote",
				Summary: "Append a paragraph with an endnote anchor and body in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--anchor-text", Required: true, Description: "Visible body text that owns the endnote anchor."},
					{Name: "--text", Required: true, Description: "Endnote body text. Newlines create multiple note paragraphs."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-endnote ./work/doc --anchor-text \"본문 설명\" --text \"미주 내용\" --format json",
				},
			},
			{
				Name:    "add-memo",
				Summary: "Append a paragraph with a memo anchor and memo body in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--anchor-text", Required: true, Description: "Visible body text that owns the memo marker."},
					{Name: "--text", Required: true, Description: "Memo body text. Newlines create multiple memo paragraphs."},
					{Name: "--author", Required: false, Description: "Optional memo author name stored in field parameters."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-memo ./work/doc --anchor-text \"검토가 필요한 문장\" --text \"메모 내용\" --author \"홍길동\" --format json",
				},
			},
			{
				Name:    "add-bookmark",
				Summary: "Append a paragraph with a bookmark marker in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--name", Required: true, Description: "Bookmark identifier."},
					{Name: "--text", Required: true, Description: "Visible paragraph text for the bookmark location."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-bookmark ./work/doc --name intro --text \"소개 문단\" --format json",
				},
			},
			{
				Name:    "add-hyperlink",
				Summary: "Append a paragraph with a hyperlink in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--target", Required: true, Description: "URL or internal bookmark target. Example: https://example.com or #intro."},
					{Name: "--text", Required: true, Description: "Visible hyperlink text."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-hyperlink ./work/doc --target https://example.com --text \"외부 링크\" --format json",
					"hwpxctl add-hyperlink ./work/doc --target #intro --text \"소개로 이동\" --format json",
				},
			},
			{
				Name:    "add-heading",
				Summary: "Append a title, heading, or outline paragraph in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--kind", Required: false, Description: "Paragraph kind: title, heading, or outline. Defaults to heading."},
					{Name: "--level", Required: false, Description: "Heading or outline level. Required for heading/outline styles."},
					{Name: "--text", Required: true, Description: "Visible paragraph text."},
					{Name: "--bookmark", Required: false, Description: "Optional bookmark name. Generated automatically when omitted."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-heading ./work/doc --kind heading --level 1 --text \"소개\" --format json",
					"hwpxctl add-heading ./work/doc --kind outline --level 2 --text \"세부 항목\" --format json",
				},
			},
			{
				Name:    "insert-toc",
				Summary: "Insert a basic table of contents from heading and outline paragraphs.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--title", Required: false, Description: "Optional TOC title. Defaults to 목차."},
					{Name: "--max-level", Required: false, Description: "Maximum heading level to include. Defaults to 3."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl insert-toc ./work/doc --title \"목차\" --max-level 3 --format json",
				},
			},
			{
				Name:    "add-cross-reference",
				Summary: "Append a bookmark-based internal reference paragraph in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--bookmark", Required: true, Description: "Target bookmark name."},
					{Name: "--text", Required: false, Description: "Optional visible reference text. Falls back to the target paragraph text."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-cross-reference ./work/doc --bookmark heading-2 --text \"소개로 이동\" --format json",
				},
			},
			{
				Name:    "add-equation",
				Summary: "Append an equation object paragraph in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--script", Required: true, Description: "Hangul equation script text. Example: alpha over beta or a+b."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-equation ./work/doc --script \"a+b\" --format json",
				},
			},
			{
				Name:    "add-line",
				Summary: "Append a basic line drawing object in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--width-mm", Required: true, Description: "Line width in millimeters."},
					{Name: "--height-mm", Required: true, Description: "Line height in millimeters."},
					{Name: "--line-color", Required: false, Description: "Optional stroke color. Example: #000000."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-line ./work/doc --width-mm 50 --height-mm 10 --line-color \"#2F5597\" --format json",
				},
			},
			{
				Name:    "add-ellipse",
				Summary: "Append a basic ellipse drawing object in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--width-mm", Required: true, Description: "Ellipse width in millimeters."},
					{Name: "--height-mm", Required: true, Description: "Ellipse height in millimeters."},
					{Name: "--line-color", Required: false, Description: "Optional stroke color. Example: #000000."},
					{Name: "--fill-color", Required: false, Description: "Optional fill color. Example: #FFF2CC."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-ellipse ./work/doc --width-mm 40 --height-mm 20 --fill-color \"#FFF2CC\" --format json",
				},
			},
			{
				Name:    "add-rectangle",
				Summary: "Append a basic rectangle drawing object in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--width-mm", Required: true, Description: "Rectangle width in millimeters."},
					{Name: "--height-mm", Required: true, Description: "Rectangle height in millimeters."},
					{Name: "--line-color", Required: false, Description: "Optional stroke color. Example: #000000."},
					{Name: "--fill-color", Required: false, Description: "Optional fill color. Example: #FFF2CC."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-rectangle ./work/doc --width-mm 40 --height-mm 20 --fill-color \"#FFF2CC\" --format json",
				},
			},
			{
				Name:    "add-textbox",
				Summary: "Append a basic textbox drawing object in the first section.",
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to an unpacked HWPX directory."},
				},
				Options: []optionSpec{
					{Name: "--width-mm", Required: true, Description: "Textbox width in millimeters."},
					{Name: "--height-mm", Required: true, Description: "Textbox height in millimeters."},
					{Name: "--text", Required: true, Description: "Textbox body text. Newlines create multiple paragraphs."},
					{Name: "--line-color", Required: false, Description: "Optional stroke color. Example: #000000."},
					{Name: "--fill-color", Required: false, Description: "Optional fill color. Example: #FFFFFF."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				JSONCapable: true,
				Examples: []string{
					"hwpxctl add-textbox ./work/doc --width-mm 60 --height-mm 25 --text \"글상자 본문\" --format json",
				},
			},
			{
				Name:        "schema",
				Summary:     "Print machine-readable command metadata.",
				JSONCapable: true,
				Options: []optionSpec{
					{Name: "--format", Values: []string{"text", "json"}, Description: "Defaults to JSON for this command when omitted."},
				},
				Examples: []string{
					"hwpxctl schema",
					"hwpxctl schema --format text",
				},
			},
		},
		Response: responseSpec{
			Format: "JSON envelope",
			Fields: []responseField{
				{Name: "schemaVersion", Type: "string", Description: "Response contract version."},
				{Name: "command", Type: "string", Description: "Executed command name."},
				{Name: "success", Type: "boolean", Description: "Whether the command succeeded."},
				{Name: "data", Type: "object", Description: "Command-specific payload."},
				{Name: "error", Type: "object", Description: "Structured error payload when success=false."},
			},
		},
	})
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

func fallbackCLIString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func maxIntCLI(left, right int) int {
	if left >= right {
		return left
	}
	return right
}

func parseOptionalIntArg(values map[string]string, key string) (int, error) {
	value, ok := values[key]
	if !ok || strings.TrimSpace(value) == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, commandError{
			message: fmt.Sprintf("invalid integer for --%s: %s", key, value),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return parsed, nil
}

func parseOptionalFloatArg(values map[string]string, key string) (float64, error) {
	value, ok := values[key]
	if !ok || strings.TrimSpace(value) == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, commandError{
			message: fmt.Sprintf("invalid number for --%s: %s", key, value),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return parsed, nil
}

func optionalFloatPointer(values map[string]string, key string) (*float64, error) {
	if _, ok := values[key]; !ok {
		return nil, nil
	}
	value, err := parseOptionalFloatArg(values, key)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func parseOptionalBoolArg(values map[string]string, key string) (*bool, error) {
	value, ok := values[key]
	if !ok || strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, commandError{
			message: fmt.Sprintf("invalid boolean for --%s: %s", key, value),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return &parsed, nil
}

func parseOptionalColorArg(values map[string]string, key string) (string, error) {
	value, ok := values[key]
	if !ok || strings.TrimSpace(value) == "" {
		return "", nil
	}

	normalized := strings.ToUpper(strings.TrimSpace(value))
	if len(normalized) != 7 || !strings.HasPrefix(normalized, "#") {
		return "", commandError{
			message: fmt.Sprintf("invalid color for --%s: %s", key, value),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	for _, ch := range normalized[1:] {
		if (ch < '0' || ch > '9') && (ch < 'A' || ch > 'F') {
			return "", commandError{
				message: fmt.Sprintf("invalid color for --%s: %s", key, value),
				code:    1,
				kind:    "invalid_arguments",
			}
		}
	}
	return normalized, nil
}

func optionalIntPointer(values map[string]string, key string) (*int, error) {
	if _, ok := values[key]; !ok {
		return nil, nil
	}
	value, err := requireIntArg(values, key)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func requireIntArg(values map[string]string, key string) (int, error) {
	if _, ok := values[key]; !ok {
		return 0, commandError{
			message: fmt.Sprintf("missing --%s", key),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return parseOptionalIntArg(values, key)
}

func isAllowedValue(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func splitParagraphs(text string) []string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.Split(normalized, "\n")
}

func parseCellMatrix(raw string) [][]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	rows := strings.Split(raw, ";")
	matrix := make([][]string, 0, len(rows))
	for _, row := range rows {
		cells := strings.Split(row, ",")
		for index := range cells {
			cells[index] = strings.TrimSpace(cells[index])
		}
		matrix = append(matrix, cells)
	}
	return matrix
}

func absolutePath(value string) string {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return absolute
}
