package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func absolutePath(value string) string {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return value
	}
	return absolute
}
