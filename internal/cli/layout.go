package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathucop/project-hwpx-cli/internal/hwpx"
)

func runSetHeader(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	return runSetHeaderFooter("header", cmd, args, stdout, defaultFormat)
}

func runSetFooter(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	return runSetHeaderFooter("footer", cmd, args, stdout, defaultFormat)
}

func runRemoveHeader(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	return runRemoveHeaderFooter("header", cmd, args, stdout, defaultFormat)
}

func runRemoveFooter(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	return runRemoveHeaderFooter("footer", cmd, args, stdout, defaultFormat)
}

func runSetHeaderFooter(kind string, cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
	if err := maybeRecordChange(opts, "set-"+kind, fmt.Sprintf("Updated %s", kind), &report); err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "remove-"+kind, fmt.Sprintf("Removed %s", kind), &report); err != nil {
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

func runRemoveHeaderFooter(kind string, cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, false)
	if err != nil {
		return err
	}

	var report hwpx.Report
	if kind == "header" {
		report, err = hwpx.RemoveHeader(opts.input)
	} else {
		report, err = hwpx.RemoveFooter(opts.input)
	}
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "remove-" + kind,
			Success:       true,
			Data: headerFooterResult{
				InputPath: absolutePath(opts.input),
				Kind:      kind,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Removed %s from %s\n", kind, opts.input)
	return err
}

func runSetPageNumber(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
	if err := maybeRecordChange(opts, "set-page-number", "Updated page number", &report); err != nil {
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

func runSetColumns(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	count, err := requireIntArg(opts.values, "count")
	if err != nil {
		return err
	}
	if count <= 0 {
		return commandError{
			message: "set-columns requires positive --count",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	gapMM, err := parseOptionalFloatArg(opts.values, "gap-mm")
	if err != nil {
		return err
	}
	if gapMM < 0 {
		return commandError{
			message: "set-columns requires zero or greater --gap-mm",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, err := hwpx.SetColumns(opts.input, hwpx.ColumnSpec{
		Count: count,
		GapMM: gapMM,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-columns", fmt.Sprintf("Updated columns to %d", count), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-columns",
			Success:       true,
			Data: columnsResult{
				InputPath: absolutePath(opts.input),
				Count:     count,
				GapMM:     gapMM,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated columns in %s\n", opts.input)
	return err
}

func runSetPageLayout(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	orientation := strings.ToUpper(strings.TrimSpace(opts.values["orientation"]))
	if orientation != "" && !isAllowedValue(orientation, "PORTRAIT", "LANDSCAPE") {
		return commandError{
			message: "set-page-layout requires a valid --orientation",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	widthMM, err := optionalFloatPointer(opts.values, "width-mm")
	if err != nil {
		return err
	}
	heightMM, err := optionalFloatPointer(opts.values, "height-mm")
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
	topMarginMM, err := optionalFloatPointer(opts.values, "top-margin-mm")
	if err != nil {
		return err
	}
	bottomMarginMM, err := optionalFloatPointer(opts.values, "bottom-margin-mm")
	if err != nil {
		return err
	}
	headerMarginMM, err := optionalFloatPointer(opts.values, "header-margin-mm")
	if err != nil {
		return err
	}
	footerMarginMM, err := optionalFloatPointer(opts.values, "footer-margin-mm")
	if err != nil {
		return err
	}
	gutterMarginMM, err := optionalFloatPointer(opts.values, "gutter-margin-mm")
	if err != nil {
		return err
	}
	gutterType := strings.ToUpper(strings.TrimSpace(opts.values["gutter-type"]))

	borderFillIDRef, err := optionalIntPointer(opts.values, "border-fill-id-ref")
	if err != nil {
		return err
	}
	borderTextBorder := strings.ToUpper(strings.TrimSpace(opts.values["border-text-border"]))
	borderFillArea := strings.ToUpper(strings.TrimSpace(opts.values["border-fill-area"]))
	borderHeaderInside, err := parseOptionalBoolArg(opts.values, "border-header-inside")
	if err != nil {
		return err
	}
	borderFooterInside, err := parseOptionalBoolArg(opts.values, "border-footer-inside")
	if err != nil {
		return err
	}
	borderOffsetLeftMM, err := optionalFloatPointer(opts.values, "border-offset-left-mm")
	if err != nil {
		return err
	}
	borderOffsetRightMM, err := optionalFloatPointer(opts.values, "border-offset-right-mm")
	if err != nil {
		return err
	}
	borderOffsetTopMM, err := optionalFloatPointer(opts.values, "border-offset-top-mm")
	if err != nil {
		return err
	}
	borderOffsetBottomMM, err := optionalFloatPointer(opts.values, "border-offset-bottom-mm")
	if err != nil {
		return err
	}

	if orientation == "" &&
		widthMM == nil &&
		heightMM == nil &&
		leftMarginMM == nil &&
		rightMarginMM == nil &&
		topMarginMM == nil &&
		bottomMarginMM == nil &&
		headerMarginMM == nil &&
		footerMarginMM == nil &&
		gutterMarginMM == nil &&
		gutterType == "" &&
		borderFillIDRef == nil &&
		borderTextBorder == "" &&
		borderFillArea == "" &&
		borderHeaderInside == nil &&
		borderFooterInside == nil &&
		borderOffsetLeftMM == nil &&
		borderOffsetRightMM == nil &&
		borderOffsetTopMM == nil &&
		borderOffsetBottomMM == nil {
		return commandError{
			message: "set-page-layout requires at least one page layout option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, err := hwpx.SetPageLayout(opts.input, hwpx.PageLayoutSpec{
		Orientation:          orientation,
		WidthMM:              widthMM,
		HeightMM:             heightMM,
		LeftMarginMM:         leftMarginMM,
		RightMarginMM:        rightMarginMM,
		TopMarginMM:          topMarginMM,
		BottomMarginMM:       bottomMarginMM,
		HeaderMarginMM:       headerMarginMM,
		FooterMarginMM:       footerMarginMM,
		GutterMarginMM:       gutterMarginMM,
		GutterType:           gutterType,
		BorderFillIDRef:      borderFillIDRef,
		BorderTextBorder:     borderTextBorder,
		BorderFillArea:       borderFillArea,
		BorderHeaderInside:   borderHeaderInside,
		BorderFooterInside:   borderFooterInside,
		BorderOffsetLeftMM:   borderOffsetLeftMM,
		BorderOffsetRightMM:  borderOffsetRightMM,
		BorderOffsetTopMM:    borderOffsetTopMM,
		BorderOffsetBottomMM: borderOffsetBottomMM,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-page-layout", "Updated page layout", &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-page-layout",
			Success:       true,
			Data: pageLayoutResult{
				InputPath:            absolutePath(opts.input),
				Orientation:          orientation,
				WidthMM:              widthMM,
				HeightMM:             heightMM,
				LeftMarginMM:         leftMarginMM,
				RightMarginMM:        rightMarginMM,
				TopMarginMM:          topMarginMM,
				BottomMarginMM:       bottomMarginMM,
				HeaderMarginMM:       headerMarginMM,
				FooterMarginMM:       footerMarginMM,
				GutterMarginMM:       gutterMarginMM,
				GutterType:           gutterType,
				BorderFillIDRef:      borderFillIDRef,
				BorderTextBorder:     borderTextBorder,
				BorderFillArea:       borderFillArea,
				BorderHeaderInside:   borderHeaderInside,
				BorderFooterInside:   borderFooterInside,
				BorderOffsetLeftMM:   borderOffsetLeftMM,
				BorderOffsetRightMM:  borderOffsetRightMM,
				BorderOffsetTopMM:    borderOffsetTopMM,
				BorderOffsetBottomMM: borderOffsetBottomMM,
				Report:               report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated page layout in %s\n", opts.input)
	return err
}
