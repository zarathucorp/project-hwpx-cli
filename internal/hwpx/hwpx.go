package hwpx

import (
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx/core"
	"github.com/zarathucorp/project-hwpx-cli/internal/hwpx/shared"
)

func Inspect(filePath string) (Report, error) {
	return core.Inspect(filePath)
}

func Validate(targetPath string) (Report, error) {
	return core.Validate(targetPath)
}

func AnalyzeTemplate(targetPath string) (TemplateAnalysis, error) {
	return core.AnalyzeTemplate(targetPath)
}

func LoadTemplateContract(path string) (TemplateContract, error) {
	return core.LoadTemplateContract(path)
}

func ValidateTemplateContract(contract TemplateContract) error {
	return core.ValidateTemplateContract(contract)
}

func VerifyTemplateContractFingerprint(contract TemplateContract, analysis TemplateAnalysis) error {
	return core.VerifyTemplateContractFingerprint(contract, analysis)
}

func FindTargets(targetPath string, query TargetQuery) ([]TemplateTargetMatch, error) {
	return core.FindTargets(targetPath, query)
}

func RemoveGuides(targetDir string, selector SectionSelector, reason string) (Report, []TemplateTextCandidate, error) {
	return shared.RemoveGuides(targetDir, selector, reason)
}

func PlanFillTemplate(targetDir string, selector SectionSelector, replacements []FillTemplateReplacement) ([]FillTemplateChange, []FillTemplateMiss, error) {
	return shared.PlanFillTemplate(targetDir, selector, replacements)
}

func FillTemplate(targetDir string, selector SectionSelector, replacements []FillTemplateReplacement) (Report, []FillTemplateChange, []FillTemplateMiss, error) {
	return shared.FillTemplate(targetDir, selector, replacements)
}

func RoundtripCheck(targetPath string) (RoundtripCheckReport, error) {
	return core.RoundtripCheck(targetPath)
}

func ExtractText(filePath string) (string, error) {
	return core.ExtractText(filePath)
}

func ExportMarkdown(targetPath string) (string, int, error) {
	return core.ExportMarkdown(targetPath)
}

func ExportHTML(targetPath string) (string, int, error) {
	return core.ExportHTML(targetPath)
}

func Unpack(filePath, outputDir string) error {
	return core.Unpack(filePath, outputDir)
}

func Pack(inputDir, outputFile string) error {
	return core.Pack(inputDir, outputFile)
}
