package wmts

import (
	"fmt"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/tools"
	"math"
	"net/http"
)

// Resolution defines the properties for a WMTS grid zoom level.
type Resolution struct {
	ScaleDenominator float64 // Scale denominator for the zoom level
	CellSize         float64 // Cell size in meters
	MatrixWidth      float64 // Number of tiles in the width of the matrix
	MatrixHeight     float64 // Number of tiles in the height of the matrix
}

// Grid represents the WMTS Swiss Grid system with 28 zoom levels (0-27).
type Grid struct {
	Bbox            BBox // Bounding box of the grid in LV95 (EPSG:2056)
	SpatialREF      int  // SwissGrid LV95 (EPSG:2056)
	TileURLTemplate string
	UNIT            string
	MetersPerUnit   int
	TileSize        float64 // Tile size in pixels
	topLeftX        float64 // top-left corner X in LV95 (EPSG:2056)
	topLeftY        float64 // top-left corner Y in LV95 (EPSG:2056)
	tileSize        float64 // Tile size in pixels
	WmsBackendUrl   string
	WmsStartParams  string

	// resolutions is a map of zoom levels to their properties.
	resolutions map[int]Resolution
}

// GetTile calculates the tile indices (col, row) for a given coordinate and zoom level.
func (g *Grid) GetTile(coordX, coordY float64, zoomLevel int) (int, int, error) {
	if _, ok := g.resolutions[zoomLevel]; !ok {
		return 0, 0, fmt.Errorf("unsupported zoom level: %d. Please choose between 0 and %d", zoomLevel, g.MaxZoom())
	}

	zoomInfo := g.resolutions[zoomLevel]
	resolution := zoomInfo.CellSize

	tileCol := int((coordX - g.topLeftX) / (g.tileSize * resolution))
	tileRow := int((g.topLeftY - coordY) / (g.tileSize * resolution))

	return tileCol, tileRow, nil
}

// MaxZoom returns the maximum supported zoom level.
func (g *Grid) MaxZoom() int {
	maxZoom := 0
	for zoom := range g.resolutions {
		if zoom > maxZoom {
			maxZoom = zoom
		}
	}
	return maxZoom
}

// NumZoomLevels returns the number of supported zoom levels.
func (g *Grid) NumZoomLevels() int {
	return len(g.resolutions)
}

// MinZoom returns the minimum supported zoom level.
func (g *Grid) MinZoom() int {
	minZoom := 0
	first := true
	for zoom := range g.resolutions {
		if first {
			minZoom = zoom
			first = false
		}
		if zoom < minZoom {
			minZoom = zoom
		}
	}
	return minZoom
}

// IsValidTile checks if the given tile indices are valid for the specified zoom level.
func (g *Grid) IsValidTile(zoomLevel, tileCol, tileRow int) bool {
	if _, ok := g.resolutions[zoomLevel]; !ok {
		return false
	}
	if tileCol < 0 || tileCol > g.GetMaxNumCols(zoomLevel) {
		return false
	}
	if tileRow < 0 || tileRow > g.GetMaxNumRows(zoomLevel) {
		return false
	}
	return true
}

// GetTileBBox calculates the bounding box for a given tile.
func (g *Grid) GetTileBBox(zoomLevel, tileCol, tileRow int) (*BBox, error) {
	if !g.IsValidTile(zoomLevel, tileCol, tileRow) {

		if _, ok := g.resolutions[zoomLevel]; !ok {
			return nil, fmt.Errorf("unsupported zoom level. Please choose between 0 and %d", g.MaxZoom())
		}
		if tileCol < 0 || tileCol > g.GetMaxNumCols(zoomLevel) {
			return nil, fmt.Errorf("invalid column index. Please choose between 0 and %d", g.GetMaxNumCols(zoomLevel))
		}
		if tileRow < 0 || tileRow > g.GetMaxNumRows(zoomLevel) {
			return nil, fmt.Errorf("invalid row index. Please choose between 0 and %d", g.GetMaxNumRows(zoomLevel))
		}

		return nil, fmt.Errorf("invalid tile indices") // Should not happen based on previous checks, but good practice.
	}

	zoomInfo := g.resolutions[zoomLevel]
	resolution := zoomInfo.CellSize
	xMin := g.topLeftX + float64(tileCol)*g.tileSize*resolution
	yMax := g.topLeftY - float64(tileRow)*g.tileSize*resolution
	xMax := xMin + g.tileSize*resolution
	yMin := yMax - g.tileSize*resolution
	bb, err := NewBBox(xMin, yMin, xMax, yMax)
	if err != nil {
		return nil, err
	}
	return bb, nil
}

