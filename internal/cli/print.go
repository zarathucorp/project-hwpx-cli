package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

func runPrintPDF(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}
	if opts.output == "" {
		return commandError{
			message: "print-pdf requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	workspaceDir, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := hwpx.PrintToPDF(opts.input, opts.output, workspaceDir); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "print-pdf",
			Success:       true,
			Data: printPDFResult{
				InputPath:  absolutePath(opts.input),
				OutputPath: absolutePath(opts.output),
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Printed PDF to %s\n", opts.output)
	return err
}
