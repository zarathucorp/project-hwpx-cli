package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
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

	report, tableIndex, err := hwpx.AddTable(opts.input, spec)
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

	report, err := hwpx.AddNestedTable(opts.input, tableIndex, row, col, spec)
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
				InputPath:  absolutePath(opts.input),
				TableIndex: tableIndex,
				Row:        row,
				Col:        col,
				Rows:       rows,
				Cols:       cols,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added nested table (%dx%d) to table #%d cell (%d,%d) in %s\n", rows, cols, tableIndex, row, col, opts.input)
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
	if err := maybeRecordChange(opts, "set-table-cell", fmt.Sprintf("Updated table %d cell (%d,%d)", tableIndex, row, col), &report); err != nil {
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

	report, err := hwpx.MergeTableCells(opts.input, tableIndex, startRow, startCol, endRow, endCol)
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
				InputPath:  absolutePath(opts.input),
				TableIndex: tableIndex,
				StartRow:   startRow,
				StartCol:   startCol,
				EndRow:     endRow,
				EndCol:     endCol,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Merged table #%d cells (%d,%d) to (%d,%d) in %s\n", tableIndex, startRow, startCol, endRow, endCol, opts.input)
	return err
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

	report, err := hwpx.SplitTableCell(opts.input, tableIndex, row, col)
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
				InputPath:  absolutePath(opts.input),
				TableIndex: tableIndex,
				Row:        row,
				Col:        col,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Split table #%d cell (%d,%d) in %s\n", tableIndex, row, col, opts.input)
	return err
}
