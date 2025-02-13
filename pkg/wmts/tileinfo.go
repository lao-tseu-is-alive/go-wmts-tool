package wmts

import "fmt"

type TileInfo struct {
	ZoomLevel int
	Col       int
	Row       int
	BBox      BBox
}

// GetTileInfoByXY returns the tile info for the given x, y and zoom level.
func (g *Grid) GetTileInfoByXY(x, y, zoomLevel int) (*TileInfo, error) {
	if x < 0 || y < 0 || zoomLevel < 0 {
		return nil, fmt.Errorf("invalid x, y or zoom level")
	}

	if _, ok := g.resolutions[zoomLevel]; !ok {
		return nil, fmt.Errorf("zoom level %d not supported", zoomLevel)
	}

	// Get the resolution for the given zoom level.
	zoomInfo := g.resolutions[zoomLevel]
	resolution := zoomInfo["cellSize"]

	// Calculate the tile column and row.
	tileCol := int((float64(x) - g.topLeftX) / (g.tileSize * resolution))
	tileRow := int((g.topLeftY - float64(y)) / (g.tileSize * resolution))

	// Check if the tile indices are valid.
	if !g.IsValidTile(zoomLevel, tileCol, tileRow) {
		return nil, fmt.Errorf("invalid tile indices")
	}

	// Calculate the bounding box for the tile.
	tileBBox, err := g.GetTileBBox(zoomLevel, tileCol, tileRow)
	if err != nil {
		return nil, fmt.Errorf("error calculating tile bounding box")
	}

	// Create a new TileInfo object.
	tileInfo := &TileInfo{
		ZoomLevel: zoomLevel,
		Col:       tileCol,
		Row:       tileRow,
		BBox:      *tileBBox,
	}

	return tileInfo, nil
}
