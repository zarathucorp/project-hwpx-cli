package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx"
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
	if err := maybeRecordChange(opts, "add-rectangle", fmt.Sprintf("Added rectangle %s", shapeID), &report); err != nil {
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
	if err := maybeRecordChange(opts, "add-line", fmt.Sprintf("Added line %s", shapeID), &report); err != nil {
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
	if err := maybeRecordChange(opts, "add-ellipse", fmt.Sprintf("Added ellipse %s", shapeID), &report); err != nil {
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
	if err := maybeRecordChange(opts, "add-textbox", fmt.Sprintf("Added textbox %s", shapeID), &report); err != nil {
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

func runSetObjectPosition(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	objectType := strings.ToLower(strings.TrimSpace(opts.values["type"]))
	if !isAllowedValue(objectType, "image", "rectangle", "line", "ellipse", "textbox") {
		return commandError{
			message: "set-object-position requires --type image, rectangle, line, ellipse, or textbox",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	index, err := requireIntArg(opts.values, "index")
	if err != nil {
		return err
	}

	treatAsChar, err := parseOptionalBoolArg(opts.values, "treat-as-char")
	if err != nil {
		return err
	}
	xmm, err := optionalFloatPointer(opts.values, "x-mm")
	if err != nil {
		return err
	}
	ymm, err := optionalFloatPointer(opts.values, "y-mm")
	if err != nil {
		return err
	}
	horzAlign := strings.ToUpper(strings.TrimSpace(opts.values["horz-align"]))
	vertAlign := strings.ToUpper(strings.TrimSpace(opts.values["vert-align"]))
	if horzAlign != "" && !isAllowedValue(horzAlign, "LEFT", "CENTER", "RIGHT") {
		return commandError{
			message: "set-object-position requires a valid --horz-align",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if vertAlign != "" && !isAllowedValue(vertAlign, "TOP", "CENTER", "BOTTOM") {
		return commandError{
			message: "set-object-position requires a valid --vert-align",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if treatAsChar == nil && xmm == nil && ymm == nil && horzAlign == "" && vertAlign == "" {
		return commandError{
			message: "set-object-position requires at least one position option",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, objectID, err := hwpx.SetObjectPosition(opts.input, hwpx.ObjectPositionSpec{
		Type:        objectType,
		Index:       index,
		TreatAsChar: treatAsChar,
		XMM:         xmm,
		YMM:         ymm,
		HorzAlign:   horzAlign,
		VertAlign:   vertAlign,
	})
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "set-object-position", fmt.Sprintf("Updated %s %d position", objectType, index), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "set-object-position",
			Success:       true,
			Data: objectPositionResult{
				InputPath:    absolutePath(opts.input),
				Type:         objectType,
				Index:        index,
				ObjectID:     objectID,
				TreatAsChar:  treatAsChar,
				XMM:          xmm,
				YMM:          ymm,
				HorzAlign:    horzAlign,
				VertAlign:    vertAlign,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Updated %s %d position in %s\n", objectType, index, opts.input)
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
	if err := maybeRecordChange(opts, "add-equation", fmt.Sprintf("Added equation %s", itemID), &report); err != nil {
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
