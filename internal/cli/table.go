package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx"
)

func runAddTable(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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

	spec, err := parseTableSpecOptions(opts.values, rows, cols, cells)
	if err != nil {
		return err
	}

	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, tableIndex, err := hwpx.AddTable(opts.input, selector, spec)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-table", fmt.Sprintf("Added table %d (%dx%d)", tableIndex, rows, cols), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-table",
			Success:       true,
			Data: tableAddResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				TableIndex:   tableIndex,
				Rows:         rows,
				Cols:         cols,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added table #%d (%dx%d) to section %d in %s\n", tableIndex, rows, cols, resolveSelectedSectionIndex(selector), opts.input)
	return err
}

func runAddNestedTable(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
		for _, values := range cells {
			if len(values) > cols {
				cols = len(values)
			}
		}
	}
	if rows <= 0 || cols <= 0 {
		return commandError{
			message: "add-nested-table requires positive --rows/--cols or a non-empty --cells matrix",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	spec, err := parseTableSpecOptions(opts.values, rows, cols, cells)
	if err != nil {
		return err
	}

	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, err := hwpx.AddNestedTable(opts.input, selector, tableIndex, row, col, spec)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-nested-table", fmt.Sprintf("Added nested table to table %d cell (%d,%d)", tableIndex, row, col), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-nested-table",
			Success:       true,
			Data: nestedTableAddResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				TableIndex:   tableIndex,
				Row:          row,
				Col:          col,
				Rows:         rows,
				Cols:         cols,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added nested table (%dx%d) to section %d table #%d cell (%d,%d) in %s\n", rows, cols, resolveSelectedSectionIndex(selector), tableIndex, row, col, opts.input)
	return err
}

func runSetTableCell(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	text, hasText := opts.values["text"]
	cellStyleSpec, backgroundColor, err := parseTableCellStyleSpec(opts.values)
	if err != nil {
		return err
	}

	align := strings.ToUpper(strings.TrimSpace(opts.values["align"]))
	if align != "" && !isAllowedValue(align, "LEFT", "RIGHT", "CENTER", "JUSTIFY", "DISTRIBUTE", "DISTRIBUTE_SPACE") {
		return commandError{
			message: "set-table-cell requires a valid --align",
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
	contentStyleRequested := align != "" || bold != nil || italic != nil || underline != nil || textColor != "" || fontName != "" || fontSizePt != nil
	if contentStyleRequested && !hasText {
		return commandError{
			message: "set-table-cell content style options require --text",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	cellStyleSpec.Text = nil
	hasCellStyle := tableCellContainerStyleHasChanges(cellStyleSpec)
	if !hasText && !hasCellStyle {
		return commandError{
			message: "set-table-cell requires --text or at least one cell style option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	var report hwpx.Report
	paraPrID := ""
	charPrIDs := []string{}
	appliedRuns := 0

	if hasText {
		report, paraPrID, charPrIDs, appliedRuns, err = hwpx.SetTableCellContent(opts.input, selector, tableIndex, row, col, hwpx.TableCellTextSpec{
			Text: text,
			ParagraphLayout: hwpx.ParagraphLayoutSpec{
				Align: align,
			},
			TextStyle: hwpx.TextStyleSpec{
				Bold:       bold,
				Italic:     italic,
				Underline:  underline,
				TextColor:  textColor,
				FontName:   fontName,
				FontSizePt: fontSizePt,
			},
		})
		if err != nil {
			return err
		}
	}

	if hasCellStyle {
		report, err = hwpx.SetTableCell(opts.input, selector, tableIndex, row, col, cellStyleSpec)
		if err != nil {
			return err
		}
	}

	if err := maybeRecordChange(opts, "set-table-cell", fmt.Sprintf("Updated table %d cell (%d,%d)", tableIndex, row, col), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-table-cell",
			Success:       true,
			Data: tableCellEditResult{
				InputPath:           absolutePath(opts.input),
				SectionIndex:        resolveSelectedSectionIndex(selector),
				SectionPath:         resolveSelectedSectionPath(selector),
				TableIndex:          tableIndex,
				Row:                 row,
				Col:                 col,
				Text:                stringPointerIf(hasText, text),
				ParagraphCount:      paragraphCountIf(hasText, text),
				ParaPrIDRef:         paraPrID,
				AppliedRuns:         appliedRuns,
				CharPrIDs:           charPrIDs,
				Align:               align,
				Bold:                bold,
				Italic:              italic,
				Underline:           underline,
				TextColor:           textColor,
				FontName:            fontName,
				FontSizePt:          fontSizePt,
				VertAlign:           cellStyleSpec.VertAlign,
				MarginLeftMM:        cellStyleSpec.MarginLeftMM,
				MarginRightMM:       cellStyleSpec.MarginRightMM,
				MarginTopMM:         cellStyleSpec.MarginTopMM,
				MarginBottomMM:      cellStyleSpec.MarginBottomMM,
				BorderStyle:         cellStyleSpec.BorderStyle,
				BorderColor:         cellStyleSpec.BorderColor,
				BorderWidthMM:       cellStyleSpec.BorderWidthMM,
				BorderLeftStyle:     cellStyleSpec.BorderLeftStyle,
				BorderRightStyle:    cellStyleSpec.BorderRightStyle,
				BorderTopStyle:      cellStyleSpec.BorderTopStyle,
				BorderBottomStyle:   cellStyleSpec.BorderBottomStyle,
				BorderLeftColor:     cellStyleSpec.BorderLeftColor,
				BorderRightColor:    cellStyleSpec.BorderRightColor,
				BorderTopColor:      cellStyleSpec.BorderTopColor,
				BorderBottomColor:   cellStyleSpec.BorderBottomColor,
				BorderLeftWidthMM:   cellStyleSpec.BorderLeftWidthMM,
				BorderRightWidthMM:  cellStyleSpec.BorderRightWidthMM,
				BorderTopWidthMM:    cellStyleSpec.BorderTopWidthMM,
				BorderBottomWidthMM: cellStyleSpec.BorderBottomWidthMM,
				FillColor:           cellStyleSpec.FillColor,
				BackgroundColor:     backgroundColor,
				Report:              report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated section %d table #%d cell (%d,%d) in %s\n", resolveSelectedSectionIndex(selector), tableIndex, row, col, opts.input)
	return err
}

func parseTableSpecOptions(values map[string]string, rows, cols int, cells [][]string) (hwpx.TableSpec, error) {
	spec := hwpx.TableSpec{
		Rows:  rows,
		Cols:  cols,
		Cells: cells,
	}

	width, err := optionalPositiveFloatArg(values, "width-mm")
	if err != nil {
		return hwpx.TableSpec{}, err
	}
	height, err := optionalPositiveFloatArg(values, "height-mm")
	if err != nil {
		return hwpx.TableSpec{}, err
	}
	colWidths, err := parsePositiveFloatListArg(values, "col-widths-mm")
	if err != nil {
		return hwpx.TableSpec{}, err
	}
	rowHeights, err := parsePositiveFloatListArg(values, "row-heights-mm")
	if err != nil {
		return hwpx.TableSpec{}, err
	}
	marginLeft, err := optionalNonNegativeFloatArg(values, "margin-left-mm")
	if err != nil {
		return hwpx.TableSpec{}, err
	}
	marginRight, err := optionalNonNegativeFloatArg(values, "margin-right-mm")
	if err != nil {
		return hwpx.TableSpec{}, err
	}
	marginTop, err := optionalNonNegativeFloatArg(values, "margin-top-mm")
	if err != nil {
		return hwpx.TableSpec{}, err
	}
	marginBottom, err := optionalNonNegativeFloatArg(values, "margin-bottom-mm")
	if err != nil {
		return hwpx.TableSpec{}, err
	}

	if len(colWidths) > 0 && len(colWidths) != cols {
		return hwpx.TableSpec{}, commandError{
			message: fmt.Sprintf("--col-widths-mm requires %d values, got %d", cols, len(colWidths)),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if len(rowHeights) > 0 && len(rowHeights) != rows {
		return hwpx.TableSpec{}, commandError{
			message: fmt.Sprintf("--row-heights-mm requires %d values, got %d", rows, len(rowHeights)),
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	spec.WidthMM = width
	spec.HeightMM = height
	spec.ColWidthsMM = colWidths
	spec.RowHeightsMM = rowHeights
	spec.MarginLeftMM = marginLeft
	spec.MarginRightMM = marginRight
	spec.MarginTopMM = marginTop
	spec.MarginBottomMM = marginBottom
	return spec, nil
}

func optionalPositiveFloatArg(values map[string]string, key string) (*float64, error) {
	if _, ok := values[key]; !ok {
		return nil, nil
	}
	value, err := parseOptionalFloatArg(values, key)
	if err != nil {
		return nil, err
	}
	if value <= 0 {
		return nil, commandError{
			message: fmt.Sprintf("--%s must be greater than 0", key),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return &value, nil
}

func optionalNonNegativeFloatArg(values map[string]string, key string) (*float64, error) {
	if _, ok := values[key]; !ok {
		return nil, nil
	}
	value, err := parseOptionalFloatArg(values, key)
	if err != nil {
		return nil, err
	}
	if value < 0 {
		return nil, commandError{
			message: fmt.Sprintf("--%s must be 0 or greater", key),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return &value, nil
}

func parsePositiveFloatListArg(values map[string]string, key string) ([]float64, error) {
	raw, ok := values[key]
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	result := make([]float64, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		parsed, err := parseOptionalFloatArg(map[string]string{key: trimmed}, key)
		if err != nil {
			return nil, err
		}
		if parsed <= 0 {
			return nil, commandError{
				message: fmt.Sprintf("--%s values must be greater than 0", key),
				code:    1,
				kind:    "invalid_arguments",
			}
		}
		result = append(result, parsed)
	}
	if len(result) == 0 {
		return nil, commandError{
			message: fmt.Sprintf("--%s requires at least one numeric value", key),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return result, nil
}

func parseTableCellStyleSpec(values map[string]string) (hwpx.TableCellStyleSpec, string, error) {
	text, hasText := values["text"]

	vertAlign := strings.TrimSpace(values["vert-align"])
	if vertAlign != "" {
		vertAlign = strings.ToUpper(vertAlign)
		if vertAlign != "TOP" && vertAlign != "CENTER" && vertAlign != "BOTTOM" {
			return hwpx.TableCellStyleSpec{}, "", commandError{
				message: "set-table-cell --vert-align must be TOP, CENTER, or BOTTOM",
				code:    1,
				kind:    "invalid_arguments",
			}
		}
	}

	marginLeftMM, err := optionalFloatPointer(values, "margin-left-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	marginRightMM, err := optionalFloatPointer(values, "margin-right-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	marginTopMM, err := optionalFloatPointer(values, "margin-top-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	marginBottomMM, err := optionalFloatPointer(values, "margin-bottom-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderWidthMM, err := optionalFloatPointer(values, "border-width-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderLeftWidthMM, err := optionalFloatPointer(values, "border-left-width-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderRightWidthMM, err := optionalFloatPointer(values, "border-right-width-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderTopWidthMM, err := optionalFloatPointer(values, "border-top-width-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderBottomWidthMM, err := optionalFloatPointer(values, "border-bottom-width-mm")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}

	for _, item := range []struct {
		name  string
		value *float64
	}{
		{name: "margin-left-mm", value: marginLeftMM},
		{name: "margin-right-mm", value: marginRightMM},
		{name: "margin-top-mm", value: marginTopMM},
		{name: "margin-bottom-mm", value: marginBottomMM},
		{name: "border-width-mm", value: borderWidthMM},
		{name: "border-left-width-mm", value: borderLeftWidthMM},
		{name: "border-right-width-mm", value: borderRightWidthMM},
		{name: "border-top-width-mm", value: borderTopWidthMM},
		{name: "border-bottom-width-mm", value: borderBottomWidthMM},
	} {
		if item.value != nil && *item.value < 0 {
			return hwpx.TableCellStyleSpec{}, "", commandError{
				message: fmt.Sprintf("set-table-cell --%s must be zero or greater", item.name),
				code:    1,
				kind:    "invalid_arguments",
			}
		}
	}
	if borderWidthMM != nil && *borderWidthMM <= 0 {
		return hwpx.TableCellStyleSpec{}, "", commandError{
			message: "set-table-cell --border-width-mm must be greater than zero",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	for _, item := range []struct {
		name  string
		value *float64
	}{
		{name: "border-left-width-mm", value: borderLeftWidthMM},
		{name: "border-right-width-mm", value: borderRightWidthMM},
		{name: "border-top-width-mm", value: borderTopWidthMM},
		{name: "border-bottom-width-mm", value: borderBottomWidthMM},
	} {
		if item.value != nil && *item.value <= 0 {
			return hwpx.TableCellStyleSpec{}, "", commandError{
				message: fmt.Sprintf("set-table-cell --%s must be greater than zero", item.name),
				code:    1,
				kind:    "invalid_arguments",
			}
		}
	}

	borderStyle, err := parseTableCellBorderStyle(values, "border-style")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderLeftStyle, err := parseTableCellBorderStyle(values, "border-left-style")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderRightStyle, err := parseTableCellBorderStyle(values, "border-right-style")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderTopStyle, err := parseTableCellBorderStyle(values, "border-top-style")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderBottomStyle, err := parseTableCellBorderStyle(values, "border-bottom-style")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}

	borderColor, err := parseOptionalColorArg(values, "border-color")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderLeftColor, err := parseOptionalColorArg(values, "border-left-color")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderRightColor, err := parseOptionalColorArg(values, "border-right-color")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderTopColor, err := parseOptionalColorArg(values, "border-top-color")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	borderBottomColor, err := parseOptionalColorArg(values, "border-bottom-color")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	fillColor, err := parseOptionalColorArg(values, "fill-color")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	backgroundColor, err := parseOptionalColorArg(values, "background-color")
	if err != nil {
		return hwpx.TableCellStyleSpec{}, "", err
	}
	if fillColor != "" && backgroundColor != "" && fillColor != backgroundColor {
		return hwpx.TableCellStyleSpec{}, "", commandError{
			message: "set-table-cell --fill-color and --background-color must match when both are set",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if fillColor == "" {
		fillColor = backgroundColor
	}

	spec := hwpx.TableCellStyleSpec{
		VertAlign:           vertAlign,
		MarginLeftMM:        marginLeftMM,
		MarginRightMM:       marginRightMM,
		MarginTopMM:         marginTopMM,
		MarginBottomMM:      marginBottomMM,
		BorderStyle:         borderStyle,
		BorderColor:         borderColor,
		BorderWidthMM:       borderWidthMM,
		BorderLeftStyle:     borderLeftStyle,
		BorderRightStyle:    borderRightStyle,
		BorderTopStyle:      borderTopStyle,
		BorderBottomStyle:   borderBottomStyle,
		BorderLeftColor:     borderLeftColor,
		BorderRightColor:    borderRightColor,
		BorderTopColor:      borderTopColor,
		BorderBottomColor:   borderBottomColor,
		BorderLeftWidthMM:   borderLeftWidthMM,
		BorderRightWidthMM:  borderRightWidthMM,
		BorderTopWidthMM:    borderTopWidthMM,
		BorderBottomWidthMM: borderBottomWidthMM,
		FillColor:           fillColor,
	}
	if hasText {
		spec.Text = &text
	}
	return spec, backgroundColor, nil
}

func tableCellContainerStyleHasChanges(spec hwpx.TableCellStyleSpec) bool {
	return spec.VertAlign != "" ||
		spec.MarginLeftMM != nil ||
		spec.MarginRightMM != nil ||
		spec.MarginTopMM != nil ||
		spec.MarginBottomMM != nil ||
		spec.BorderStyle != "" ||
		spec.BorderColor != "" ||
		spec.BorderWidthMM != nil ||
		spec.BorderLeftStyle != "" ||
		spec.BorderRightStyle != "" ||
		spec.BorderTopStyle != "" ||
		spec.BorderBottomStyle != "" ||
		spec.BorderLeftColor != "" ||
		spec.BorderRightColor != "" ||
		spec.BorderTopColor != "" ||
		spec.BorderBottomColor != "" ||
		spec.BorderLeftWidthMM != nil ||
		spec.BorderRightWidthMM != nil ||
		spec.BorderTopWidthMM != nil ||
		spec.BorderBottomWidthMM != nil ||
		spec.FillColor != ""
}

func parseTableCellBorderStyle(values map[string]string, key string) (string, error) {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return "", nil
	}

	value = strings.ToUpper(value)
	if value == "DOUBLE" {
		value = "DOUBLE_SLIM"
	}
	if value != "NONE" && value != "SOLID" && value != "DASH" && value != "DOUBLE_SLIM" {
		return "", commandError{
			message: fmt.Sprintf("set-table-cell --%s must be NONE, SOLID, DASH, or DOUBLE_SLIM", key),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return value, nil
}

func stringPointerIf(ok bool, value string) *string {
	if !ok {
		return nil
	}
	return &value
}

func paragraphCountIf(ok bool, value string) int {
	if !ok {
		return 0
	}
	return len(splitParagraphs(value))
}

func runSetTableCellLayout(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	tableIndex, row, col, paragraphIndex, err := parseTableCellParagraphTarget(opts.values, true)
	if err != nil {
		return err
	}
	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	spec, err := parseParagraphLayoutSpec(opts.values, "set-table-cell-layout")
	if err != nil {
		return err
	}

	report, paraPrID, err := hwpx.SetTableCellParagraphLayout(opts.input, selector, tableIndex, row, col, paragraphIndex, spec)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-table-cell-layout", fmt.Sprintf("Updated table %d cell (%d,%d) paragraph %d layout", tableIndex, row, col, paragraphIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-table-cell-layout",
			Success:       true,
			Data: tableCellParagraphLayoutResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				TableIndex:   tableIndex,
				Row:          row,
				Col:          col,
				Paragraph:    paragraphIndex,
				ParaPrIDRef:  paraPrID,
				Align:        spec.Align,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated section %d table #%d cell (%d,%d) paragraph %d layout in %s\n", resolveSelectedSectionIndex(selector), tableIndex, row, col, paragraphIndex, opts.input)
	return err
}

func runSetTableCellTextStyle(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	tableIndex, row, col, paragraphIndex, err := parseTableCellParagraphTarget(opts.values, true)
	if err != nil {
		return err
	}
	selector, err := parseSectionSelector(opts.values, false)
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

	spec, err := parseTextStyleSpec(opts.values, "set-table-cell-text-style")
	if err != nil {
		return err
	}

	report, charPrIDs, appliedRuns, err := hwpx.SetTableCellTextStyle(opts.input, selector, tableIndex, row, col, paragraphIndex, runIndex, spec)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-table-cell-text-style", fmt.Sprintf("Updated table %d cell (%d,%d) paragraph %d style", tableIndex, row, col, paragraphIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-table-cell-text-style",
			Success:       true,
			Data: tableCellTextStyleResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				TableIndex:   tableIndex,
				Row:          row,
				Col:          col,
				Paragraph:    paragraphIndex,
				Run:          runIndex,
				AppliedRuns:  appliedRuns,
				CharPrIDs:    charPrIDs,
				Bold:         spec.Bold,
				Italic:       spec.Italic,
				Underline:    spec.Underline,
				TextColor:    spec.TextColor,
				FontName:     spec.FontName,
				FontSizePt:   spec.FontSizePt,
				Report:       report,
			},
		})
	}

	if runIndex != nil {
		_, err = fmt.Fprintf(stdout, "Updated section %d table #%d cell (%d,%d) paragraph %d run %d style in %s\n", resolveSelectedSectionIndex(selector), tableIndex, row, col, paragraphIndex, *runIndex, opts.input)
		return err
	}

	_, err = fmt.Fprintf(stdout, "Updated section %d table #%d cell (%d,%d) paragraph %d style across %d run(s) in %s\n", resolveSelectedSectionIndex(selector), tableIndex, row, col, paragraphIndex, appliedRuns, opts.input)
	return err
}

func runMergeTableCells(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	tableIndex, err := requireIntArg(opts.values, "table")
	if err != nil {
		return err
	}
	startRow, err := requireIntArg(opts.values, "start-row")
	if err != nil {
		return err
	}
	startCol, err := requireIntArg(opts.values, "start-col")
	if err != nil {
		return err
	}
	endRow, err := requireIntArg(opts.values, "end-row")
	if err != nil {
		return err
	}
	endCol, err := requireIntArg(opts.values, "end-col")
	if err != nil {
		return err
	}
	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, err := hwpx.MergeTableCells(opts.input, selector, tableIndex, startRow, startCol, endRow, endCol)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "merge-table-cells", fmt.Sprintf("Merged table %d cells (%d,%d)-(%d,%d)", tableIndex, startRow, startCol, endRow, endCol), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "merge-table-cells",
			Success:       true,
			Data: tableMergeResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				TableIndex:   tableIndex,
				StartRow:     startRow,
				StartCol:     startCol,
				EndRow:       endRow,
				EndCol:       endCol,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Merged section %d table #%d cells (%d,%d) to (%d,%d) in %s\n", resolveSelectedSectionIndex(selector), tableIndex, startRow, startCol, endRow, endCol, opts.input)
	return err
}

func runNormalizeTableBorders(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	tableIndex, err := requireIntArg(opts.values, "table")
	if err != nil {
		return err
	}
	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, err := hwpx.NormalizeTableBorders(opts.input, selector, tableIndex)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "normalize-table-borders", fmt.Sprintf("Normalized table %d borders", tableIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "normalize-table-borders",
			Success:       true,
			Data: tableBorderNormalizeResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				TableIndex:   tableIndex,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Normalized section %d table #%d borders in %s\n", resolveSelectedSectionIndex(selector), tableIndex, opts.input)
	return err
}

func parseTableCellParagraphTarget(values map[string]string, requireParagraph bool) (int, int, int, int, error) {
	tableIndex, err := requireIntArg(values, "table")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	row, err := requireIntArg(values, "row")
	if err != nil {
		return 0, 0, 0, 0, err
	}
	col, err := requireIntArg(values, "col")
	if err != nil {
		return 0, 0, 0, 0, err
	}

	paragraphIndex := 0
	if requireParagraph {
		paragraphIndex, err = requireIntArg(values, "paragraph")
		if err != nil {
			return 0, 0, 0, 0, err
		}
	}

	return tableIndex, row, col, paragraphIndex, nil
}

func parseParagraphLayoutSpec(values map[string]string, commandName string) (hwpx.ParagraphLayoutSpec, error) {
	align := strings.ToUpper(strings.TrimSpace(values["align"]))
	if align != "" && !isAllowedValue(align, "LEFT", "RIGHT", "CENTER", "JUSTIFY", "DISTRIBUTE", "DISTRIBUTE_SPACE") {
		return hwpx.ParagraphLayoutSpec{}, commandError{
			message: commandName + " requires a valid --align",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	indentMM, err := optionalFloatPointer(values, "indent-mm")
	if err != nil {
		return hwpx.ParagraphLayoutSpec{}, err
	}
	leftMarginMM, err := optionalFloatPointer(values, "left-margin-mm")
	if err != nil {
		return hwpx.ParagraphLayoutSpec{}, err
	}
	rightMarginMM, err := optionalFloatPointer(values, "right-margin-mm")
	if err != nil {
		return hwpx.ParagraphLayoutSpec{}, err
	}
	spaceBeforeMM, err := optionalFloatPointer(values, "space-before-mm")
	if err != nil {
		return hwpx.ParagraphLayoutSpec{}, err
	}
	spaceAfterMM, err := optionalFloatPointer(values, "space-after-mm")
	if err != nil {
		return hwpx.ParagraphLayoutSpec{}, err
	}
	lineSpacingPercent, err := optionalIntPointer(values, "line-spacing-percent")
	if err != nil {
		return hwpx.ParagraphLayoutSpec{}, err
	}
	if lineSpacingPercent != nil && *lineSpacingPercent <= 0 {
		return hwpx.ParagraphLayoutSpec{}, commandError{
			message: commandName + " requires positive --line-spacing-percent",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if align == "" && indentMM == nil && leftMarginMM == nil && rightMarginMM == nil && spaceBeforeMM == nil && spaceAfterMM == nil && lineSpacingPercent == nil {
		return hwpx.ParagraphLayoutSpec{}, commandError{
			message: commandName + " requires at least one layout option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	return hwpx.ParagraphLayoutSpec{
		Align:              align,
		IndentMM:           indentMM,
		LeftMarginMM:       leftMarginMM,
		RightMarginMM:      rightMarginMM,
		SpaceBeforeMM:      spaceBeforeMM,
		SpaceAfterMM:       spaceAfterMM,
		LineSpacingPercent: lineSpacingPercent,
	}, nil
}

func parseTextStyleSpec(values map[string]string, commandName string) (hwpx.TextStyleSpec, error) {
	bold, err := parseOptionalBoolArg(values, "bold")
	if err != nil {
		return hwpx.TextStyleSpec{}, err
	}
	italic, err := parseOptionalBoolArg(values, "italic")
	if err != nil {
		return hwpx.TextStyleSpec{}, err
	}
	underline, err := parseOptionalBoolArg(values, "underline")
	if err != nil {
		return hwpx.TextStyleSpec{}, err
	}
	textColor, err := parseOptionalColorArg(values, "text-color")
	if err != nil {
		return hwpx.TextStyleSpec{}, err
	}
	fontName := strings.TrimSpace(values["font-name"])
	fontSizePt, err := optionalPositiveFloatPointer(values, "font-size-pt")
	if err != nil {
		return hwpx.TextStyleSpec{}, err
	}
	if bold == nil && italic == nil && underline == nil && textColor == "" && fontName == "" && fontSizePt == nil {
		return hwpx.TextStyleSpec{}, commandError{
			message: commandName + " requires at least one style option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	return hwpx.TextStyleSpec{
		Bold:       bold,
		Italic:     italic,
		Underline:  underline,
		TextColor:  textColor,
		FontName:   fontName,
		FontSizePt: fontSizePt,
	}, nil
}

func runSplitTableCell(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
	selector, err := parseSectionSelector(opts.values, false)
	if err != nil {
		return err
	}

	report, err := hwpx.SplitTableCell(opts.input, selector, tableIndex, row, col)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "split-table-cell", fmt.Sprintf("Split table %d cell (%d,%d)", tableIndex, row, col), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "split-table-cell",
			Success:       true,
			Data: tableCellEditResult{
				InputPath:    absolutePath(opts.input),
				SectionIndex: resolveSelectedSectionIndex(selector),
				SectionPath:  resolveSelectedSectionPath(selector),
				TableIndex:   tableIndex,
				Row:          row,
				Col:          col,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Split section %d table #%d cell (%d,%d) in %s\n", resolveSelectedSectionIndex(selector), tableIndex, row, col, opts.input)
	return err
}