// GetBBox returns the bounding box of the entire grid.
func (g *Grid) GetBBox() BBox {
	return g.Bbox
}

// GetTileWidth returns the width of a tile in meters.
func (g *Grid) GetTileWidth() float64 {
	return g.tileSize * float64(g.MetersPerUnit)
}

// GetTileHeight returns the height of a tile in meters.
func (g *Grid) GetTileHeight() float64 {
	return g.tileSize * float64(g.MetersPerUnit)
}

// GetHeight returns the total height of the grid in meters.
func (g *Grid) GetHeight() float64 {
	return g.Bbox.YMax - g.Bbox.YMin
}

// GetWidth returns the total width of the grid in meters.
func (g *Grid) GetWidth() float64 {
	return g.Bbox.XMax - g.Bbox.XMin
}

// GetMaxNumRows returns the maximum number of rows for a given zoom level.
func (g *Grid) GetMaxNumRows(zoomLevel int) int {
	if _, ok := g.resolutions[zoomLevel]; !ok {
		panic(fmt.Sprintf("Unsupported zoom level. Please choose between 0 and %d.", g.MaxZoom()))
	}
	zoomInfo := g.resolutions[zoomLevel]
	if zoomInfo.MatrixHeight != 0 {
		return int(zoomInfo.MatrixHeight)
	}
	cellSize := zoomInfo.CellSize
	if cellSize == 0 {
		panic(fmt.Sprintf("cellSize was not found for zoom_level %d", zoomLevel))
	}
	return int(math.Round(g.GetHeight() / (g.tileSize * cellSize)))
}

// GetMaxNumCols returns the maximum number of columns for a given zoom level.
func (g *Grid) GetMaxNumCols(zoomLevel int) int {
	if _, ok := g.resolutions[zoomLevel]; !ok {
		panic(fmt.Sprintf("Unsupported zoom level. Please choose between 0 and %d.", g.MaxZoom()))
	}
	zoomInfo := g.resolutions[zoomLevel]
	if zoomInfo.MatrixWidth != 0 {
		return int(zoomInfo.MatrixWidth)
	}
	cellSize := zoomInfo.CellSize
	if cellSize == 0 {
		panic(fmt.Sprintf("cellSize was not found for zoom_level %d", zoomLevel))
	}
	return int(math.Round(g.GetWidth() / (g.tileSize * cellSize)))
}

// SaveTileImage get the wms request for a given tile and save the png file in the local cache path
func (g *Grid) SaveTileImage(zoomLevel, tileCol, tileRow int, lc LayerConfig, client *http.Client) (string, error) {
	bbox, err := g.GetTileBBox(zoomLevel, tileCol, tileRow)
	if err != nil {
		errMsg := fmt.Sprintf("error in GetTileBBox  zoom:%d, col:%d, row:%d", zoomLevel, tileCol, tileRow)
		return errMsg, err
	}
	layers := lc.WMSLayers
	params := g.GetWMSParams(*bbox, layers, int(g.GetTileWidth()), int(g.GetTileHeight()), "png") // Use GetTileWidth
	wmsURL := fmt.Sprintf("%s?%s%s", g.WmsBackendUrl, g.WmsStartParams, tools.BuildQueryString(params))
	imgPath := GetWmtsImgPath(lc.WMTSURLPrefix, lc.Name, lc.WMTSURLStyle, lc.WMTSDimensionYear, lc.WMTSMatrixSet, "png", zoomLevel, tileRow, tileCol)
	err = tools.GetPngFromUrl(client, wmsURL, imgPath, 2)
	if err != nil {
		errMsg := fmt.Sprintf("error in GetPngFromUrl tile  zoom:%d, col:%d, row:%d", zoomLevel, tileCol, tileRow)
		return errMsg, err
	}
	return imgPath, nil
}

// GetTileWmsUrl returns the WMS URL for a given tile.
func (g *Grid) GetTileWmsUrl(zoomLevel, tileCol, tileRow int, layers string) (string, error) {
	bbox, err := g.GetTileBBox(zoomLevel, tileCol, tileRow)
	if err != nil {
		return "", err
	}
	params := g.GetWMSParams(*bbox, layers, int(g.GetTileWidth()), int(g.GetTileHeight()), "png")
	wmsURL := fmt.Sprintf("%s?%s%s", g.WmsBackendUrl, g.WmsStartParams, tools.BuildQueryString(params))
	return wmsURL, nil
}
