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
