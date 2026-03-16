package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

func runEmbedImage(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	imagePath, ok := opts.values["image"]
	if !ok {
		return commandError{
			message: "embed-image requires --image",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, embedded, err := hwpx.EmbedImage(opts.input, imagePath)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "embed-image",
			Success:       true,
			Data: imageEmbedResult{
				InputPath:  absolutePath(opts.input),
				ImagePath:  absolutePath(imagePath),
				ItemID:     embedded.ItemID,
				BinaryPath: embedded.BinaryPath,
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Embedded image as %s in %s\n", embedded.ItemID, opts.input)
	return err
}

func runInsertImage(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	imagePath, ok := opts.values["image"]
	if !ok {
		return commandError{
			message: "insert-image requires --image",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	widthMM, err := parseOptionalFloatArg(opts.values, "width-mm")
	if err != nil {
		return err
	}

	report, placed, err := hwpx.InsertImage(opts.input, imagePath, widthMM)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "insert-image",
			Success:       true,
			Data: imageInsertResult{
				InputPath:    absolutePath(opts.input),
				ImagePath:    absolutePath(imagePath),
				ItemID:       placed.ItemID,
				BinaryPath:   placed.BinaryPath,
				PixelWidth:   placed.PixelWidth,
				PixelHeight:  placed.PixelHeight,
				PlacedWidth:  placed.Width,
				PlacedHeight: placed.Height,
				Report:       report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Inserted image %s into %s\n", placed.ItemID, opts.input)
	return err
}
