package wmts

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png" // registers the png decoder used by image.Decode
	"math"
	"net/http"
	"os"
	"time"

	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/imgTools"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/tools"
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
	WmsBackendUrl   string
	WmsStartParams  string

	// resolutions is a map of zoom levels to their properties.
	resolutions map[int]Resolution
	l           golog.MyLogger
}

// GetTile calculates the tile indices (col, row) for a given coordinate and zoom level.
func (g *Grid) GetTile(coordX, coordY float64, zoomLevel int) (int, int, error) {
	if _, ok := g.resolutions[zoomLevel]; !ok {
		return 0, 0, fmt.Errorf("unsupported zoom level: %d. Please choose between 0 and %d", zoomLevel, g.MaxZoom())
	}

	zoomInfo := g.resolutions[zoomLevel]
	resolution := zoomInfo.CellSize

	tileCol := int((coordX - g.topLeftX) / (g.TileSize * resolution))
	tileRow := int((g.topLeftY - coordY) / (g.TileSize * resolution))

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
		maxCols := g.GetMaxNumCols(zoomLevel)
		maxRows := g.GetMaxNumRows(zoomLevel)

		if _, ok := g.resolutions[zoomLevel]; !ok {
			return nil, fmt.Errorf("unsupported zoom level. Please choose between 0 and %d", g.MaxZoom())
		}
		if tileCol < 0 || tileCol > maxCols {
			return nil, fmt.Errorf("invalid column index. Please choose between 0 and %d", g.GetMaxNumCols(zoomLevel))
		}
		if tileRow < 0 || tileRow > maxRows {
			return nil, fmt.Errorf("invalid row index. Please choose between 0 and %d", g.GetMaxNumRows(zoomLevel))
		}

		return nil, fmt.Errorf("invalid tile indices: zoom=%d, col=%d (max=%d), row=%d (max=%d)",
			zoomLevel, tileCol, maxCols, tileRow, maxRows) // Should not happen based on previous checks, but good practice.
	}

	zoomInfo := g.resolutions[zoomLevel]
	resolution := zoomInfo.CellSize
	xMin := g.topLeftX + float64(tileCol)*g.TileSize*resolution
	yMax := g.topLeftY - float64(tileRow)*g.TileSize*resolution
	xMax := xMin + g.TileSize*resolution
	yMin := yMax - g.TileSize*resolution
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
	return g.TileSize * float64(g.MetersPerUnit)
}

// GetTileHeight returns the height of a tile in meters.
func (g *Grid) GetTileHeight() float64 {
	return g.TileSize * float64(g.MetersPerUnit)
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
	return int(math.Round(g.GetHeight() / (g.TileSize * cellSize)))
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
	return int(math.Round(g.GetWidth() / (g.TileSize * cellSize)))
}

// SaveTileImage get the wms request for a given tile and save the png file in the local cache path
func (g *Grid) SaveTileImage(zoomLevel, tileCol, tileRow, buffer int, lc LayerConfig, basePath string, client *http.Client) (string, error) {
	bbox, err := g.GetTileBBox(zoomLevel, tileCol, tileRow)
	if err != nil {
		errMsg := fmt.Sprintf("error in GetTileBBox  zoom:%d, col:%d, row:%d", zoomLevel, tileCol, tileRow)
		return errMsg, err
	}
	layers := lc.WMSLayers
	params := g.GetWMSParams(*bbox, layers, int(g.GetTileWidth()), int(g.GetTileHeight()), buffer, DefaultImageFormat) // Use GetTileWidth
	wmsURL := fmt.Sprintf("%s?%s%s", g.WmsBackendUrl, g.WmsStartParams, tools.BuildQueryString(params))
	imgPath := lc.TileImgPath(basePath, zoomLevel, tileRow, tileCol)
	err = tools.GetPngFromUrl(client, wmsURL, imgPath, buffer, tools.DefaultMaxRetries, g.l)
	if err != nil {
		errMsg := fmt.Sprintf("error in GetPngFromUrl tile  zoom:%d, col:%d, row:%d", zoomLevel, tileCol, tileRow)
		return errMsg, err
	}
	return imgPath, nil
}

// GetTileWmsUrl returns the WMS URL for a given tile.
func (g *Grid) GetTileWmsUrl(zoomLevel, tileCol, tileRow, buffer int, layers string) (string, error) {
	bbox, err := g.GetTileBBox(zoomLevel, tileCol, tileRow)
	if err != nil {
		return "", err
	}
	params := g.GetWMSParams(*bbox, layers, int(g.GetTileWidth()), int(g.GetTileHeight()), buffer, DefaultImageFormat)
	wmsURL := fmt.Sprintf("%s?%s%s", g.WmsBackendUrl, g.WmsStartParams, tools.BuildQueryString(params))
	return wmsURL, nil
}

