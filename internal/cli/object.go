package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

func runAddRectangle(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	widthMM, err := parseOptionalFloatArg(opts.values, "width-mm")
	if err != nil {
		return err
	}
	heightMM, err := parseOptionalFloatArg(opts.values, "height-mm")
	if err != nil {
		return err
	}
	if widthMM <= 0 || heightMM <= 0 {
		return commandError{
			message: "add-rectangle requires positive --width-mm and --height-mm",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	lineColor := strings.TrimSpace(opts.values["line-color"])
	fillColor := strings.TrimSpace(opts.values["fill-color"])
	report, shapeID, width, height, err := hwpx.AddRectangle(opts.input, hwpx.RectangleSpec{
		WidthMM:   widthMM,
		HeightMM:  heightMM,
		LineColor: lineColor,
		FillColor: fillColor,
	})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-rectangle",
			Success:       true,
			Data: rectangleResult{
				InputPath: absolutePath(opts.input),
				ShapeID:   shapeID,
				Width:     width,
				Height:    height,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added rectangle %s to %s\n", shapeID, opts.input)
	return err
}

func runAddEquation(args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(args, defaultFormat, true)
	if err != nil {
		return err
	}

	script := strings.TrimSpace(opts.values["script"])
	if script == "" {
		return commandError{
			message: "missing required --script",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, itemID, err := hwpx.AddEquation(opts.input, hwpx.EquationSpec{
		Script: script,
	})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-equation",
			Success:       true,
			Data: equationResult{
				InputPath: absolutePath(opts.input),
				Script:    script,
				ItemID:    itemID,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added equation %s to %s\n", itemID, opts.input)
	return err
}
