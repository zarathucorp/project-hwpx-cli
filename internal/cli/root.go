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

type paragraphUpdateResult struct {
	InputPath    string      `json:"inputPath"`
	Paragraph    int         `json:"paragraph"`
	PreviousText string      `json:"previousText,omitempty"`
	RemovedText  string      `json:"removedText,omitempty"`
	Deleted      bool        `json:"deleted"`
	Report       hwpx.Report `json:"report"`
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
		return writeStructuredError(stdout, "", format, err)
	}

	if len(args) == 0 {
		writeHelp(stdout)
		return nil
	}

	switch args[0] {
	case "-h", "--help":
		writeHelp(stdout)
		return nil
	case "-v", "--version":
		_, err := fmt.Fprintln(stdout, version)
		return err
	}

	command := args[0]
	rest := args[1:]

	switch command {
	case "inspect":
		err = runInspect(rest, stdout, format)
	case "validate":
		err = runValidate(rest, stdout, format)
	case "text":
		err = runText(rest, stdout, format)
	case "unpack":
		err = runUnpack(rest, stdout, format)
	case "pack":
		err = runPack(rest, stdout, format)
	case "create":
		err = runCreate(rest, stdout, format)
	case "append-text":
		err = runAppendText(rest, stdout, format)
	case "set-paragraph-text":
		err = runSetParagraphText(rest, stdout, format)
	case "delete-paragraph":
		err = runDeleteParagraph(rest, stdout, format)
	case "add-section":
		err = runAddSection(rest, stdout, format)
	case "delete-section":
		err = runDeleteSection(rest, stdout, format)
	case "add-table":
		err = runAddTable(rest, stdout, format)
	case "set-table-cell":
		err = runSetTableCell(rest, stdout, format)
	case "merge-table-cells":
		err = runMergeTableCells(rest, stdout, format)
	case "split-table-cell":
		err = runSplitTableCell(rest, stdout, format)
	case "embed-image":
		err = runEmbedImage(rest, stdout, format)
	case "insert-image":
		err = runInsertImage(rest, stdout, format)
	case "set-header":
		err = runSetHeader(rest, stdout, format)
	case "set-footer":
		err = runSetFooter(rest, stdout, format)
	case "set-page-number":
		err = runSetPageNumber(rest, stdout, format)
	case "add-footnote":
		err = runAddNote("footnote", rest, stdout, format)
	case "add-endnote":
		err = runAddNote("endnote", rest, stdout, format)
	case "add-memo":
		err = runAddMemo(rest, stdout, format)
	case "add-bookmark":
		err = runAddBookmark(rest, stdout, format)
	case "add-hyperlink":
		err = runAddHyperlink(rest, stdout, format)
	case "add-heading":
		err = runAddHeading(rest, stdout, format)
	case "insert-toc":
		err = runInsertTOC(rest, stdout, format)
	case "add-cross-reference":
		err = runAddCrossReference(rest, stdout, format)
	case "add-equation":
		err = runAddEquation(rest, stdout, format)
	case "add-rectangle":
		err = runAddRectangle(rest, stdout, format)
	case "print-pdf":
		err = runPrintPDF(rest, stdout, format)
	case "schema":
		err = runSchema(rest, stdout, format)
	default:
		err = commandError{
			message: fmt.Sprintf("unknown command: %s", command),
			code:    1,
			kind:    "unknown_command",
		}
	}

	if err != nil {
		return writeStructuredError(stdout, command, format, err)
	}
	return nil
}

