package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx"
)

func runAddNote(kind string, cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	anchorText := opts.values["anchor-text"]
	if strings.TrimSpace(anchorText) == "" {
		return commandError{
			message: "missing required --anchor-text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	noteText := opts.values["text"]
	if strings.TrimSpace(noteText) == "" {
		return commandError{
			message: "missing required --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	spec := hwpx.NoteSpec{
		AnchorText: anchorText,
		Text:       splitParagraphs(noteText),
	}

	var (
		report hwpx.Report
		number int
	)
	if kind == "footnote" {
		report, number, err = hwpx.AddFootnote(opts.input, spec)
	} else {
		report, number, err = hwpx.AddEndnote(opts.input, spec)
	}
	if err != nil {
		return err
	}
	commandName := "add-" + kind
	if err := maybeRecordChange(opts, commandName, fmt.Sprintf("Added %s %d", kind, number), &report); err != nil {
		return err
	}
	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       commandName,
			Success:       true,
			Data: noteResult{
				InputPath: absolutePath(opts.input),
				Kind:      kind,
				Number:    number,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added %s %d to %s\n", kind, number, opts.input)
	return err
}

func runAddMemo(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	anchorText := opts.values["anchor-text"]
	if strings.TrimSpace(anchorText) == "" {
		return commandError{
			message: "missing required --anchor-text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	memoText := opts.values["text"]
	if strings.TrimSpace(memoText) == "" {
		return commandError{
			message: "missing required --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	author := strings.TrimSpace(opts.values["author"])
	report, memoID, fieldID, number, err := hwpx.AddMemo(opts.input, hwpx.MemoSpec{
		AnchorText: anchorText,
		Text:       splitParagraphs(memoText),
		Author:     author,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-memo", fmt.Sprintf("Added memo %d", number), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-memo",
			Success:       true,
			Data: memoResult{
				InputPath: absolutePath(opts.input),
				MemoID:    memoID,
				FieldID:   fieldID,
				Number:    number,
				Author:    author,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added memo %d to %s\n", number, opts.input)
	return err
}
