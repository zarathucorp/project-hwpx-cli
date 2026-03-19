package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx"
)

type fillTemplateMappingFile struct {
	Replacements []hwpx.FillTemplateReplacement `json:"replacements"`
}

func runInspect(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	report, err := hwpx.Inspect(opts.input)
	if err != nil {
		return err
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "inspect",
			Success:       true,
			Data: inspectResult{
				InputPath: absolutePath(opts.input),
				Report:    report,
			},
		})
	case formatText:
		return writeInspectText(stdout, opts.input, report)
	default:
		return writeJSON(stdout, report)
	}
}

func runValidate(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	report, err := hwpx.Validate(opts.input)
	if err != nil {
		return err
	}

	switch opts.format {
	case formatJSON:
		if !report.Valid {
			return commandError{
				message: "validation failed",
				code:    1,
				kind:    "validation_failed",
				data: validateResult{
					InputPath: absolutePath(opts.input),
					Report:    report,
				},
			}
		}

		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "validate",
			Success:       true,
			Data: validateResult{
				InputPath: absolutePath(opts.input),
				Report:    report,
			},
		})
	case formatText:
		if err := writeValidateText(stdout, opts.input, report); err != nil {
			return err
		}
	default:
		if err := writeJSON(stdout, report); err != nil {
			return err
		}
	}

	if !report.Valid {
		return commandError{
			message: "validation failed",
			code:    1,
			kind:    "validation_failed",
		}
	}
	return nil
}

func runAnalyzeTemplate(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	analysis, err := hwpx.AnalyzeTemplate(opts.input)
	if err != nil {
		return err
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "analyze-template",
			Success:       true,
			Data: templateAnalysisResult{
				InputPath: absolutePath(opts.input),
				Analysis:  analysis,
			},
		})
	default:
		_, err = fmt.Fprintf(
			stdout,
			"input: %s\nsections: %d\ntables: %d\nplaceholders: %d\nguides: %d\n",
			absolutePath(opts.input),
			analysis.SectionCount,
			analysis.TableCount,
			analysis.PlaceholderCount,
			analysis.GuideCount,
		)
		return err
	}
}

func runFindTargets(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	query := hwpx.TargetQuery{
		Anchor:      opts.values["anchor"],
		NearText:    opts.values["near-text"],
		TableLabel:  opts.values["table-label"],
		Placeholder: opts.values["placeholder"],
	}
	if strings.TrimSpace(query.Anchor) == "" &&
		strings.TrimSpace(query.NearText) == "" &&
		strings.TrimSpace(query.TableLabel) == "" &&
		strings.TrimSpace(query.Placeholder) == "" {
		return commandError{
			message: "find-targets requires at least one of --anchor, --near-text, --table-label, or --placeholder",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	matches, err := hwpx.FindTargets(opts.input, query)
	if err != nil {
		return err
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "find-targets",
			Success:       true,
			Data: findTargetsResult{
				InputPath: absolutePath(opts.input),
				Query:     query,
				Count:     len(matches),
				Matches:   matches,
			},
		})
	default:
		if len(matches) == 0 {
			_, err = fmt.Fprintln(stdout, "No matching targets found")
			return err
		}

		for _, match := range matches {
			if _, err := fmt.Fprintf(
				stdout,
				"kind=%s query=%s section=%d paragraph=%s table=%s cell=%s style=%q label=%q reason=%q text=%q\n",
				match.Kind,
				match.QueryType,
				match.SectionIndex,
				formatOptionalInt(match.ParagraphIndex),
				formatOptionalInt(match.TableIndex),
				formatAnalysisCell(match.Cell),
				match.StyleSummary,
				match.LabelText,
				match.Reason,
				match.Text,
			); err != nil {
				return err
			}
		}
		return nil
	}
}

