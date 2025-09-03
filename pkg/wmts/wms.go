package wmts

import (
	"fmt"
	"strconv"
)

// GetWMSParams generates a map of WMS parameters based on the provided inputs.
func (g *Grid) GetWMSParams(bbox BBox, layers string, width, height int, imageFormat string) map[string]string {
	if width <= 0 {
		width = 256
	}
	if height <= 0 {
		height = 256
	}
	if imageFormat == "" {
		imageFormat = "png"
	}

	bboxStr := bbox.String()

	params := map[string]string{
		"SERVICE":     "WMS",
		"VERSION":     "1.3.0",
		"REQUEST":     "GetMap",
		"FORMAT":      fmt.Sprintf("image/%s", imageFormat),
		"TRANSPARENT": strconv.FormatBool(imageFormat == "png"), // "true" if png, "false" otherwise
		"LAYERS":      layers,
		"WIDTH":       fmt.Sprintf("%d", width),
		"HEIGHT":      fmt.Sprintf("%d", height),
		"CRS":         "EPSG:2056",
		"STYLES":      "",
		"BBOX":        bboxStr,
		"BUFFER":      "500",
	}

	return params
}

// Helper function to convert a BBox to a string (if not already defined as a method)
