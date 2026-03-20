package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

type commandRunner func(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error

func newRootCommand(stdout, stderr io.Writer, defaultFormat outputFormat) *cobra.Command {
	var showVersion bool

	root := &cobra.Command{
		Use:               "hwpxctl",
		Short:             "HWPX package inspection and editing CLI",
		Long:              "hwpxctl inspects, validates, extracts, and edits HWPX packages.",
		SilenceErrors:     true,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
				return err
			}
			return cmd.Help()
		},
	}

	root.SetOut(stdout)
	root.SetErr(stderr)
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version")
	root.PersistentFlags().String("format", "", "Select output mode: text or json")

	for _, spec := range buildSchemaDoc().Commands {
		handler, ok := lookupCommandRunner(spec.Name)
		if !ok {
			continue
		}
		root.AddCommand(newSubcommand(spec, handler, defaultFormat))
	}

	return root
}

func newSubcommand(spec commandSpec, handler commandRunner, defaultFormat outputFormat) *cobra.Command {
	command := &cobra.Command{
		Use:               commandUse(spec),
		Short:             spec.Summary,
		Long:              commandDescription(spec),
		Example:           strings.Join(spec.Examples, "\n"),
		SilenceErrors:     true,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, ok := mutatingCommandNames[spec.Name]; ok {
				targetDir := lockTargetFromArgs(args)
				if targetDir != "" {
					return withMutationLock(targetDir, spec.Name, func() error {
						return handler(cmd, args, cmd.OutOrStdout(), defaultFormat)
					})
				}
			}
			return handler(cmd, args, cmd.OutOrStdout(), defaultFormat)
		},
	}

	for _, option := range spec.Options {
		name := strings.TrimPrefix(option.Name, "--")
		description := option.Description

		if name == "format" {
			continue
		}

		if name == "output" {
			command.Flags().StringP(name, "o", "", description)
		} else {
			command.Flags().String(name, "", description)
		}

		if option.Required {
			_ = command.MarkFlagRequired(name)
		}
	}

	return command
}

func commandUse(spec commandSpec) string {
	if len(spec.Arguments) == 0 {
		return spec.Name
	}

	parts := []string{spec.Name}
	for _, argument := range spec.Arguments {
		placeholder := fmt.Sprintf("<%s>", argument.Name)
		if !argument.Required {
			placeholder = fmt.Sprintf("[%s]", argument.Name)
		}
		parts = append(parts, placeholder)
	}

	return strings.Join(parts, " ")
}

