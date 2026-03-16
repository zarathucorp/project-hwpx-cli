package cli

import (
	"fmt"
	"io"

	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

func runAddTable(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
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

	report, tableIndex, err := hwpx.AddTable(opts.input, hwpx.TableSpec{
		Rows:  rows,
		Cols:  cols,
		Cells: cells,
	})
	if err != nil {
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

func runSetTableCell(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
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

func runMergeTableCells(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
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

func runSplitTableCell(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
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
