package object

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func AddEquation(targetDir string, spec shared.EquationSpec) (shared.Report, string, error) {
	return shared.AddEquation(targetDir, spec)
}

func AddRectangle(targetDir string, spec shared.RectangleSpec) (shared.Report, string, int, int, error) {
	return shared.AddRectangle(targetDir, spec)
}
