package wmts

import (
	"fmt"
	"strconv"
)

// GetWMSParams generates a map of WMS parameters based on the provided inputs.
func (g *Grid) GetWMSParams(bbox BBox, layers string, width, height, buffer int, imageFormat string) map[string]string {
	if width <= 0 {
		width = DefaultTileSize
	}
	if height <= 0 {
		height = DefaultTileSize
	}
	if imageFormat == "" {
		imageFormat = DefaultImageFormat
	}
	// The BBOX needs to be expanded to account for the buffer.
	// We'll calculate the new BBox based on the resolution.
	resolution := (bbox.XMax - bbox.XMin) / float64(width)
	bufferUnits := float64(buffer) * resolution

	bufferedBbox := BBox{
		XMin: bbox.XMin - bufferUnits,
		YMin: bbox.YMin - bufferUnits,
		XMax: bbox.XMax + bufferUnits,
		YMax: bbox.YMax + bufferUnits,
	}

	params := map[string]string{
		"SERVICE":     "WMS",
		"VERSION":     "1.3.0",
		"REQUEST":     "GetMap",
		"FORMAT":      fmt.Sprintf("image/%s", imageFormat),
		"TRANSPARENT": strconv.FormatBool(imageFormat == DefaultImageFormat), // "true" if png, "false" otherwise
		"LAYERS":      layers,
		// The width and height must also be increased
		"WIDTH":  fmt.Sprintf("%d", width+(buffer*2)),
		"HEIGHT": fmt.Sprintf("%d", height+(buffer*2)),
		"CRS":    fmt.Sprintf("EPSG:%d", DefaultSpatialRef),
		"STYLES": "",
		"BBOX":   bufferedBbox.String(),
	}

	return params
}

// Helper function to convert a BBox to a string (if not already defined as a method)
