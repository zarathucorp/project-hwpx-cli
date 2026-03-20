package hwpx

import (
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx/core"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"
)

type ManifestItem = core.ManifestItem
type AnalysisCell = core.AnalysisCell
type TemplateCell = core.TemplateCell
type TemplateParagraph = core.TemplateParagraph
type TemplateSection = core.TemplateSection
type TemplateTable = core.TemplateTable
type TemplateTextCandidate = core.TemplateTextCandidate
type TemplateFingerprint = core.TemplateFingerprint
type TargetQuery = core.TargetQuery
type TemplateTargetSectionContext = core.TemplateTargetSectionContext
type TemplateTargetTableContext = core.TemplateTargetTableContext
type TemplateTargetParagraphContext = core.TemplateTargetParagraphContext
type TemplateTargetContext = core.TemplateTargetContext
type TemplateTargetMatch = core.TemplateTargetMatch
type TemplateContract = core.TemplateContract
type TemplateContractField = core.TemplateContractField
type TemplateContractTable = core.TemplateContractTable
type TemplateContractColumn = core.TemplateContractColumn
type TemplateContractSelector = core.TemplateContractSelector
type TemplateAnalysis = core.TemplateAnalysis
type RoundtripSnapshot = core.RoundtripSnapshot
type RoundtripIssue = core.RoundtripIssue
type RoundtripCheckReport = core.RoundtripCheckReport
type Summary = core.Summary
type Report = core.Report

type TableSpec = shared.TableSpec
type TableCellStyleSpec = shared.TableCellStyleSpec
type ImageEmbed = shared.ImageEmbed
type ImagePlacement = shared.ImagePlacement
type ObjectPositionSpec = shared.ObjectPositionSpec
type HeaderFooterSpec = shared.HeaderFooterSpec
type PageNumberSpec = shared.PageNumberSpec
type ColumnSpec = shared.ColumnSpec
type PageLayoutSpec = shared.PageLayoutSpec
type ParagraphLayoutSpec = shared.ParagraphLayoutSpec
type ParagraphListSpec = shared.ParagraphListSpec
type TextStyleSpec = shared.TextStyleSpec
type TableCellTextSpec = shared.TableCellTextSpec
type SectionSelector = shared.SectionSelector
type FillTemplateReplacement = shared.FillTemplateReplacement
type FillTemplateChange = shared.FillTemplateChange
type FillTemplateMiss = shared.FillTemplateMiss
type TableCellCoordinate = shared.TableCellCoordinate
type RunStyleFilter = shared.RunStyleFilter
type RunStyleMatch = shared.RunStyleMatch
type RunTextReplacement = shared.RunTextReplacement
type ObjectFilter = shared.ObjectFilter
type ObjectMatch = shared.ObjectMatch
type TagFilter = shared.TagFilter
type TagMatch = shared.TagMatch
type AttributeFilter = shared.AttributeFilter
type AttributeMatch = shared.AttributeMatch
type XPathFilter = shared.XPathFilter
type XPathMatch = shared.XPathMatch
type HistoryEntrySpec = shared.HistoryEntrySpec
type NoteSpec = shared.NoteSpec
type MemoSpec = shared.MemoSpec
type BookmarkSpec = shared.BookmarkSpec
type HyperlinkSpec = shared.HyperlinkSpec
type HeadingSpec = shared.HeadingSpec
type TOCSpec = shared.TOCSpec
type CrossReferenceSpec = shared.CrossReferenceSpec
type EquationSpec = shared.EquationSpec
type RectangleSpec = shared.RectangleSpec
type LineSpec = shared.LineSpec
type EllipseSpec = shared.EllipseSpec
type TextBoxSpec = shared.TextBoxSpec