func commandDescription(spec commandSpec) string {
	var builder strings.Builder

	builder.WriteString(spec.Summary)

	if len(spec.Arguments) > 0 {
		builder.WriteString("\n\nArguments:\n")
		for _, argument := range spec.Arguments {
			required := "optional"
			if argument.Required {
				required = "required"
			}
			builder.WriteString(fmt.Sprintf("  %s (%s): %s\n", argument.Name, required, argument.Description))
		}
	}

	if len(spec.Options) > 0 {
		builder.WriteString("\nOptions:\n")
		for _, option := range spec.Options {
			if option.Name == "--format" {
				continue
			}
			required := "optional"
			if option.Required {
				required = "required"
			}
			builder.WriteString(fmt.Sprintf("  %s (%s): %s\n", option.Name, required, option.Description))
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

func lookupCommandRunner(name string) (commandRunner, bool) {
	switch name {
	case "inspect":
		return runInspect, true
	case "validate":
		return runValidate, true
	case "analyze-template":
		return runAnalyzeTemplate, true
	case "find-targets":
		return runFindTargets, true
	case "scaffold-template-contract":
		return runScaffoldTemplateContract, true
	case "remove-guides":
		return runRemoveGuides, true
	case "fill-template":
		return runFillTemplate, true
	case "preview-diff":
		return runPreviewDiff, true
	case "roundtrip-check":
		return runRoundtripCheck, true
	case "safe-pack":
		return runSafePack, true
	case "text":
		return runText, true
	case "export-markdown":
		return runExportMarkdown, true
	case "export-html":
		return runExportHTML, true
	case "unpack":
		return runUnpack, true
	case "pack":
		return runPack, true
	case "create":
		return runCreate, true
	case "append-text":
		return runAppendText, true
	case "add-run-text":
		return runAddRunText, true
	case "set-run-text":
		return runSetRunText, true
	case "find-runs-by-style":
		return runFindRunsByStyle, true
	case "replace-runs-by-style":
		return runReplaceRunsByStyle, true
	case "find-objects":
		return runFindObjects, true
	case "find-by-tag":
		return runFindByTag, true
	case "find-by-attr":
		return runFindByAttr, true
	case "find-by-xpath":
		return runFindByXPath, true
	case "set-paragraph-text":
		return runSetParagraphText, true
	case "set-paragraph-layout":
		return runSetParagraphLayout, true
	case "set-paragraph-list":
		return runSetParagraphList, true
	case "set-text-style":
		return runSetTextStyle, true
	case "delete-paragraph":
		return runDeleteParagraph, true
	case "add-section":
		return runAddSection, true
	case "delete-section":
		return runDeleteSection, true
	case "add-table":
		return runAddTable, true
	case "add-nested-table":
		return runAddNestedTable, true
	case "set-table-cell":
		return runSetTableCell, true
	case "set-table-cell-layout":
		return runSetTableCellLayout, true
	case "set-table-cell-text-style":
		return runSetTableCellTextStyle, true
	case "merge-table-cells":
		return runMergeTableCells, true
	case "split-table-cell":
		return runSplitTableCell, true
	case "normalize-table-borders":
		return runNormalizeTableBorders, true
	case "embed-image":
		return runEmbedImage, true
	case "insert-image":
		return runInsertImage, true
	case "set-object-position":
		return runSetObjectPosition, true
	case "set-header":
		return runSetHeader, true
	case "set-footer":
		return runSetFooter, true
	case "remove-header":
		return runRemoveHeader, true
	case "remove-footer":
		return runRemoveFooter, true
	case "set-page-number":
		return runSetPageNumber, true
	case "set-columns":
		return runSetColumns, true
	case "set-page-layout":
		return runSetPageLayout, true
	case "add-footnote":
		return func(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
			return runAddNote("footnote", cmd, args, stdout, defaultFormat)
		}, true
	case "add-endnote":
		return func(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
			return runAddNote("endnote", cmd, args, stdout, defaultFormat)
		}, true
	case "add-memo":
		return runAddMemo, true
	case "add-bookmark":
		return runAddBookmark, true
	case "add-hyperlink":
		return runAddHyperlink, true
	case "add-heading":
		return runAddHeading, true
	case "insert-toc":
		return runInsertTOC, true
	case "add-cross-reference":
		return runAddCrossReference, true
	case "add-equation":
		return runAddEquation, true
	case "add-line":
		return runAddLine, true
	case "add-ellipse":
		return runAddEllipse, true
	case "add-rectangle":
		return runAddRectangle, true
	case "add-textbox":
		return runAddTextBox, true
	case "schema":
		return runSchema, true
	default:
		return nil, false
	}
}

func normalizeCLIError(err error) error {
	if err == nil {
		return nil
	}

	var commandErr commandError
	if ok := errorAsCommandError(err, &commandErr); ok {
		return err
	}

	message := err.Error()
	if strings.HasPrefix(message, "unknown command ") {
		name := strings.TrimPrefix(message, "unknown command \"")
		if index := strings.Index(name, "\""); index >= 0 {
			name = name[:index]
		}
		return commandError{
			message: fmt.Sprintf("unknown command: %s", name),
			code:    1,
			kind:    "unknown_command",
		}
	}

	return commandError{
		message: message,
		code:    1,
		kind:    "invalid_arguments",
	}
}

func errorAsCommandError(err error, target *commandError) bool {
	if err == nil || target == nil {
		return false
	}

	value, ok := err.(commandError)
	if !ok {
		return false
	}

	*target = value
	return true
}

func detectCommandName(args []string) string {
	for index := 0; index < len(args); index++ {
		current := args[index]
		switch current {
		case "--format":
			index++
			continue
		case "-v", "--version", "-h", "--help":
			continue
		}
		if strings.HasPrefix(current, "-") {
			continue
		}
		return current
	}
	return ""
}
