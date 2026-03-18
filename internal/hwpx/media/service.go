package media

import "github.com/zarathucop/project-hwpx-cli/internal/hwpx/shared"

func EmbedImage(targetDir, imagePath string) (shared.Report, shared.ImageEmbed, error) {
	return shared.EmbedImage(targetDir, imagePath)
}

func InsertImage(targetDir, imagePath string, widthMM float64) (shared.Report, shared.ImagePlacement, error) {
	return shared.InsertImage(targetDir, imagePath, widthMM)
}
