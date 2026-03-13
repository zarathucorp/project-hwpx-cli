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
	"unicode/utf8"

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
	case "add-table":
		err = runAddTable(rest, stdout, format)
	case "set-table-cell":
		err = runSetTableCell(rest, stdout, format)
	case "embed-image":
		err = runEmbedImage(rest, stdout, format)
	case "insert-image":
		err = runInsertImage(rest, stdout, format)
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

func runInspect(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	report, err := hwpx.Inspect(opts.input)
	if err != nil {
		return err
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "inspect",
			Success:       true,
			Data: inspectResult{
				InputPath: absolutePath(opts.input),
				Report:    report,
			},
		})
	case formatText:
		return writeInspectText(stdout, opts.input, report)
	default:
		return writeJSON(stdout, report)
	}
}

func runValidate(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	report, err := hwpx.Validate(opts.input)
	if err != nil {
		return err
	}

	switch opts.format {
	case formatJSON:
		if !report.Valid {
			return commandError{
				message: "validation failed",
				code:    1,
				kind:    "validation_failed",
				data: validateResult{
					InputPath: absolutePath(opts.input),
					Report:    report,
				},
			}
		}

		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "validate",
			Success:       true,
			Data: validateResult{
				InputPath: absolutePath(opts.input),
				Report:    report,
			},
		})
	case formatText:
		if err := writeValidateText(stdout, opts.input, report); err != nil {
			return err
		}
	default:
		if err := writeJSON(stdout, report); err != nil {
			return err
		}
	}

	if !report.Valid {
		return commandError{
			message: "validation failed",
			code:    1,
			kind:    "validation_failed",
		}
	}
	return nil
}

func runText(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	text, err := hwpx.ExtractText(opts.input)
	if err != nil {
		return err
	}

	if opts.output != "" {
		if err := os.MkdirAll(filepath.Dir(opts.output), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(opts.output, []byte(text), 0o644); err != nil {
			return err
		}
	}

	switch opts.format {
	case formatJSON:
		result := textResult{
			InputPath:      absolutePath(opts.input),
			LineCount:      countLines(text),
			CharacterCount: utf8.RuneCountInString(text),
		}
		if opts.output != "" {
			result.OutputPath = absolutePath(opts.output)
		} else {
			result.Text = text
		}
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "text",
			Success:       true,
			Data:          result,
		})
	default:
		if opts.output == "" {
			_, err = fmt.Fprintln(stdout, text)
			return err
		}
		return nil
	}
}

