package main

import (
	"flag"
	"fmt"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/tools"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/version"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/wmts"
	"log"
)

//saveWmtsTiles allows to save all png tiles for a given zoom level and layer

const (
	APP                        = "saveWmtsTiles"
	defaultWmtsFilePath        = "/var/sig/tiles/1.0.0"
	defaultWmtsConfig          = "wmtsConfig.yaml"
	defaultLayer               = "fonds_geo_osm_bdcad_couleur"
	defaultZoomLevel           = 2
	defaultMaxClientTimeOutSec = 10
	defaultMaxIdleConn         = 100
	defaultMaxIdleConnPerHost  = 100
	defaultIdleConnTimeoutSec  = 90
)

func main() {
	l, err := golog.NewLogger("zap", golog.DebugLevel, fmt.Sprintf("%s:", APP))
	if err != nil {
		log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", APP, version.VERSION, version.Build, version.REPOSITORY)
	// get the yaml config file name received from config parameter
	configFileName := flag.String("config", defaultWmtsConfig, "config file name")
	layerName := flag.String("layer", defaultLayer, "config file name")
	zoomLevel := flag.Int("zoom", defaultZoomLevel, "zoom level")
	flag.Parse()
	l.Info("ℹ️ Using zoom level : %d", zoomLevel)
	l.Info("ℹ️ Using layer : %s", *layerName)
	l.Info("ℹ️ Reading config file: %s", *configFileName)
	layers, err := wmts.LoadLayerConfigFromYAML(*configFileName)
	if err != nil {
		l.Fatal("error loading %s layer config: %v", *configFileName, err)
	}
	// Check if there are layers loaded
	if len(layers) == 0 {
		l.Fatal("💥💥 no layers loaded from %s", configFileName)
	}
	l.Info("ℹ️ Found %d layers in config file: %s", len(layers), *configFileName)
	isLayerNameInConfig := false
	for name, layer := range layers {
		fmt.Printf("Layer: %s\n", name)
		if name == *layerName {
			isLayerNameInConfig = true
		}
		wmts.PrintLayerInfo(layer)
	}
	if !isLayerNameInConfig {
		l.Fatal("💥💥 layer %s not found in %s", *layerName, *configFileName)
	}
	l.Info("ℹ️ Using layer: %s", *layerName)
	layerConfig := layers[*layerName]
	wmsBackEndUrl := layerConfig.WMSBackendURL
	wmsStartParams := layerConfig.WMSBackendPrefix
	wmtsBBox := layerConfig.WMTSBBox
	xMin, yMin, xMax, yMax := wmtsBBox[0], wmtsBBox[1], wmtsBBox[2], wmtsBBox[3]

	// Create a new grid
	myGrid := wmts.CreateNewLausanneGridFromEnvOrFail(wmsBackEndUrl, wmsStartParams)
	numCols := myGrid.GetMaxNumCols(*zoomLevel)
	numRows := myGrid.GetMaxNumRows(*zoomLevel)
	l.Info("ℹ️ will generate % rows and %d cols for zoom level %d", numRows, numCols, *zoomLevel)
	// get the min col and row from the lausanne wmts grid package
	minCol, maxRow, err := myGrid.GetTile(xMin, yMin, *zoomLevel)
	if err != nil {
		l.Fatal("💥💥 GetTile(%f, %f, %d) got error: %v", xMin, yMin, *zoomLevel, err)
	}
	maxCol, minRow, err := myGrid.GetTile(xMax, yMax, *zoomLevel)
	if err != nil {
		l.Fatal("💥💥 GetTile(%f, %f, %d) got error: %v", xMax, yMax, *zoomLevel, err)
	}
	l.Info("ℹ️ minCol: %d, minRow: %d", minCol, minRow)
	l.Info("ℹ️ maxCol: %d, maxRow: %d", maxCol, maxRow)
	client := tools.CreateHTTPClient(defaultMaxClientTimeOutSec, defaultMaxIdleConn, defaultMaxIdleConnPerHost, defaultIdleConnTimeoutSec)
	for row := minRow; row < maxRow; row++ {
		for col := minCol; col < maxCol; col++ {
			imagePath, err := myGrid.SaveTileImage(*zoomLevel, col, row, layers[*layerName], client)
			if err != nil {
				l.Error("💥 SaveTileImage for %s got error: %v", imagePath, err)
			}
			l.Info("ℹ️ zoom: %d, tile (row:%3d, col:%3d) saved image to imagePath: %s", *zoomLevel, col, row, imagePath)
		}

	}

}
