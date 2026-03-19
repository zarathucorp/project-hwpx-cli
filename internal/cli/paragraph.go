package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx"
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

	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	paragraphs := splitParagraphs(text)
	report, added, err := hwpx.AddParagraphs(opts.input, selector, paragraphs)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "append-text", fmt.Sprintf("Appended %d paragraph(s)", added), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "append-text",
			Success:       true,
			Data: paragraphEditResult{
				InputPath:       absolutePath(opts.input),
				SectionIndex:    resolveSelectedSectionIndex(selector),
				SectionPath:     resolveSelectedSectionPath(selector),
				AddedParagraphs: added,
				Report:          report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added %d paragraph(s) to section %d in %s\n", added, resolveSelectedSectionIndex(selector), opts.input)
	return err
}

func runAddRunText(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
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
			message: "add-run-text requires --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	var runIndex *int
	if _, ok := opts.values["run"]; ok {
		value, err := requireIntArg(opts.values, "run")
		if err != nil {
			return err
		}
		runIndex = &value
	}

	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, insertedRun, charPrIDRef, err := hwpx.AddRunText(opts.input, selector, paragraphIndex, runIndex, text)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-run-text", fmt.Sprintf("Inserted run %d into paragraph %d", insertedRun, paragraphIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-run-text",
			Success:       true,
			Data: runTextAddResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				Paragraph:    paragraphIndex,
				Run:          insertedRun,
				Text:         text,
				CharPrIDRef:  charPrIDRef,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Inserted run %d into section %d paragraph %d in %s\n", insertedRun, resolveSelectedSectionIndex(selector), paragraphIndex, opts.input)
	return err
}

