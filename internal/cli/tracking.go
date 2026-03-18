package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/zarathu/project-hwpx-cli/internal/hwpx"
)

var mutatingCommandNames = map[string]struct{}{
	"append-text":           {},
	"add-run-text":          {},
	"set-run-text":          {},
	"replace-runs-by-style": {},
	"set-paragraph-text":    {},
	"set-paragraph-layout":  {},
	"set-paragraph-list":    {},
	"set-text-style":        {},
	"delete-paragraph":      {},
	"add-section":           {},
	"delete-section":        {},
	"add-table":             {},
	"add-nested-table":      {},
	"set-table-cell":        {},
	"merge-table-cells":     {},
	"split-table-cell":      {},
	"embed-image":           {},
	"insert-image":          {},
	"set-object-position":   {},
	"set-header":            {},
	"set-footer":            {},
	"remove-header":         {},
	"remove-footer":         {},
	"set-page-number":       {},
	"set-columns":           {},
	"set-page-layout":       {},
	"add-footnote":          {},
	"add-endnote":           {},
	"add-memo":              {},
	"add-bookmark":          {},
	"add-hyperlink":         {},
	"add-heading":           {},
	"insert-toc":            {},
	"add-cross-reference":   {},
	"add-equation":          {},
	"add-line":              {},
	"add-ellipse":           {},
	"add-rectangle":         {},
	"add-textbox":           {},
}

var trackingOptionSpecs = []optionSpec{
	{Name: "--track-changes", Values: []string{"true", "false"}, Required: false, Description: "Opt-in historyEntry recording for this mutation."},
	{Name: "--change-author", Required: false, Description: "Optional author stored in historyEntry. Defaults to hwpxctl."},
	{Name: "--change-summary", Required: false, Description: "Optional historyEntry summary. Falls back to the command action text."},
}

func maybeRecordChange(opts namedCommandOptions, commandName, fallbackSummary string, report *hwpx.Report) error {
	enabled, err := parseOptionalBoolArg(opts.values, "track-changes")
	if err != nil {
		return err
	}
	if enabled == nil || !*enabled {
		return nil
	}

	summary := strings.TrimSpace(opts.values["change-summary"])
	if summary == "" {
		summary = strings.TrimSpace(fallbackSummary)
	}
	if summary == "" {
		summary = commandName
	}

	author := strings.TrimSpace(opts.values["change-author"])
	if author == "" {
		author = "hwpxctl"
	}

	if err := hwpx.RecordHistory(opts.input, hwpx.HistoryEntrySpec{
		Command:   commandName,
		Author:    author,
		Summary:   summary,
		Timestamp: time.Now(),
	}); err != nil {
		return fmt.Errorf("record history entry: %w", err)
	}

	if report == nil {
		return nil
	}

	updatedReport, err := hwpx.Validate(opts.input)
	if err != nil {
		return err
	}
	*report = updatedReport
	return nil
}

func decorateSchemaDoc(doc schemaDoc) schemaDoc {
	for index, command := range doc.Commands {
		if _, ok := mutatingCommandNames[command.Name]; !ok {
			continue
		}
		doc.Commands[index].Options = appendUniqueOptionSpecs(command.Options, trackingOptionSpecs...)
	}
	return doc
}

func appendUniqueOptionSpecs(options []optionSpec, extras ...optionSpec) []optionSpec {
	merged := append([]optionSpec{}, options...)
	for _, extra := range extras {
		if hasOptionSpec(merged, extra.Name) {
			continue
		}
		merged = append(merged, extra)
	}
	return merged
}

func hasOptionSpec(options []optionSpec, name string) bool {
	for _, option := range options {
		if option.Name == name {
			return true
		}
	}
	return false
}
