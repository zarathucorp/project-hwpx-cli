package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx"
)

var supportedObjectTypes = map[string]struct{}{
	"table":     {},
	"image":     {},
	"equation":  {},
	"rectangle": {},
	"line":      {},
	"ellipse":   {},
	"textbox":   {},
}

func runFindObjects(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	objectTypes, err := parseObjectTypesArg(opts.values["type"])
	if err != nil {
		return err
	}
	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}

	matches, err := hwpx.FindObjects(opts.input, selector, hwpx.ObjectFilter{Types: objectTypes})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "find-objects",
			Success:       true,
			Data: objectSearchResult{
				InputPath: absolutePath(opts.input),
				Count:     len(matches),
				Matches:   matches,
			},
		})
	}

	if len(matches) == 0 {
		_, err = fmt.Fprintln(stdout, "No objects found")
		return err
	}

	for _, match := range matches {
		if _, err := fmt.Fprintf(stdout, "index=%d section=%d type=%s paragraph=%d run=%d table=%s cell=%s path=%s id=%s ref=%s text=%q\n", match.Index, match.SectionIndex, match.Type, match.ParagraphIndex, match.Run, formatOptionalInt(match.TableIndex), formatCellCoordinate(match.Cell), match.Path, match.ID, match.Ref, match.Text); err != nil {
			return err
		}
	}
	return nil
}

func runFindByTag(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	tag := strings.TrimSpace(opts.values["tag"])
	if tag == "" {
		return commandError{
			message: "find-by-tag requires --tag",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}

	matches, err := hwpx.FindByTag(opts.input, selector, hwpx.TagFilter{Tag: tag})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "find-by-tag",
			Success:       true,
			Data: tagSearchResult{
				InputPath: absolutePath(opts.input),
				Count:     len(matches),
				Matches:   matches,
			},
		})
	}

	if len(matches) == 0 {
		_, err = fmt.Fprintln(stdout, "No matching tags found")
		return err
	}

	for _, match := range matches {
		if _, err := fmt.Fprintf(stdout, "index=%d section=%d paragraph=%d run=%d table=%s cell=%s path=%s tag=%s text=%q\n", match.Index, match.SectionIndex, match.ParagraphIndex, match.Run, formatOptionalInt(match.TableIndex), formatCellCoordinate(match.Cell), match.Path, match.Tag, match.Text); err != nil {
			return err
		}
	}
	return nil
}

func runFindByAttr(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	attr := strings.TrimSpace(opts.values["attr"])
	if attr == "" {
		return commandError{
			message: "find-by-attr requires --attr",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}

	matches, err := hwpx.FindByAttr(opts.input, selector, hwpx.AttributeFilter{
		Attr:  attr,
		Value: strings.TrimSpace(opts.values["value"]),
		Tag:   strings.TrimSpace(opts.values["tag"]),
	})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "find-by-attr",
			Success:       true,
			Data: attributeSearchResult{
				InputPath: absolutePath(opts.input),
				Count:     len(matches),
				Matches:   matches,
			},
		})
	}

	if len(matches) == 0 {
		_, err = fmt.Fprintln(stdout, "No matching attributes found")
		return err
	}

	for _, match := range matches {
		if _, err := fmt.Fprintf(stdout, "index=%d section=%d paragraph=%d run=%d table=%s cell=%s path=%s tag=%s attr=%s value=%q text=%q\n", match.Index, match.SectionIndex, match.ParagraphIndex, match.Run, formatOptionalInt(match.TableIndex), formatCellCoordinate(match.Cell), match.Path, match.Tag, match.Attr, match.Value, match.Text); err != nil {
			return err
		}
	}
	return nil
}

func runFindByXPath(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	expr := strings.TrimSpace(opts.values["expr"])
	if expr == "" {
		return commandError{
			message: "find-by-xpath requires --expr",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}

	matches, err := hwpx.FindByXPath(opts.input, selector, hwpx.XPathFilter{Expr: expr})
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "find-by-xpath",
			Success:       true,
			Data: xpathSearchResult{
				InputPath: absolutePath(opts.input),
				Count:     len(matches),
				Matches:   matches,
			},
		})
	}

	if len(matches) == 0 {
		_, err = fmt.Fprintln(stdout, "No matching xpath results found")
		return err
	}

	for _, match := range matches {
		if _, err := fmt.Fprintf(stdout, "index=%d section=%d paragraph=%d run=%d table=%s cell=%s path=%s tag=%s text=%q\n", match.Index, match.SectionIndex, match.ParagraphIndex, match.Run, formatOptionalInt(match.TableIndex), formatCellCoordinate(match.Cell), match.Path, match.Tag, match.Text); err != nil {
			return err
		}
	}
	return nil
}

func formatOptionalInt(value *int) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *value)
}

func formatCellCoordinate(value *hwpx.TableCellCoordinate) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("(%d,%d)", value.Row, value.Col)
}

func parseObjectTypesArg(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	types := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		value := strings.ToLower(strings.TrimSpace(part))
		if value == "" {
			continue
		}
		if _, ok := supportedObjectTypes[value]; !ok {
			return nil, commandError{
				message: fmt.Sprintf("unsupported object type: %s", value),
				code:    1,
				kind:    "invalid_arguments",
			}
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		types = append(types, value)
	}
	return types, nil
}