// IsMetaTileFresh reports whether all the tiles of a meta-tile already exist in the local cache
// and were all written less than maxAge ago. Older tiles are considered outdated.
func (g *Grid) IsMetaTileFresh(zoomLevel, startCol, startRow, numCols, numRows int, lc LayerConfig, basePath string, maxAge time.Duration) bool {
	for row := startRow; row < startRow+numRows; row++ {
		for col := startCol; col < startCol+numCols; col++ {
			info, err := os.Stat(lc.TileImgPath(basePath, zoomLevel, row, col))
			if err != nil || time.Since(info.ModTime()) > maxAge {
				return false
			}
		}
	}
	return true
}

// MetaTileDesc returns a human-readable description of the tiles covered by a meta-tile,
// used in logs so operators can check them (e.g. "zoom:9 rows 15432-15435 cols 9192-9195").
func MetaTileDesc(zoomLevel, startCol, startRow, numCols, numRows int) string {
	return fmt.Sprintf("zoom:%d rows %d-%d cols %d-%d", zoomLevel, startRow, startRow+numRows-1, startCol, startCol+numCols-1)
}

// SaveTilesFromMetaTile fetches a larger image (a "meta-tile") from the WMS server,
// splits it into individual tiles, and saves them to the local cache.
// This approach reduces the number of HTTP requests, improving performance.
// Transient WMS failures are retried up to maxRetries times.
// It returns the number of png tiles actually written, also on error (some tiles may have been saved).
func (g *Grid) SaveTilesFromMetaTile(zoomLevel, startCol, startRow, numCols, numRows, buffer, maxRetries int, lc LayerConfig, basePath string, client *http.Client) (int, error) {
	// 1. Calculate the bounding box for the entire meta-tile.
	// BBox of the top-left tile
	topLeftBBox, err := g.GetTileBBox(zoomLevel, startCol, startRow)
	if err != nil {
		return 0, fmt.Errorf("failed to get bounding box for top-left tile: %w", err)
	}

	// BBox of the bottom-right tile
	bottomRightBBox, err := g.GetTileBBox(zoomLevel, startCol+numCols-1, startRow+numRows-1)
	if err != nil {
		return 0, fmt.Errorf("failed to get bounding box for bottom-right tile: %w", err)
	}

	// The meta-tile's bounding box is the combination of the top-left and bottom-right tile BBoxes.
	metaBBox := &BBox{
		XMin: topLeftBBox.XMin,
		YMin: bottomRightBBox.YMin,
		XMax: bottomRightBBox.XMax,
		YMax: topLeftBBox.YMax,
	}

	// 2. Make a single WMS request for the entire meta-tile.
	metaTileWidth := int(g.GetTileWidth()) * numCols
	metaTileHeight := int(g.GetTileHeight()) * numRows
	params := g.GetWMSParams(*metaBBox, lc.WMSLayers, metaTileWidth, metaTileHeight, buffer, DefaultImageFormat)
	wmsURL := fmt.Sprintf("%s?%s%s", g.WmsBackendUrl, g.WmsStartParams, tools.BuildQueryString(params))

	body, err := tools.FetchImageWithRetry(client, wmsURL, MetaTileDesc(zoomLevel, startCol, startRow, numCols, numRows), maxRetries, g.l)
	if err != nil {
		return 0, err
	}

	// Decode the image from the response body.
	bufferedImage, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to decode meta-tile image: %w", err)
	}

	// 3. Split the meta-tile image into individual tiles.
	tileWidth := int(g.GetTileWidth())
	tileHeight := int(g.GetTileHeight())
	img := imgTools.CropImage(bufferedImage, buffer, g.l)
	tiles, err := imgTools.SplitImage(img, tileWidth, tileHeight)
	if err != nil {
		return 0, fmt.Errorf("failed to split meta-tile image: %w", err)
	}

	// 4. Save each individual tile.
	written := 0
	for row := 0; row < numRows; row++ {
		for col := 0; col < numCols; col++ {
			imgPath := lc.TileImgPath(basePath, zoomLevel, startRow+row, startCol+col)
			if err := tools.SavePng(imgPath, tiles[written]); err != nil {
				return written, fmt.Errorf("failed to save tile image after writing %d/%d png: %w", written, numCols*numRows, err)
			}
			written++
		}
	}

	return written, nil
}
