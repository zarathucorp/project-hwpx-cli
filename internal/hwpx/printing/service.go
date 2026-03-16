package printing

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func PrintToPDF(inputPath, outputPath, workspaceDir string) error {
	return shared.PrintToPDF(inputPath, outputPath, workspaceDir)
}
