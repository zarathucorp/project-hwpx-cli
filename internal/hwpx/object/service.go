package object

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/shared"

func AddEquation(targetDir string, spec shared.EquationSpec) (shared.Report, string, error) {
	return shared.AddEquation(targetDir, spec)
}

func AddRectangle(targetDir string, spec shared.RectangleSpec) (shared.Report, string, int, int, error) {
	return shared.AddRectangle(targetDir, spec)
}

func AddLine(targetDir string, spec shared.LineSpec) (shared.Report, string, int, int, error) {
	return shared.AddLine(targetDir, spec)
}

func AddEllipse(targetDir string, spec shared.EllipseSpec) (shared.Report, string, int, int, error) {
	return shared.AddEllipse(targetDir, spec)
}

func AddTextBox(targetDir string, spec shared.TextBoxSpec) (shared.Report, string, int, int, error) {
	return shared.AddTextBox(targetDir, spec)
}

func FindObjects(targetDir string, filter shared.ObjectFilter) ([]shared.ObjectMatch, error) {
	return shared.FindObjects(targetDir, filter)
}
