package cli

import (
	"io"

	"github.com/spf13/cobra"
)

func runSchema(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, false)
	if err != nil {
		return err
	}
	if !opts.formatExplicit {
		opts.format = formatJSON
	}
	if opts.input != "" {
		return commandError{
			message: "schema does not accept a positional input path",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	doc := buildSchemaDoc()
	if opts.format == formatText {
		writeSchemaText(stdout, doc)
		return nil
	}
	return writeJSON(stdout, doc)
}
