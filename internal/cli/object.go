package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

func runAddRectangle(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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

func runAddLine(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
			message: "add-line requires positive --width-mm and --height-mm",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	lineColor := strings.TrimSpace(opts.values["line-color"])
	report, shapeID, width, height, err := hwpx.AddLine(opts.input, hwpx.LineSpec{
		WidthMM:   widthMM,
		HeightMM:  heightMM,
		LineColor: lineColor,
	})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-line",
			Success:       true,
			Data: lineResult{
				InputPath: absolutePath(opts.input),
				ShapeID:   shapeID,
				Width:     width,
				Height:    height,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added line %s to %s\n", shapeID, opts.input)
	return err
}

func runAddEllipse(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
			message: "add-ellipse requires positive --width-mm and --height-mm",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	lineColor := strings.TrimSpace(opts.values["line-color"])
	fillColor := strings.TrimSpace(opts.values["fill-color"])
	report, shapeID, width, height, err := hwpx.AddEllipse(opts.input, hwpx.EllipseSpec{
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
			Command:       "add-ellipse",
			Success:       true,
			Data: ellipseResult{
				InputPath: absolutePath(opts.input),
				ShapeID:   shapeID,
				Width:     width,
				Height:    height,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added ellipse %s to %s\n", shapeID, opts.input)
	return err
}

func runAddTextBox(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
			message: "add-textbox requires positive --width-mm and --height-mm",
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

	lineColor := strings.TrimSpace(opts.values["line-color"])
	fillColor := strings.TrimSpace(opts.values["fill-color"])
	paragraphs := splitParagraphs(text)
	report, shapeID, width, height, err := hwpx.AddTextBox(opts.input, hwpx.TextBoxSpec{
		WidthMM:   widthMM,
		HeightMM:  heightMM,
		Text:      paragraphs,
		LineColor: lineColor,
		FillColor: fillColor,
	})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-textbox",
			Success:       true,
			Data: textBoxResult{
				InputPath: absolutePath(opts.input),
				ShapeID:   shapeID,
				Text:      paragraphs,
				Width:     width,
				Height:    height,
				Report:    report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added textbox %s to %s\n", shapeID, opts.input)
	return err
}

func runAddEquation(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
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
