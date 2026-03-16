package hwpx

import "github.com/zarathu/project-hwpx-cli/internal/hwpx/core"

func Inspect(filePath string) (Report, error) {
	return core.Inspect(filePath)
}

func Validate(targetPath string) (Report, error) {
	return core.Validate(targetPath)
}

func ExtractText(filePath string) (string, error) {
	return core.ExtractText(filePath)
}

func Unpack(filePath, outputDir string) error {
	return core.Unpack(filePath, outputDir)
}

func Pack(inputDir, outputFile string) error {
	return core.Pack(inputDir, outputFile)
}