func parseCommandOptions(args []string, defaultFormat outputFormat, requireInput bool) (commandOptions, error) {
	opts := commandOptions{format: defaultFormat}
	if opts.format == formatDefault {
		opts.format = formatText
	}

	for index := 0; index < len(args); index++ {
		current := args[index]

		switch current {
		case "-o", "--output":
			if index+1 >= len(args) {
				return commandOptions{}, commandError{
					message: "missing value for --output",
					code:    1,
					kind:    "invalid_arguments",
				}
			}
			if err := validatePathArg(args[index+1]); err != nil {
				return commandOptions{}, err
			}
			opts.output = args[index+1]
			index++
		case "--format":
			if index+1 >= len(args) {
				return commandOptions{}, commandError{
					message: "missing value for --format",
					code:    1,
					kind:    "invalid_arguments",
				}
			}
			format, err := parseOutputFormat(args[index+1])
			if err != nil {
				return commandOptions{}, err
			}
			opts.format = format
			opts.formatExplicit = true
			index++
		case "-h", "--help":
			return commandOptions{}, commandError{
				message: "subcommand help is not implemented; use --help",
				code:    1,
				kind:    "invalid_arguments",
			}
		default:
			if strings.HasPrefix(current, "-") {
				return commandOptions{}, commandError{
					message: fmt.Sprintf("unknown option: %s", current),
					code:    1,
					kind:    "invalid_arguments",
				}
			}
			if err := validatePathArg(current); err != nil {
				return commandOptions{}, err
			}
			if opts.input != "" {
				return commandOptions{}, commandError{
					message: "too many positional arguments",
					code:    1,
					kind:    "invalid_arguments",
				}
			}
			opts.input = current
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

func parseNamedCommandOptions(args []string, defaultFormat outputFormat, requireInput bool) (namedCommandOptions, error) {
	opts := namedCommandOptions{
		commandOptions: commandOptions{format: defaultFormat},
		values:         map[string]string{},
	}
	if opts.format == formatDefault {
		opts.format = formatText
	}

	for index := 0; index < len(args); index++ {
		current := args[index]

		switch current {
		case "-o", "--output":
			if index+1 >= len(args) {
				return namedCommandOptions{}, commandError{
					message: "missing value for --output",
					code:    1,
					kind:    "invalid_arguments",
				}
			}
			if err := validatePathArg(args[index+1]); err != nil {
				return namedCommandOptions{}, err
			}
			opts.output = args[index+1]
			index++
		case "--format":
			if index+1 >= len(args) {
				return namedCommandOptions{}, commandError{
					message: "missing value for --format",
					code:    1,
					kind:    "invalid_arguments",
				}
			}
			format, err := parseOutputFormat(args[index+1])
			if err != nil {
				return namedCommandOptions{}, err
			}
			opts.format = format
			opts.formatExplicit = true
			index++
		case "-h", "--help":
			return namedCommandOptions{}, commandError{
				message: "subcommand help is not implemented; use --help",
				code:    1,
				kind:    "invalid_arguments",
			}
		default:
			if strings.HasPrefix(current, "--") {
				if index+1 >= len(args) {
					return namedCommandOptions{}, commandError{
						message: fmt.Sprintf("missing value for %s", current),
						code:    1,
						kind:    "invalid_arguments",
					}
				}
				if current == "--image" {
					if err := validatePathArg(args[index+1]); err != nil {
						return namedCommandOptions{}, err
					}
				}
				opts.values[strings.TrimPrefix(current, "--")] = args[index+1]
				index++
				continue
			}
			if strings.HasPrefix(current, "-") {
				return namedCommandOptions{}, commandError{
					message: fmt.Sprintf("unknown option: %s", current),
					code:    1,
					kind:    "invalid_arguments",
				}
			}
			if err := validatePathArg(current); err != nil {
				return namedCommandOptions{}, err
			}
			if opts.input != "" {
				return namedCommandOptions{}, commandError{
					message: "too many positional arguments",
					code:    1,
					kind:    "invalid_arguments",
				}
			}
			opts.input = current
		}
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

func writeHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, `hwpxctl

Usage:
  hwpxctl inspect <file.hwpx> [--format text|json]
  hwpxctl validate <file.hwpx|directory> [--format text|json]
  hwpxctl text <file.hwpx> [--output <file.txt>] [--format text|json]
  hwpxctl unpack <file.hwpx> --output <directory> [--format text|json]
  hwpxctl pack <directory> --output <file.hwpx> [--format text|json]
  hwpxctl create --output <directory> [--format text|json]
  hwpxctl append-text <directory> --text <text> [--format text|json]
  hwpxctl set-paragraph-text <directory> --paragraph <n> --text <text> [--format text|json]
  hwpxctl delete-paragraph <directory> --paragraph <n> [--format text|json]
  hwpxctl add-section <directory> [--format text|json]
  hwpxctl delete-section <directory> --section <n> [--format text|json]
  hwpxctl add-table <directory> [--rows <n>] [--cols <n>] [--cells <r1c1,r1c2;r2c1,r2c2>] [--format text|json]
  hwpxctl set-table-cell <directory> --table <n> --row <n> --col <n> --text <text> [--format text|json]
  hwpxctl merge-table-cells <directory> --table <n> --start-row <n> --start-col <n> --end-row <n> --end-col <n> [--format text|json]
  hwpxctl split-table-cell <directory> --table <n> --row <n> --col <n> [--format text|json]
  hwpxctl embed-image <directory> --image <file> [--format text|json]
  hwpxctl insert-image <directory> --image <file> [--width-mm <n>] [--format text|json]
  hwpxctl set-header <directory> --text <text> [--apply-page-type <BOTH|EVEN|ODD>] [--format text|json]
  hwpxctl set-footer <directory> --text <text> [--apply-page-type <BOTH|EVEN|ODD>] [--format text|json]
  hwpxctl set-page-number <directory> [--position <pos>] [--type <fmt>] [--side-char <char>] [--start-page <n>] [--format text|json]
  hwpxctl add-footnote <directory> --anchor-text <text> --text <text> [--format text|json]
  hwpxctl add-endnote <directory> --anchor-text <text> --text <text> [--format text|json]
  hwpxctl add-memo <directory> --anchor-text <text> --text <text> [--author <text>] [--format text|json]
  hwpxctl add-bookmark <directory> --name <name> --text <text> [--format text|json]
  hwpxctl add-hyperlink <directory> --target <url|#bookmark> --text <text> [--format text|json]
  hwpxctl add-heading <directory> --kind <title|heading|outline> --text <text> [--level <n>] [--bookmark <name>] [--format text|json]
  hwpxctl insert-toc <directory> [--title <text>] [--max-level <n>] [--format text|json]
  hwpxctl add-cross-reference <directory> --bookmark <name> [--text <text>] [--format text|json]
  hwpxctl add-equation <directory> --script <text> [--format text|json]
  hwpxctl add-rectangle <directory> --width-mm <n> --height-mm <n> [--line-color <hex>] [--fill-color <hex>] [--format text|json]
  hwpxctl print-pdf <file.hwpx> --output <file.pdf> [--format text|json]
  hwpxctl schema [--format text|json]

Options:
  --format <text|json>  Output mode for agent or human consumers
  -o, --output <path>   Write result to a file or directory
  -h, --help            Show help
  -v, --version         Show version

Environment:
  HWPXCTL_FORMAT        Default output mode when --format is omitted`)
}

func buildSchemaDoc() schemaDoc {
	return schemaDoc{
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
				Name:        "print-pdf",
				Summary:     "Render a .hwpx file through Hancom Viewer and save it as PDF on macOS.",
				JSONCapable: true,
				Arguments: []argument{
					{Name: "input", Required: true, Description: "Path to a .hwpx file."},
				},
				Options: []optionSpec{
					{Name: "--output", Required: true, Description: "Destination .pdf file."},
					{Name: "--format", Values: []string{"text", "json"}, Description: "Selects human or machine-readable output."},
				},
				Examples: []string{
					"hwpxctl print-pdf ./out/doc.hwpx --output ./out/doc.print.pdf --format json",
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
	}
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
