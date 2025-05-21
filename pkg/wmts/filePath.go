package wmts

import "fmt"

// GetWmtsImgPath gives a standard wmts path string based on given parameters
func GetWmtsImgPath(basePath, prefix, layer, style, year, matrixSet, fileExtension string, zoom, row, col int) string {
	//{basePath}{prefix}/{layer}/{style}/{year}/{matrixSet}/{zoom}/{row}/{col}.png
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%d/%d/%d.%s", basePath, prefix, layer, style, year, matrixSet, zoom, row, col, fileExtension)
}
