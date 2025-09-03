package wmts

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"

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
func (g *Grid) SaveTileImage(zoomLevel, tileCol, tileRow int, lc LayerConfig, basePath string, client *http.Client) (string, error) {
	bbox, err := g.GetTileBBox(zoomLevel, tileCol, tileRow)
	if err != nil {
		errMsg := fmt.Sprintf("error in GetTileBBox  zoom:%d, col:%d, row:%d", zoomLevel, tileCol, tileRow)
		return errMsg, err
	}
	layers := lc.WMSLayers
	params := g.GetWMSParams(*bbox, layers, int(g.GetTileWidth()), int(g.GetTileHeight()), "png") // Use GetTileWidth
	wmsURL := fmt.Sprintf("%s?%s%s", g.WmsBackendUrl, g.WmsStartParams, tools.BuildQueryString(params))
	imgPath := GetWmtsImgPath(basePath, lc.WMTSURLPrefix, lc.Name, lc.WMTSURLStyle, lc.WMTSDimensionYear, lc.WMTSMatrixSet, "png", zoomLevel, tileRow, tileCol)
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

// SaveTilesFromMetaTile fetches a larger image (a "meta-tile") from the WMS server,
// splits it into individual tiles, and saves them to the local cache.
// This approach reduces the number of HTTP requests, improving performance.
func (g *Grid) SaveTilesFromMetaTile(zoomLevel, startCol, startRow, numCols, numRows int, lc LayerConfig, basePath string, client *http.Client) error {
	// 1. Calculate the bounding box for the entire meta-tile.
	// BBox of the top-left tile
	topLeftBBox, err := g.GetTileBBox(zoomLevel, startCol, startRow)
	if err != nil {
		return fmt.Errorf("failed to get bounding box for top-left tile: %w", err)
	}

	// BBox of the bottom-right tile
	bottomRightBBox, err := g.GetTileBBox(zoomLevel, startCol+numCols-1, startRow+numRows-1)
	if err != nil {
		return fmt.Errorf("failed to get bounding box for bottom-right tile: %w", err)
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
	params := g.GetWMSParams(*metaBBox, lc.WMSLayers, metaTileWidth, metaTileHeight, "png")
	wmsURL := fmt.Sprintf("%s?%s%s", g.WmsBackendUrl, g.WmsStartParams, tools.BuildQueryString(params))

	resp, err := client.Get(wmsURL)
	if err != nil {
		return fmt.Errorf("WMS request for meta-tile failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WMS request returned non-OK status: %d", resp.StatusCode)
	}

	// Decode the image from the response body.
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to decode meta-tile image: %w", err)
	}

	// 3. Split the meta-tile image into individual tiles.
	tileWidth := int(g.GetTileWidth())
	tileHeight := int(g.GetTileHeight())
	tiles, err := imgTools.SplitImage(img, tileWidth, tileHeight)
	if err != nil {
		return fmt.Errorf("failed to split meta-tile image: %w", err)
	}

	// 4. Save each individual tile.
	tileIndex := 0
	for row := 0; row < numRows; row++ {
		for col := 0; col < numCols; col++ {
			tileRow := startRow + row
			tileCol := startCol + col
			imgPath := GetWmtsImgPath(basePath, lc.WMTSURLPrefix, lc.Name, lc.WMTSURLStyle, lc.WMTSDimensionYear, lc.WMTSMatrixSet, "png", zoomLevel, tileRow, tileCol)

			// Create directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(imgPath), os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory for tile: %w", err)
			}

			outFile, err := os.Create(imgPath)
			if err != nil {
				return fmt.Errorf("failed to create tile image file: %w", err)
			}
			defer outFile.Close()

			if err := png.Encode(outFile, tiles[tileIndex]); err != nil {
				return fmt.Errorf("failed to encode tile image: %w", err)
			}
			tileIndex++
		}
	}

	return nil
}
