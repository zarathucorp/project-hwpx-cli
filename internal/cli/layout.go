package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

func runSetHeader(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	return runSetHeaderFooter("header", args, stdout, defaultFormat)
}

func runSetFooter(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	return runSetHeaderFooter("footer", args, stdout, defaultFormat)
}

func runSetHeaderFooter(kind string, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	text, ok := opts.values["text"]
	if !ok {
		return commandError{
			message: fmt.Sprintf("set-%s requires --text", kind),
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	applyPageType := strings.ToUpper(strings.TrimSpace(opts.values["apply-page-type"]))
	if applyPageType == "" {
		applyPageType = "BOTH"
	}

	var report hwpx.Report
	spec := hwpx.HeaderFooterSpec{
		Text:          splitParagraphs(text),
		ApplyPageType: applyPageType,
	}
	if kind == "header" {
		report, err = hwpx.SetHeaderText(opts.input, spec)
	} else {
		report, err = hwpx.SetFooterText(opts.input, spec)
	}
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-" + kind,
			Success:       true,
			Data: headerFooterResult{
				InputPath:     absolutePath(opts.input),
				Kind:          kind,
				ApplyPageType: applyPageType,
				Report:        report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated %s in %s\n", kind, opts.input)
	return err
}

func runSetPageNumber(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	position := strings.ToUpper(strings.TrimSpace(opts.values["position"]))
	if position == "" {
		position = "BOTTOM_CENTER"
	}
	formatType := strings.ToUpper(strings.TrimSpace(opts.values["type"]))
	if formatType == "" {
		formatType = "DIGIT"
	}
	sideChar := opts.values["side-char"]
	startPage, err := parseOptionalIntArg(opts.values, "start-page")
	if err != nil {
		return err
	}

	report, err := hwpx.SetPageNumber(opts.input, hwpx.PageNumberSpec{
		Position:   position,
		FormatType: formatType,
		SideChar:   sideChar,
		StartPage:  startPage,
	})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-page-number",
			Success:       true,
			Data: pageNumberResult{
				InputPath:  absolutePath(opts.input),
				Position:   position,
				FormatType: formatType,
				SideChar:   sideChar,
				StartPage:  startPage,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated page number in %s\n", opts.input)
	return err
}