func runSetRunText(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	paragraphIndex, err := requireIntArg(opts.values, "paragraph")
	if err != nil {
		return err
	}
	runIndex, err := requireIntArg(opts.values, "run")
	if err != nil {
		return err
	}
	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}
	text, ok := opts.values["text"]
	if !ok {
		return commandError{
			message: "set-run-text requires --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, previousText, charPrIDRef, err := hwpx.SetRunText(opts.input, selector, paragraphIndex, runIndex, text)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-run-text", fmt.Sprintf("Updated paragraph %d run %d text", paragraphIndex, runIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-run-text",
			Success:       true,
			Data: runTextUpdateResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				Paragraph:    paragraphIndex,
				Run:          runIndex,
				Text:         text,
				PreviousText: previousText,
				CharPrIDRef:  charPrIDRef,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated section %d paragraph %d run %d text in %s\n", resolveSelectedSectionIndex(selector), paragraphIndex, runIndex, opts.input)
	return err
}

func runFindRunsByStyle(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
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
	fontName := strings.TrimSpace(opts.values["font-name"])
	fontSizePt, err := optionalPositiveFloatPointer(opts.values, "font-size-pt")
	if err != nil {
		return err
	}
	if bold == nil && italic == nil && underline == nil && textColor == "" && fontName == "" && fontSizePt == nil {
		return commandError{
			message: "find-runs-by-style requires at least one style option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}

	matches, err := hwpx.FindRunsByStyle(opts.input, selector, hwpx.RunStyleFilter{
		Bold:       bold,
		Italic:     italic,
		Underline:  underline,
		TextColor:  textColor,
		FontName:   fontName,
		FontSizePt: fontSizePt,
	})
	if err != nil {
		return err
	}
	if matches == nil {
		matches = []hwpx.RunStyleMatch{}
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "find-runs-by-style",
			Success:       true,
			Data: runStyleSearchResult{
				InputPath: absolutePath(opts.input),
				Count:     len(matches),
				Matches:   matches,
			},
		})
	}

	if len(matches) == 0 {
		_, err = fmt.Fprintln(stdout, "No matching runs found")
		return err
	}

	for _, match := range matches {
		if _, err := fmt.Fprintf(stdout, "section=%d paragraph=%d run=%d table=%s cell=%s charPr=%s text=%q\n", match.SectionIndex, match.ParagraphIndex, match.Run, formatOptionalInt(match.TableIndex), formatCellCoordinate(match.Cell), match.CharPrIDRef, match.Text); err != nil {
			return err
		}
	}
	return nil
}

func runReplaceRunsByStyle(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	text, ok := opts.values["text"]
	if !ok {
		return commandError{
			message: "replace-runs-by-style requires --text",
			code:    1,
			kind:    "invalid_arguments",
		}
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
	fontName := strings.TrimSpace(opts.values["font-name"])
	fontSizePt, err := optionalPositiveFloatPointer(opts.values, "font-size-pt")
	if err != nil {
		return err
	}
	if bold == nil && italic == nil && underline == nil && textColor == "" && fontName == "" && fontSizePt == nil {
		return commandError{
			message: "replace-runs-by-style requires at least one style option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}

	report, replacements, err := hwpx.ReplaceRunsByStyle(opts.input, selector, hwpx.RunStyleFilter{
		Bold:       bold,
		Italic:     italic,
		Underline:  underline,
		TextColor:  textColor,
		FontName:   fontName,
		FontSizePt: fontSizePt,
	}, text)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "replace-runs-by-style", fmt.Sprintf("Replaced %d run(s) by style", len(replacements)), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "replace-runs-by-style",
			Success:       true,
			Data: runStyleReplaceResult{
				InputPath:    absolutePath(opts.input),
				Count:        len(replacements),
				Text:         text,
				Replacements: replacements,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Replaced %d run(s) in %s\n", len(replacements), opts.input)
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
	selector, err := parseSectionSelector(opts.values, false)
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

	report, previousText, err := hwpx.SetParagraphText(opts.input, selector, paragraphIndex, text)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-paragraph-text", fmt.Sprintf("Updated paragraph %d", paragraphIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-paragraph-text",
			Success:       true,
			Data: paragraphUpdateResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				Paragraph:    paragraphIndex,
				PreviousText: previousText,
				Deleted:      false,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated section %d paragraph %d in %s\n", resolveSelectedSectionIndex(selector), paragraphIndex, opts.input)
	return err
}

func runSetParagraphLayout(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	paragraphIndex, err := requireIntArg(opts.values, "paragraph")
	if err != nil {
		return err
	}

	align := strings.ToUpper(strings.TrimSpace(opts.values["align"]))
	if align != "" && !isAllowedValue(align, "LEFT", "RIGHT", "CENTER", "JUSTIFY", "DISTRIBUTE", "DISTRIBUTE_SPACE") {
		return commandError{
			message: "set-paragraph-layout requires a valid --align",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	indentMM, err := optionalFloatPointer(opts.values, "indent-mm")
	if err != nil {
		return err
	}
	leftMarginMM, err := optionalFloatPointer(opts.values, "left-margin-mm")
	if err != nil {
		return err
	}
	rightMarginMM, err := optionalFloatPointer(opts.values, "right-margin-mm")
	if err != nil {
		return err
	}
	spaceBeforeMM, err := optionalFloatPointer(opts.values, "space-before-mm")
	if err != nil {
		return err
	}
	spaceAfterMM, err := optionalFloatPointer(opts.values, "space-after-mm")
	if err != nil {
		return err
	}
	lineSpacingPercent, err := optionalIntPointer(opts.values, "line-spacing-percent")
	if err != nil {
		return err
	}
	if lineSpacingPercent != nil && *lineSpacingPercent <= 0 {
		return commandError{
			message: "set-paragraph-layout requires positive --line-spacing-percent",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if align == "" && indentMM == nil && leftMarginMM == nil && rightMarginMM == nil && spaceBeforeMM == nil && spaceAfterMM == nil && lineSpacingPercent == nil {
		return commandError{
			message: "set-paragraph-layout requires at least one layout option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, paraPrID, err := hwpx.SetParagraphLayout(opts.input, selector, paragraphIndex, hwpx.ParagraphLayoutSpec{
		Align:              align,
		IndentMM:           indentMM,
		LeftMarginMM:       leftMarginMM,
		RightMarginMM:      rightMarginMM,
		SpaceBeforeMM:      spaceBeforeMM,
		SpaceAfterMM:       spaceAfterMM,
		LineSpacingPercent: lineSpacingPercent,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-paragraph-layout", fmt.Sprintf("Updated paragraph %d layout", paragraphIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-paragraph-layout",
			Success:       true,
			Data: paragraphLayoutResult{
				InputPath:          absolutePath(opts.input),
				SectionIndex:       resolveSelectedSectionIndex(selector),
				SectionPath:        resolveSelectedSectionPath(selector),
				Paragraph:          paragraphIndex,
				ParaPrIDRef:        paraPrID,
				Align:              align,
				IndentMM:           indentMM,
				LeftMarginMM:       leftMarginMM,
				RightMarginMM:      rightMarginMM,
				SpaceBeforeMM:      spaceBeforeMM,
				SpaceAfterMM:       spaceAfterMM,
				LineSpacingPercent: lineSpacingPercent,
				Report:             report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated section %d paragraph %d layout in %s\n", resolveSelectedSectionIndex(selector), paragraphIndex, opts.input)
	return err
}

func runSetParagraphList(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	paragraphIndex, err := requireIntArg(opts.values, "paragraph")
	if err != nil {
		return err
	}

	kind := strings.ToLower(strings.TrimSpace(opts.values["kind"]))
	if !isAllowedValue(kind, "bullet", "number", "none") {
		return commandError{
			message: "set-paragraph-list requires --kind bullet, number, or none",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	level := 0
	if _, ok := opts.values["level"]; ok {
		level, err = requireIntArg(opts.values, "level")
		if err != nil {
			return err
		}
	}

	var startNumber *int
	if _, ok := opts.values["start-number"]; ok {
		value, err := requireIntArg(opts.values, "start-number")
		if err != nil {
			return err
		}
		startNumber = &value
	}

	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, paraPrID, err := hwpx.SetParagraphList(opts.input, selector, paragraphIndex, hwpx.ParagraphListSpec{
		Kind:        kind,
		Level:       level,
		StartNumber: startNumber,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-paragraph-list", fmt.Sprintf("Updated paragraph %d list", paragraphIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-paragraph-list",
			Success:       true,
			Data: paragraphListResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				Paragraph:    paragraphIndex,
				Kind:         kind,
				Level:        level,
				StartNumber:  startNumber,
				ParaPrIDRef:  paraPrID,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated section %d paragraph %d list in %s\n", resolveSelectedSectionIndex(selector), paragraphIndex, opts.input)
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
	fontName := strings.TrimSpace(opts.values["font-name"])
	fontSizePt, err := optionalPositiveFloatPointer(opts.values, "font-size-pt")
	if err != nil {
		return err
	}
	if bold == nil && italic == nil && underline == nil && textColor == "" && fontName == "" && fontSizePt == nil {
		return commandError{
			message: "set-text-style requires at least one style option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, charPrIDs, appliedRuns, err := hwpx.ApplyTextStyle(opts.input, selector, paragraphIndex, runIndex, hwpx.TextStyleSpec{
		Bold:       bold,
		Italic:     italic,
		Underline:  underline,
		TextColor:  textColor,
		FontName:   fontName,
		FontSizePt: fontSizePt,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-text-style", fmt.Sprintf("Updated text style in paragraph %d", paragraphIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-text-style",
			Success:       true,
			Data: textStyleResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				Paragraph:    paragraphIndex,
				Run:          runIndex,
				AppliedRuns:  appliedRuns,
				CharPrIDs:    charPrIDs,
				Bold:         bold,
				Italic:       italic,
				Underline:    underline,
				TextColor:    textColor,
				FontName:     fontName,
				FontSizePt:   fontSizePt,
				Report:       report,
			},
		})
	}

	if runIndex != nil {
		_, err = fmt.Fprintf(stdout, "Updated section %d paragraph %d run %d style in %s\n", resolveSelectedSectionIndex(selector), paragraphIndex, *runIndex, opts.input)
		return err
	}

	_, err = fmt.Fprintf(stdout, "Updated section %d paragraph %d style across %d run(s) in %s\n", resolveSelectedSectionIndex(selector), paragraphIndex, appliedRuns, opts.input)
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

	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, removedText, err := hwpx.DeleteParagraph(opts.input, selector, paragraphIndex)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "delete-paragraph", fmt.Sprintf("Deleted paragraph %d", paragraphIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "delete-paragraph",
			Success:       true,
			Data: paragraphUpdateResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				Paragraph:    paragraphIndex,
				RemovedText:  removedText,
				Deleted:      true,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Deleted section %d paragraph %d from %s\n", resolveSelectedSectionIndex(selector), paragraphIndex, opts.input)
	return err
}
