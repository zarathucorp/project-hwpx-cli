package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

func runAppendText(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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

func runSetParagraphText(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	paragraphIndex, err := requireIntArg(opts.values, "paragraph")
	if err != nil {
		return err
	}
	text, ok := opts.values["text"]
	if !ok {
		return commandError{
			message: "set-paragraph-text requires --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, previousText, err := hwpx.SetParagraphText(opts.input, paragraphIndex, text)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-paragraph-text",
			Success:       true,
			Data: paragraphUpdateResult{
				InputPath:    absolutePath(opts.input),
				Paragraph:    paragraphIndex,
				PreviousText: previousText,
				Deleted:      false,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated paragraph %d in %s\n", paragraphIndex, opts.input)
	return err
}

func runSetTextStyle(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	paragraphIndex, err := requireIntArg(opts.values, "paragraph")
	if err != nil {
		return err
	}

	var runIndex *int
	if _, ok := opts.values["run"]; ok {
		value, err := requireIntArg(opts.values, "run")
		if err != nil {
			return err
		}
		runIndex = &value
	}

	bold, err := parseOptionalBoolArg(opts.values, "bold")
	if err != nil {
		return err
	}
	italic, err := parseOptionalBoolArg(opts.values, "italic")
	if err != nil {
		return err
	}
	underline, err := parseOptionalBoolArg(opts.values, "underline")
	if err != nil {
		return err
	}
	textColor, err := parseOptionalColorArg(opts.values, "text-color")
	if err != nil {
		return err
	}
	if bold == nil && italic == nil && underline == nil && textColor == "" {
		return commandError{
			message: "set-text-style requires at least one style option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, charPrIDs, appliedRuns, err := hwpx.ApplyTextStyle(opts.input, paragraphIndex, runIndex, hwpx.TextStyleSpec{
		Bold:      bold,
		Italic:    italic,
		Underline: underline,
		TextColor: textColor,
	})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-text-style",
			Success:       true,
			Data: textStyleResult{
				InputPath:   absolutePath(opts.input),
				Paragraph:   paragraphIndex,
				Run:         runIndex,
				AppliedRuns: appliedRuns,
				CharPrIDs:   charPrIDs,
				Bold:        bold,
				Italic:      italic,
				Underline:   underline,
				TextColor:   textColor,
				Report:      report,
			},
		})
	}

	if runIndex != nil {
		_, err = fmt.Fprintf(stdout, "Updated paragraph %d run %d style in %s\n", paragraphIndex, *runIndex, opts.input)
		return err
	}

	_, err = fmt.Fprintf(stdout, "Updated paragraph %d style across %d run(s) in %s\n", paragraphIndex, appliedRuns, opts.input)
	return err
}

func runDeleteParagraph(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	paragraphIndex, err := requireIntArg(opts.values, "paragraph")
	if err != nil {
		return err
	}

	report, removedText, err := hwpx.DeleteParagraph(opts.input, paragraphIndex)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "delete-paragraph",
			Success:       true,
			Data: paragraphUpdateResult{
				InputPath:   absolutePath(opts.input),
				Paragraph:   paragraphIndex,
				RemovedText: removedText,
				Deleted:     true,
				Report:      report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Deleted paragraph %d from %s\n", paragraphIndex, opts.input)
	return err
}