func runUnpack(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}
	if opts.output == "" {
		return commandError{
			message: "unpack requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	if err := hwpx.Unpack(opts.input, opts.output); err != nil {
		return err
	}

	if opts.format == formatJSON {
		report, err := hwpx.Validate(opts.output)
		if err != nil {
			return err
		}
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "unpack",
			Success:       true,
			Data: unpackResult{
				InputPath:  absolutePath(opts.input),
				OutputPath: absolutePath(opts.output),
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Unpacked to %s\n", opts.output)
	return err
}

func runPack(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}
	if opts.output == "" {
		return commandError{
			message: "pack requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	if err := hwpx.Pack(opts.input, opts.output); err != nil {
		return err
	}

	if opts.format == formatJSON {
		report, err := hwpx.Validate(opts.output)
		if err != nil {
			return err
		}
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "pack",
			Success:       true,
			Data: packResult{
				InputPath:  absolutePath(opts.input),
				OutputPath: absolutePath(opts.output),
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Packed to %s\n", opts.output)
	return err
}

func runSchema(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(args, defaultFormat, false)
	if err != nil {
		return err
	}
	if !opts.formatExplicit {
		opts.format = formatJSON
	}
	if opts.input != "" {
		return commandError{
			message: "schema does not accept a positional input path",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	doc := buildSchemaDoc()
	if opts.format == formatText {
		writeSchemaText(stdout, doc)
		return nil
	}
	return writeJSON(stdout, doc)
}

func runCreate(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, false)
	if err != nil {
		return err
	}
	if opts.input != "" {
		return commandError{
			message: "create does not accept a positional input path",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if opts.output == "" {
		return commandError{
			message: "create requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, err := hwpx.CreateEditableDocument(opts.output)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "create",
			Success:       true,
			Data: createResult{
				OutputPath: absolutePath(opts.output),
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Created editable document at %s\n", opts.output)
	return err
}

func runAppendText(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	text, ok := opts.values["text"]
	if !ok {
		return commandError{
			message: "append-text requires --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	paragraphs := splitParagraphs(text)
	report, added, err := hwpx.AddParagraphs(opts.input, paragraphs)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "append-text",
			Success:       true,
			Data: paragraphEditResult{
				InputPath:       absolutePath(opts.input),
				AddedParagraphs: added,
				Report:          report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added %d paragraph(s) to %s\n", added, opts.input)
	return err
}

func runAddTable(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	cells := parseCellMatrix(opts.values["cells"])
	rows, err := parseOptionalIntArg(opts.values, "rows")
	if err != nil {
		return err
	}
	cols, err := parseOptionalIntArg(opts.values, "cols")
	if err != nil {
		return err
	}

	if rows == 0 {
		rows = len(cells)
	}
	if cols == 0 {
		for _, row := range cells {
			if len(row) > cols {
				cols = len(row)
			}
		}
	}
	if rows <= 0 || cols <= 0 {
		return commandError{
			message: "add-table requires positive --rows/--cols or a non-empty --cells matrix",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, tableIndex, err := hwpx.AddTable(opts.input, hwpx.TableSpec{
		Rows:  rows,
		Cols:  cols,
		Cells: cells,
	})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-table",
			Success:       true,
			Data: tableAddResult{
				InputPath:  absolutePath(opts.input),
				TableIndex: tableIndex,
				Rows:       rows,
				Cols:       cols,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added table #%d (%dx%d) to %s\n", tableIndex, rows, cols, opts.input)
	return err
}

func runSetTableCell(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	tableIndex, err := requireIntArg(opts.values, "table")
	if err != nil {
		return err
	}
	row, err := requireIntArg(opts.values, "row")
	if err != nil {
		return err
	}
	col, err := requireIntArg(opts.values, "col")
	if err != nil {
		return err
	}
	text, ok := opts.values["text"]
	if !ok {
		return commandError{
			message: "set-table-cell requires --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, err := hwpx.SetTableCellText(opts.input, tableIndex, row, col, text)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-table-cell",
			Success:       true,
			Data: tableCellEditResult{
				InputPath:  absolutePath(opts.input),
				TableIndex: tableIndex,
				Row:        row,
				Col:        col,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated table #%d cell (%d,%d) in %s\n", tableIndex, row, col, opts.input)
	return err
}

func runEmbedImage(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	imagePath, ok := opts.values["image"]
	if !ok {
		return commandError{
			message: "embed-image requires --image",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, embedded, err := hwpx.EmbedImage(opts.input, imagePath)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "embed-image",
			Success:       true,
			Data: imageEmbedResult{
				InputPath:  absolutePath(opts.input),
				ImagePath:  absolutePath(imagePath),
				ItemID:     embedded.ItemID,
				BinaryPath: embedded.BinaryPath,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Embedded image as %s in %s\n", embedded.ItemID, opts.input)
	return err
}

func runInsertImage(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	imagePath, ok := opts.values["image"]
	if !ok {
		return commandError{
			message: "insert-image requires --image",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	widthMM, err := parseOptionalFloatArg(opts.values, "width-mm")
	if err != nil {
		return err
	}

	report, placed, err := hwpx.InsertImage(opts.input, imagePath, widthMM)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "insert-image",
			Success:       true,
			Data: imageInsertResult{
				InputPath:    absolutePath(opts.input),
				ImagePath:    absolutePath(imagePath),
				ItemID:       placed.ItemID,
				BinaryPath:   placed.BinaryPath,
				PixelWidth:   placed.PixelWidth,
				PixelHeight:  placed.PixelHeight,
				PlacedWidth:  placed.Width,
				PlacedHeight: placed.Height,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Inserted image %s into %s\n", placed.ItemID, opts.input)
	return err
}

func runPrintPDF(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}
	if opts.output == "" {
		return commandError{
			message: "print-pdf requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	workspaceDir, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := hwpx.PrintToPDF(opts.input, opts.output, workspaceDir); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "print-pdf",
			Success:       true,
			Data: printPDFResult{
				InputPath:  absolutePath(opts.input),
				OutputPath: absolutePath(opts.output),
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Printed PDF to %s\n", opts.output)
	return err
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
  hwpxctl add-table <directory> [--rows <n>] [--cols <n>] [--cells <r1c1,r1c2;r2c1,r2c2>] [--format text|json]
  hwpxctl set-table-cell <directory> --table <n> --row <n> --col <n> --text <text> [--format text|json]
  hwpxctl embed-image <directory> --image <file> [--format text|json]
  hwpxctl insert-image <directory> --image <file> [--width-mm <n>] [--format text|json]
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
