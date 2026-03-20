package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx"
	"gopkg.in/yaml.v3"
)

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

func runScaffoldTemplateContract(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}

	strictValue, err := parseOptionalBoolArg(opts.values, "strict")
	if err != nil {
		return err
	}
	strict := true
	if strictValue != nil {
		strict = *strictValue
	}

	contract, err := hwpx.ScaffoldTemplateContract(
		opts.input,
		strings.TrimSpace(opts.values["template-id"]),
		strings.TrimSpace(opts.values["template-version"]),
		strict,
	)
	if err != nil {
		return err
	}

	payload, err := hwpx.ScaffoldTemplatePayload(contract)
	if err != nil {
		return err
	}

	contractFormat, err := resolveScaffoldTemplateArtifactFormat(strings.TrimSpace(opts.values["contract-format"]), opts.output)
	if err != nil {
		return err
	}
	contractContent, err := marshalScaffoldTemplateContract(contract, contractFormat)
	if err != nil {
		return err
	}
	if opts.output != "" {
		if err := os.MkdirAll(filepath.Dir(opts.output), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(opts.output, contractContent, 0o644); err != nil {
			return err
		}
	}

	payloadOutput := strings.TrimSpace(opts.values["payload-output"])
	payloadFormat, err := resolveScaffoldTemplateArtifactFormat(strings.TrimSpace(opts.values["payload-format"]), payloadOutput)
	if err != nil {
		return err
	}
	if payloadOutput != "" {
		payloadContent, err := marshalScaffoldTemplatePayload(payload, payloadFormat)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(payloadOutput), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(payloadOutput, payloadContent, 0o644); err != nil {
			return err
		}
	}

	result := scaffoldTemplateContractResult{
		InputPath:        absolutePath(opts.input),
		ContractFormat:   contractFormat,
		PayloadFormat:    payloadFormat,
		TemplateID:       contract.TemplateID,
		TemplateVersion:  contract.TemplateVersion,
		FieldCount:       len(contract.Fields),
		PlaceholderCount: len(contract.Fields),
		Contract:         contract,
		Payload:          payload,
	}
	if opts.output != "" {
		result.OutputPath = absolutePath(opts.output)
	}
	if payloadOutput != "" {
		result.PayloadOutputPath = absolutePath(payloadOutput)
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "scaffold-template-contract",
			Success:       true,
			Data:          result,
		})
	default:
		if opts.output != "" || payloadOutput != "" {
			_, err = fmt.Fprintf(
				stdout,
				"input: %s\noutput: %s\ncontract-format: %s\npayload-output: %s\npayload-format: %s\ntemplate-id: %s\ntemplate-version: %s\nfields: %d\n",
				result.InputPath,
				emptyDash(result.OutputPath),
				result.ContractFormat,
				emptyDash(result.PayloadOutputPath),
				emptyDash(result.PayloadFormat),
				result.TemplateID,
				result.TemplateVersion,
				result.FieldCount,
			)
			return err
		}
		_, err = stdout.Write(contractContent)
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
				"kind=%s query=%s section=%d paragraph=%s table=%s cell=%s style=%q label=%q reason=%q text=%q section-summary=%q table-summary=%q paragraph-summary=%q\n",
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
				formatFindTargetsSectionSummary(match.Context),
				formatFindTargetsTableSummary(match.Context),
				formatFindTargetsParagraphSummary(match.Context),
			); err != nil {
				return err
			}
		}
		return nil
	}
}

func resolveScaffoldTemplateArtifactFormat(requested string, outputPath string) (string, error) {
	requested = strings.ToLower(strings.TrimSpace(requested))
	switch requested {
	case "":
		if strings.EqualFold(filepath.Ext(outputPath), ".json") {
			return "json", nil
		}
		return "yaml", nil
	case "yaml", "json":
		return requested, nil
	default:
		return "", commandError{
			message: "invalid contract format: expected yaml or json",
			code:    1,
			kind:    "invalid_arguments",
		}
	}
}

func marshalScaffoldTemplateContract(contract hwpx.TemplateContract, contractFormat string) ([]byte, error) {
	switch contractFormat {
	case "json":
		return json.MarshalIndent(contract, "", "  ")
	default:
		return yaml.Marshal(contract)
	}
}

