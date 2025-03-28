package wmts

import "fmt"

// GetWmtsImgPath gives a standard wmts path string based on given parameters
func GetWmtsImgPath(prefix, layer, style, year, matrixSet, fileExtension string, zoom, row, col int) string {
	//https://tilesmn95.lausanne.ch/{prefix}/{layer}/{style}/{year}/{matrixSet}/{zoom}/{row}/{col}.png
	return fmt.Sprintf("%s/%s/%s/%s/%s/%d/%d/%d.%s", prefix, layer, style, year, matrixSet, zoom, row, col, fileExtension)
}