func runRemoveGuides(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	dryRun, err := parseOptionalBoolArg(opts.values, "dry-run")
	if err != nil {
		return err
	}
	shouldDryRun := true
	if dryRun != nil {
		shouldDryRun = *dryRun
	}

	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}
	if selector.Section == nil && !selector.AllSections {
		selector.AllSections = true
	}

	reasonFilter := strings.TrimSpace(opts.values["reason"])
	result, err := buildRemoveGuidesResult(opts.input, selector, reasonFilter)
	if err != nil {
		return err
	}

	if !shouldDryRun {
		info, statErr := os.Stat(opts.input)
		if statErr != nil {
			return statErr
		}
		if !info.IsDir() {
			return commandError{
				message: "remove-guides apply mode requires an unpacked directory input",
				code:    1,
				kind:    "invalid_arguments",
			}
		}

		var applyReport hwpx.Report
		var appliedCandidates []hwpx.TemplateTextCandidate
		if err := withMutationLock(opts.input, "remove-guides", func() error {
			report, candidates, applyErr := hwpx.RemoveGuides(opts.input, selector, reasonFilter)
			if applyErr != nil {
				return applyErr
			}
			applyReport = report
			appliedCandidates = candidates
			return nil
		}); err != nil {
			return err
		}
		result.DryRun = false
		result.Applied = true
		result.Count = len(appliedCandidates)
		result.GuideCandidates = appliedCandidates
		result.Report = &applyReport
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "remove-guides",
			Success:       true,
			Data:          result,
		})
	default:
		if _, err := fmt.Fprintf(stdout, "input: %s\ndry-run: %t\ncandidates: %d\n", result.InputPath, result.DryRun, result.Count); err != nil {
			return err
		}
		for _, candidate := range result.GuideCandidates {
			if _, err := fmt.Fprintf(
				stdout,
				"section=%d paragraph=%d table=%s cell=%s reason=%s style=%q text=%q\n",
				candidate.SectionIndex,
				candidate.ParagraphIndex,
				formatOptionalInt(candidate.TableIndex),
				formatAnalysisCell(candidate.Cell),
				candidate.Reason,
				candidate.StyleSummary,
				candidate.Text,
			); err != nil {
				return err
			}
		}
		if result.Report != nil {
			_, err = fmt.Fprintf(stdout, "render-safe: %t\n", result.Report.RenderSafe)
			return err
		}
		return nil
	}
}

func runFillTemplate(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	info, err := os.Stat(opts.input)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return commandError{
			message: "fill-template requires an unpacked directory input",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	mappingPath := strings.TrimSpace(opts.values["mapping"])
	if mappingPath == "" {
		return commandError{
			message: "fill-template requires --mapping",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	replacements, err := readFillTemplateMapping(mappingPath)
	if err != nil {
		return err
	}
	if len(replacements) == 0 {
		return commandError{
			message: "mapping file does not contain any replacements",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	dryRun, err := parseOptionalBoolArg(opts.values, "dry-run")
	if err != nil {
		return err
	}
	shouldDryRun := true
	if dryRun != nil {
		shouldDryRun = *dryRun
	}

	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}
	if selector.Section == nil && !selector.AllSections {
		selector.AllSections = true
	}

	changes, err := hwpx.PlanFillTemplate(opts.input, selector, replacements)
	if err != nil {
		return err
	}

	result := fillTemplateResult{
		InputPath:   absolutePath(opts.input),
		MappingPath: absolutePath(mappingPath),
		DryRun:      shouldDryRun,
		Applied:     false,
		Count:       len(changes),
		Changes:     changes,
	}

	if !shouldDryRun {
		var report hwpx.Report
		var applied []hwpx.FillTemplateChange
		appliedReport, appliedChanges, err := hwpx.FillTemplate(opts.input, selector, replacements)
		if err != nil {
			return err
		}
		report = appliedReport
		applied = appliedChanges
		result.DryRun = false
		result.Applied = true
		result.Count = len(applied)
		result.Changes = applied
		result.Report = &report
		if err := maybeRecordChange(opts, "fill-template", "Fill template values", result.Report); err != nil {
			return err
		}
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "fill-template",
			Success:       true,
			Data:          result,
		})
	default:
		if _, err := fmt.Fprintf(stdout, "input: %s\nmapping: %s\ndry-run: %t\nchanges: %d\n", result.InputPath, result.MappingPath, result.DryRun, result.Count); err != nil {
			return err
		}
		for _, change := range result.Changes {
			if _, err := fmt.Fprintf(
				stdout,
				"kind=%s mode=%s section=%d paragraph=%s table=%s cell=%s selector=%q previous=%q text=%q\n",
				change.Kind,
				change.Mode,
				change.SectionIndex,
				formatOptionalInt(change.ParagraphIndex),
				formatOptionalInt(change.TableIndex),
				formatCellCoordinate(change.Cell),
				change.Selector,
				change.PreviousText,
				change.Text,
			); err != nil {
				return err
			}
		}
		if result.Report != nil {
			_, err = fmt.Fprintf(stdout, "render-safe: %t\n", result.Report.RenderSafe)
			return err
		}
		return nil
	}
}

func runRoundtripCheck(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	check, err := hwpx.RoundtripCheck(opts.input)
	if err != nil {
		return err
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "roundtrip-check",
			Success:       true,
			Data: roundtripCheckResult{
				InputPath: absolutePath(opts.input),
				Check:     check,
			},
		})
	default:
		if _, err := fmt.Fprintf(stdout, "input: %s\npassed: %t\nissues: %d\n", absolutePath(opts.input), check.Passed, len(check.Issues)); err != nil {
			return err
		}
		for _, issue := range check.Issues {
			if _, err := fmt.Fprintf(stdout, "%s [%s]: %s\n", issue.Code, issue.Severity, issue.Message); err != nil {
				return err
			}
		}
		return nil
	}
}

