package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx"
)

func runAddBookmark(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	name := strings.TrimSpace(opts.values["name"])
	if name == "" {
		return commandError{
			message: "missing required --name",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	text := strings.TrimSpace(opts.values["text"])
	if text == "" {
		return commandError{
			message: "missing required --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, err := hwpx.AddBookmark(opts.input, hwpx.BookmarkSpec{
		Name: name,
		Text: text,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-bookmark", fmt.Sprintf("Added bookmark %s", name), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-bookmark",
			Success:       true,
			Data: bookmarkResult{
				InputPath: absolutePath(opts.input),
				Name:      name,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added bookmark %s to %s\n", name, opts.input)
	return err
}

func runAddHyperlink(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	target := strings.TrimSpace(opts.values["target"])
	if target == "" {
		return commandError{
			message: "missing required --target",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	text := strings.TrimSpace(opts.values["text"])
	if text == "" {
		return commandError{
			message: "missing required --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, fieldID, err := hwpx.AddHyperlink(opts.input, hwpx.HyperlinkSpec{
		Target: target,
		Text:   text,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-hyperlink", fmt.Sprintf("Added hyperlink %s", target), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-hyperlink",
			Success:       true,
			Data: hyperlinkResult{
				InputPath: absolutePath(opts.input),
				Target:    target,
				FieldID:   fieldID,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added hyperlink %s to %s\n", target, opts.input)
	return err
}

func runAddHeading(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	kind := strings.ToLower(strings.TrimSpace(opts.values["kind"]))
	if kind == "" {
		kind = "heading"
	}
	level, err := parseOptionalIntArg(opts.values, "level")
	if err != nil {
		return err
	}
	text := strings.TrimSpace(opts.values["text"])
	if text == "" {
		return commandError{
			message: "missing required --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	bookmarkName := strings.TrimSpace(opts.values["bookmark"])

	report, resolvedBookmark, err := hwpx.AddHeading(opts.input, hwpx.HeadingSpec{
		Kind:         kind,
		Level:        level,
		Text:         text,
		BookmarkName: bookmarkName,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-heading", fmt.Sprintf("Added %s %s", kind, resolvedBookmark), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-heading",
			Success:       true,
			Data: headingResult{
				InputPath:    absolutePath(opts.input),
				Kind:         kind,
				Level:        level,
				Text:         text,
				BookmarkName: resolvedBookmark,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added %s paragraph to %s with bookmark %s\n", kind, opts.input, resolvedBookmark)
	return err
}

func runInsertTOC(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	title := strings.TrimSpace(opts.values["title"])
	maxLevel, err := parseOptionalIntArg(opts.values, "max-level")
	if err != nil {
		return err
	}

	report, entryCount, err := hwpx.InsertTOC(opts.input, hwpx.TOCSpec{
		Title:    title,
		MaxLevel: maxLevel,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "insert-toc", fmt.Sprintf("Inserted TOC with %d entry(s)", entryCount), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "insert-toc",
			Success:       true,
			Data: tocResult{
				InputPath:  absolutePath(opts.input),
				Title:      fallbackCLIString(title, "목차"),
				MaxLevel:   maxIntCLI(maxLevel, 3),
				EntryCount: entryCount,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Inserted table of contents (%d entries) into %s\n", entryCount, opts.input)
	return err
}

func runAddCrossReference(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	bookmarkName := strings.TrimSpace(opts.values["bookmark"])
	if bookmarkName == "" {
		return commandError{
			message: "missing required --bookmark",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	text := strings.TrimSpace(opts.values["text"])

	report, fieldID, resolvedText, err := hwpx.AddCrossReference(opts.input, hwpx.CrossReferenceSpec{
		BookmarkName: bookmarkName,
		Text:         text,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-cross-reference", fmt.Sprintf("Added cross reference to %s", bookmarkName), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-cross-reference",
			Success:       true,
			Data: crossReferenceResult{
				InputPath:    absolutePath(opts.input),
				BookmarkName: bookmarkName,
				Text:         resolvedText,
				FieldID:      fieldID,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added cross reference to %s in %s\n", bookmarkName, opts.input)
	return err
}