func marshalScaffoldTemplatePayload(payload map[string]any, payloadFormat string) ([]byte, error) {
	switch payloadFormat {
	case "json":
		return json.MarshalIndent(payload, "", "  ")
	default:
		return yaml.Marshal(payload)
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
	templatePath := strings.TrimSpace(opts.values["template"])
	payloadPath := strings.TrimSpace(opts.values["payload"])
	if (mappingPath == "" && (templatePath == "" || payloadPath == "")) ||
		(mappingPath != "" && (templatePath != "" || payloadPath != "")) {
		return commandError{
			message: "fill-template requires either --mapping or both --template and --payload",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	replacements, resolution, resolvedMappingPath, resolvedTemplatePath, resolvedPayloadPath, err := buildFillTemplateReplacements(opts.input, mappingPath, templatePath, payloadPath)
	if err != nil {
		return err
	}
	if len(replacements) == 0 {
		return commandError{
			message: "fill-template input did not produce any replacements",
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
	failOnMiss, err := parseOptionalBoolArg(opts.values, "fail-on-miss")
	if err != nil {
		return err
	}
	shouldFailOnMiss := failOnMiss != nil && *failOnMiss
	includeRoundtripCheck, err := parseOptionalBoolArg(opts.values, "roundtrip-check")
	if err != nil {
		return err
	}
	shouldIncludeRoundtripCheck := includeRoundtripCheck != nil && *includeRoundtripCheck

	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}
	if selector.Section == nil && !selector.AllSections {
		selector.AllSections = true
	}

	changes, misses, err := hwpx.PlanFillTemplate(opts.input, selector, replacements)
	if err != nil {
		return err
	}
	hwpx.CorrelateFillTemplateResolution(resolution, changes, misses)

	result := fillTemplateResult{
		InputPath:    absolutePath(opts.input),
		MappingPath:  resolvedMappingPath,
		TemplatePath: resolvedTemplatePath,
		PayloadPath:  resolvedPayloadPath,
		Resolution:   resolution,
		DryRun:       shouldDryRun,
		FailOnMiss:   shouldFailOnMiss,
		Applied:      false,
		Count:        len(changes),
		Changes:      changes,
		MissCount:    len(misses),
		Misses:       misses,
	}
	if shouldFailOnMiss && len(misses) > 0 {
		return commandError{
			message: "fill-template detected unmatched or partial replacements",
			code:    1,
			kind:    "fill_template_miss",
			data:    result,
		}
	}

	if !shouldDryRun {
		var report hwpx.Report
		var applied []hwpx.FillTemplateChange
		var appliedMisses []hwpx.FillTemplateMiss
		appliedReport, appliedChanges, appliedMisses, err := hwpx.FillTemplate(opts.input, selector, replacements)
		if err != nil {
			return err
		}
		report = appliedReport
		applied = appliedChanges
		result.DryRun = false
		result.Applied = true
		result.Count = len(applied)
		result.Changes = applied
		result.MissCount = len(appliedMisses)
		result.Misses = appliedMisses
		hwpx.CorrelateFillTemplateResolution(result.Resolution, applied, appliedMisses)
		result.Report = &report
		if err := maybeRecordChange(opts, "fill-template", "Fill template values", result.Report); err != nil {
			return err
		}
		if shouldIncludeRoundtripCheck {
			check, err := hwpx.RoundtripCheck(opts.input)
			if err != nil {
				return err
			}
			result.Check = &check
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
		if _, err := fmt.Fprintf(stdout, "input: %s\nmapping: %s\ntemplate: %s\npayload: %s\nresolution: %s (%d)\ndry-run: %t\nfail-on-miss: %t\nchanges: %d\nmisses: %d\n", result.InputPath, emptyDash(result.MappingPath), emptyDash(result.TemplatePath), emptyDash(result.PayloadPath), formatFillTemplateResolutionKind(result.Resolution), countFillTemplateResolutionEntries(result.Resolution), result.DryRun, result.FailOnMiss, result.Count, result.MissCount); err != nil {
			return err
		}
		if result.Check != nil {
			if _, err := fmt.Fprintf(stdout, "roundtrip-check: %t\n", result.Check.Passed); err != nil {
				return err
			}
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
		for _, miss := range result.Misses {
			if _, err := fmt.Fprintf(
				stdout,
				"miss kind=%s mode=%s selector=%q table-label=%q reason=%s matched=%d requested=%d partial=%t\n",
				miss.Kind,
				miss.Mode,
				miss.Selector,
				miss.TableLabel,
				miss.Reason,
				miss.Matched,
				miss.Requested,
				miss.Partial,
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

func runPreviewDiff(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
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
			message: "preview-diff requires an unpacked directory input",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	mappingPath := strings.TrimSpace(opts.values["mapping"])
	templatePath := strings.TrimSpace(opts.values["template"])
	payloadPath := strings.TrimSpace(opts.values["payload"])
	if (mappingPath == "" && (templatePath == "" || payloadPath == "")) ||
		(mappingPath != "" && (templatePath != "" || payloadPath != "")) {
		return commandError{
			message: "preview-diff requires either --mapping or both --template and --payload",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	replacements, resolution, resolvedMappingPath, resolvedTemplatePath, resolvedPayloadPath, err := buildFillTemplateReplacements(opts.input, mappingPath, templatePath, payloadPath)
	if err != nil {
		return err
	}
	if len(replacements) == 0 {
		return commandError{
			message: "preview-diff input did not produce any replacements",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	selector, err := parseSectionSelector(opts.values, true)
	if err != nil {
		return err
	}
	if selector.Section == nil && !selector.AllSections {
		selector.AllSections = true
	}

	changes, misses, err := hwpx.PlanFillTemplate(opts.input, selector, replacements)
	if err != nil {
		return err
	}
	hwpx.CorrelateFillTemplateResolution(resolution, changes, misses)

	result := previewDiffResult{
		InputPath:    absolutePath(opts.input),
		MappingPath:  resolvedMappingPath,
		TemplatePath: resolvedTemplatePath,
		PayloadPath:  resolvedPayloadPath,
		Resolution:   resolution,
		Count:        len(changes),
		Changes:      changes,
		MissCount:    len(misses),
		Misses:       misses,
		Summary:      buildPreviewDiffSummary(changes, misses),
	}

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "preview-diff",
			Success:       true,
			Data:          result,
		})
	default:
		if _, err := fmt.Fprintf(stdout, "input: %s\nmapping: %s\ntemplate: %s\npayload: %s\nresolution: %s (%d)\nchanges: %d\nmisses: %d\n", result.InputPath, emptyDash(result.MappingPath), emptyDash(result.TemplatePath), emptyDash(result.PayloadPath), formatFillTemplateResolutionKind(result.Resolution), countFillTemplateResolutionEntries(result.Resolution), result.Count, result.MissCount); err != nil {
			return err
		}
		for _, section := range result.Summary.Sections {
			if _, err := fmt.Fprintf(stdout, "section=%d path=%s changes=%d paragraph-changes=%d table-changes=%d\n", section.SectionIndex, section.SectionPath, section.ChangeCount, section.ParagraphChangeCount, section.TableChangeCount); err != nil {
				return err
			}
		}
		for _, change := range result.Changes {
			if _, err := fmt.Fprintf(stdout, "kind=%s mode=%s section=%d paragraph=%s table=%s cell=%s selector=%q previous=%q text=%q\n", change.Kind, change.Mode, change.SectionIndex, formatOptionalInt(change.ParagraphIndex), formatOptionalInt(change.TableIndex), formatCellCoordinate(change.Cell), change.Selector, change.PreviousText, change.Text); err != nil {
				return err
			}
		}
		for _, miss := range result.Misses {
			if _, err := fmt.Fprintf(stdout, "miss kind=%s mode=%s selector=%q table-label=%q reason=%s matched=%d requested=%d partial=%t\n", miss.Kind, miss.Mode, miss.Selector, miss.TableLabel, miss.Reason, miss.Matched, miss.Requested, miss.Partial); err != nil {
				return err
			}
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

func runSafePack(cmd *cobra.Command, args []string, stdout io.Writer, defaultFormat outputFormat) error {
	opts, err := parseNamedCommandOptions(cmd, args, defaultFormat, true)
	if err != nil {
		return err
	}
	if opts.output == "" {
		return commandError{
			message: "safe-pack requires --output",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	info, err := os.Stat(opts.input)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return commandError{
			message: "safe-pack requires an unpacked directory input",
			code:    1,
			kind:    "invalid_arguments",
		}
	}

	force, err := parseOptionalBoolArg(opts.values, "force")
	if err != nil {
		return err
	}
	forced := force != nil && *force

	report, err := hwpx.Validate(opts.input)
	if err != nil {
		return err
	}
	check, err := hwpx.RoundtripCheck(opts.input)
	if err != nil {
		return err
	}

	blockedBy := collectSafePackBlocks(report, check)
	result := safePackResult{
		InputPath:  absolutePath(opts.input),
		OutputPath: absolutePath(opts.output),
		Forced:     forced,
		Packed:     false,
		Report:     &report,
		Check:      &check,
		BlockedBy:  blockedBy,
	}

	if len(blockedBy) > 0 && !forced {
		if opts.format == formatJSON {
			return commandError{
				message: "safe-pack blocked by validation or roundtrip issues",
				code:    1,
				kind:    "safety_blocked",
				data:    result,
			}
		}
		return commandError{
			message: fmt.Sprintf("safe-pack blocked: %s", strings.Join(blockedBy, ", ")),
			code:    1,
			kind:    "safety_blocked",
		}
	}

	if err := hwpx.Pack(opts.input, opts.output); err != nil {
		return err
	}
	finalReport, err := hwpx.Validate(opts.output)
	if err != nil {
		return err
	}
	result.Packed = true
	result.Report = &finalReport

	switch opts.format {
	case formatJSON:
		return writeEnvelope(stdout, responseEnvelope{
			SchemaVersion: schemaVersion,
			Command:       "safe-pack",
			Success:       true,
			Data:          result,
		})
	default:
		if _, err := fmt.Fprintf(stdout, "input: %s\noutput: %s\nforced: %t\npacked: %t\nblocked-by: %d\n", result.InputPath, result.OutputPath, result.Forced, result.Packed, len(result.BlockedBy)); err != nil {
			return err
		}
		for _, reason := range result.BlockedBy {
			if _, err := fmt.Fprintf(stdout, "block: %s\n", reason); err != nil {
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

func collectSafePackBlocks(report hwpx.Report, check hwpx.RoundtripCheckReport) []string {
	var blocked []string
	if !report.Valid {
		blocked = append(blocked, "invalid")
	}
	if !report.RenderSafe {
		blocked = append(blocked, "render-safe=false")
	}
	if !check.Passed {
		blocked = append(blocked, "roundtrip-check-failed")
	}
	return blocked
}

func buildFillTemplateReplacements(inputPath, mappingPath, templatePath, payloadPath string) ([]hwpx.FillTemplateReplacement, *hwpx.FillTemplateResolutionReport, string, string, string, error) {
	resolved, err := hwpx.ResolveFillTemplateInput(inputPath, mappingPath, templatePath, payloadPath)
	if err != nil {
		var inputErr *hwpx.FillTemplateInputError
		if errors.As(err, &inputErr) {
			return nil, nil, "", "", "", commandError{
				message: inputErr.Error(),
				code:    1,
				kind:    inputErr.Kind,
			}
		}
		return nil, nil, "", "", "", err
	}
	return resolved.Replacements, &resolved.Resolution, resolved.MappingPath, resolved.TemplatePath, resolved.PayloadPath, nil
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func formatFillTemplateResolutionKind(resolution *hwpx.FillTemplateResolutionReport) string {
	if resolution == nil || strings.TrimSpace(resolution.InputKind) == "" {
		return "-"
	}
	return resolution.InputKind
}

func countFillTemplateResolutionEntries(resolution *hwpx.FillTemplateResolutionReport) int {
	if resolution == nil {
		return 0
	}
	return resolution.EntryCount
}

func buildPreviewDiffSummary(changes []hwpx.FillTemplateChange, misses []hwpx.FillTemplateMiss) previewDiffSummary {
	sections := map[int]*previewDiffSectionSummary{}
	kinds := map[string]*previewDiffKindSummary{}
	tables := map[string]*previewDiffTableSummary{}
	missReasons := map[string]*previewDiffMissReasonSummary{}

	for _, change := range changes {
		section, ok := sections[change.SectionIndex]
		if !ok {
			section = &previewDiffSectionSummary{
				SectionIndex: change.SectionIndex,
				SectionPath:  change.SectionPath,
			}
			sections[change.SectionIndex] = section
		}
		section.ChangeCount++
		switch change.Kind {
		case "paragraph":
			section.ParagraphChangeCount++
		case "table-cell":
			section.TableChangeCount++
		}

		kind, ok := kinds[change.Kind]
		if !ok {
			kind = &previewDiffKindSummary{Kind: change.Kind}
			kinds[change.Kind] = kind
		}
		kind.ChangeCount++

		if change.TableIndex != nil {
			key := fmt.Sprintf("%d:%d:%s", change.SectionIndex, *change.TableIndex, change.TableLabel)
			table, ok := tables[key]
			if !ok {
				table = &previewDiffTableSummary{
					SectionIndex: change.SectionIndex,
					TableIndex:   *change.TableIndex,
					TableLabel:   change.TableLabel,
				}
				tables[key] = table
			}
			table.ChangeCount++
		}
	}

	for _, miss := range misses {
		reason, ok := missReasons[miss.Reason]
		if !ok {
			reason = &previewDiffMissReasonSummary{Reason: miss.Reason}
			missReasons[miss.Reason] = reason
		}
		reason.Count++
	}

	sectionIndexes := make([]int, 0, len(sections))
	for index := range sections {
		sectionIndexes = append(sectionIndexes, index)
	}
	sort.Ints(sectionIndexes)

	kindNames := make([]string, 0, len(kinds))
	for kind := range kinds {
		kindNames = append(kindNames, kind)
	}
	sort.Strings(kindNames)

	tableKeys := make([]string, 0, len(tables))
	for key := range tables {
		tableKeys = append(tableKeys, key)
	}
	sort.Strings(tableKeys)

	missKeys := make([]string, 0, len(missReasons))
	for key := range missReasons {
		missKeys = append(missKeys, key)
	}
	sort.Strings(missKeys)

	result := previewDiffSummary{
		Sections:    make([]previewDiffSectionSummary, 0, len(sectionIndexes)),
		Kinds:       make([]previewDiffKindSummary, 0, len(kindNames)),
		Tables:      make([]previewDiffTableSummary, 0, len(tableKeys)),
		MissReasons: make([]previewDiffMissReasonSummary, 0, len(missKeys)),
	}
	for _, index := range sectionIndexes {
		result.Sections = append(result.Sections, *sections[index])
	}
	for _, kind := range kindNames {
		result.Kinds = append(result.Kinds, *kinds[kind])
	}
	for _, key := range tableKeys {
		result.Tables = append(result.Tables, *tables[key])
	}
	for _, key := range missKeys {
		result.MissReasons = append(result.MissReasons, *missReasons[key])
	}
	return result
}

func formatFindTargetsSectionSummary(context *hwpx.TemplateTargetContext) string {
	if context == nil || context.Section == nil {
		return ""
	}
	return fmt.Sprintf(
		"paragraphs=%d tables=%d merged=%d header=%t footer=%t page-number=%t preview=%s",
		context.Section.ParagraphCount,
		context.Section.TableCount,
		context.Section.MergedCellCount,
		context.Section.HasHeader,
		context.Section.HasFooter,
		context.Section.HasPageNumber,
		context.Section.TextPreview,
	)
}

func formatFindTargetsTableSummary(context *hwpx.TemplateTargetContext) string {
	if context == nil || context.Table == nil {
		return ""
	}
	return fmt.Sprintf(
		"rows=%d cols=%d merged=%d nested=%d paragraphs=%d label=%s preview=%s",
		context.Table.Rows,
		context.Table.Cols,
		context.Table.MergedCellCount,
		context.Table.NestedDepth,
		context.Table.ParagraphCount,
		context.Table.LabelText,
		context.Table.TextPreview,
	)
}

func formatFindTargetsParagraphSummary(context *hwpx.TemplateTargetContext) string {
	if context == nil || context.Paragraph == nil {
		return ""
	}
	return fmt.Sprintf("style=%s preview=%s", context.Paragraph.StyleSummary, context.Paragraph.TextPreview)
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