func buildRemoveGuidesResult(inputPath string, selector hwpx.SectionSelector, reasonFilter string) (removeGuidesResult, error) {
	analysis, err := hwpx.AnalyzeTemplate(inputPath)
	if err != nil {
		return removeGuidesResult{}, err
	}

	filtered := make([]hwpx.TemplateTextCandidate, 0, len(analysis.Guides))
	affectedSections := map[int]struct{}{}
	for _, candidate := range analysis.Guides {
		if selector.Section != nil && candidate.SectionIndex != *selector.Section {
			continue
		}
		if reasonFilter != "" && !strings.EqualFold(candidate.Reason, reasonFilter) {
			continue
		}
		filtered = append(filtered, candidate)
		affectedSections[candidate.SectionIndex] = struct{}{}
	}

	sections := make([]int, 0, len(affectedSections))
	for sectionIndex := range affectedSections {
		sections = append(sections, sectionIndex)
	}
	sort.Ints(sections)

	return removeGuidesResult{
		InputPath:        absolutePath(inputPath),
		DryRun:           true,
		Applied:          false,
		Count:            len(filtered),
		Reason:           reasonFilter,
		AffectedSections: sections,
		GuideCandidates:  filtered,
	}, nil
}

func readFillTemplateMapping(path string) ([]hwpx.FillTemplateReplacement, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wrapped fillTemplateMappingFile
	if err := json.Unmarshal(content, &wrapped); err == nil && len(wrapped.Replacements) > 0 {
		return wrapped.Replacements, nil
	}

	var direct []hwpx.FillTemplateReplacement
	if err := json.Unmarshal(content, &direct); err != nil {
		return nil, commandError{
			message: fmt.Sprintf("invalid mapping json: %v", err),
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	return direct, nil
}

func runText(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	text, err := hwpx.ExtractText(opts.input)
	if err != nil {
		return err
	}

	if opts.output != "" {
		if err := os.MkdirAll(filepath.Dir(opts.output), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(opts.output, []byte(text), 0o644); err != nil {
			return err
		}
	}

	switch opts.format {
	case formatJSON:
		result := textResult{
			InputPath:      absolutePath(opts.input),
			LineCount:      countLines(text),
			CharacterCount: utf8.RuneCountInString(text),
		}
		if opts.output != "" {
			result.OutputPath = absolutePath(opts.output)
		} else {
			result.Text = text
		}
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "text",
			Success:       true,
			Data:          result,
		})
	default:
		if opts.output == "" {
			_, err = fmt.Fprintln(stdout, text)
			return err
		}
		return nil
	}
}

func runUnpack(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}
	if opts.output == "" {
		return commandError{
			message: "unpack requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	if err := hwpx.Unpack(opts.input, opts.output); err != nil {
		return err
	}

	if opts.format == formatJSON {
		report, err := hwpx.Validate(opts.output)
		if err != nil {
			return err
		}
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "unpack",
			Success:       true,
			Data: unpackResult{
				InputPath:  absolutePath(opts.input),
				OutputPath: absolutePath(opts.output),
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Unpacked to %s\n", opts.output)
	return err
}

func runPack(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}
	if opts.output == "" {
		return commandError{
			message: "pack requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	if err := hwpx.Pack(opts.input, opts.output); err != nil {
		return err
	}

	if opts.format == formatJSON {
		report, err := hwpx.Validate(opts.output)
		if err != nil {
			return err
		}
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "pack",
			Success:       true,
			Data: packResult{
				InputPath:  absolutePath(opts.input),
				OutputPath: absolutePath(opts.output),
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Packed to %s\n", opts.output)
	return err
}

func runCreate(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, false)
	if err != nil {
		return err
	}
	if opts.input != "" {
		return commandError{
			message: "create does not accept a positional input path",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
	if opts.output == "" {
		return commandError{
			message: "create requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	report, err := hwpx.CreateEditableDocument(opts.output)
	if err != nil {
		return err
	}

	if opts.format == formatJSON {
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "create",
			Success:       true,
			Data: createResult{
				OutputPath: absolutePath(opts.output),
				Report:     report,
			},
		})
	}

	_, err = fmt.Fprintf(stdout, "Created editable document at %s\n", opts.output)
	return err
}
