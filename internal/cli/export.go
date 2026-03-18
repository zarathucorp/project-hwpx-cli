package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/zarathucop/project-hwpx-cli/internal/hwpx"
)

func runExportMarkdown(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	return runExport("export-markdown", "markdown", hwpx.ExportMarkdown, cmd, args, stdout, defaultFormat)
}

func runExportHTML(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	return runExport("export-html", "html", hwpx.ExportHTML, cmd, args, stdout, defaultFormat)
}

func runExport(commandName, exportFormat string, exporter func(string) (string, int, error), cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	content, blocks, err := exporter(opts.input)
	if err != nil {
		return err
	}

	if opts.output != "" {
		if err := os.MkdirAll(filepath.Dir(opts.output), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(opts.output, []byte(content), 0o644); err != nil {
			return err
		}
	}

	result := exportResult{
		InputPath:      absolutePath(opts.input),
		Format:         exportFormat,
		LineCount:      countLines(content),
		CharacterCount: utf8.RuneCountInString(content),
		BlockCount:     blocks,
	}
	if opts.output != "" {
		result.OutputPath = absolutePath(opts.output)
	} else {
		result.Content = content
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       commandName,
			Success:       true,
			Data:          result,
		})
	}

	if opts.output == "" {
		_, err = fmt.Fprintln(stdout, content)
		return err
	}

	_, err = fmt.Fprintf(stdout, "Exported %s to %s\n", exportFormat, opts.output)
	return err
}
