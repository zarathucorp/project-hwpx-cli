package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

func runAddSection(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	report, sectionIndex, sectionPath, err := hwpx.AddSection(opts.input)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "add-section", fmt.Sprintf("Added section %d", sectionIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "add-section",
			Success:       true,
			Data: sectionEditResult{
				InputPath:   absolutePath(opts.input),
				Section:     sectionIndex,
				SectionPath: sectionPath,
				Deleted:     false,
				Report:      report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Added section %d as %s in %s\n", sectionIndex, sectionPath, opts.input)
	return err
}

func runDeleteSection(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	sectionIndex, err := requireIntArg(opts.values, "section")
	if err != nil {
		return err
	}

	report, removedPath, err := hwpx.DeleteSection(opts.input, sectionIndex)
	if err != nil {
		return err
	}
	if err := maybeRecordChange(opts, "delete-section", fmt.Sprintf("Deleted section %d", sectionIndex), &report); err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "delete-section",
			Success:       true,
			Data: sectionEditResult{
				InputPath:   absolutePath(opts.input),
				Section:     sectionIndex,
				SectionPath: removedPath,
				Deleted:     true,
				RemovedPath: removedPath,
				Report:      report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Deleted section %d (%s) from %s\n", sectionIndex, removedPath, opts.input)
	return err
}
