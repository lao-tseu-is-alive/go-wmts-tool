package main

import (
	"flag"
	"fmt"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/version"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/wmts"
	"log"
)

//saveWmtsTiles allows to save all png tiles for a given zoom level and layer

const (
	APP                 = "saveWmtsTiles"
	defaultWmtsFilePath = "/var/sig/tiles/1.0.0"
	defaultWmtsConfig   = "wmtsConfig.yaml"
)

func main() {
	l, err := golog.NewLogger("zap", golog.DebugLevel, fmt.Sprintf("%s:", APP))
	if err != nil {
		log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", APP, version.VERSION, version.Build, version.REPOSITORY)
	// get the yaml config file name received from config parameter
	configFileName := flag.String("config", defaultWmtsConfig, "config file name")
	zoomLevel := flag.Int("zoom", 2, "zoom level")
	flag.Parse()
	l.Info("ℹ️ Using zoom level : %d", zoomLevel)
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
	for name, layer := range layers {
		fmt.Printf("Layer: %s\n", name)
		fmt.Printf("  Title: %s\n", layer.Title)
		fmt.Printf("  WMS Backend URL: %s\n", layer.WMSBackendURL)
		fmt.Printf("  WMS Layers: %s\n", layer.WMSLayers)
		fmt.Printf("  BBox: %v\n", layer.BBox)
		fmt.Printf("  WMTS URL Prefix: %s\n", layer.WMTSURLPrefix)
		fmt.Printf("  Image MIME Type: %s\n\n", layer.ImageMIMEType)
	}
	wmsBackEndUrl := layers[0].WMSBackendURL
	wmsStartParams := layers[0].WMSBackendPrefix

	// Create a new grid
	myGrid := wmts.CreateNewLausanneGridFromEnvOrFail(wmsBackEndUrl, wmsStartParams)
	numCols := myGrid.GetMaxNumCols(*zoomLevel)
	numRows := myGrid.GetMaxNumRows(*zoomLevel)
	l.Info("ℹ️ will generate % rows and %d cols for zoom level %d", numRows, numCols, *zoomLevel)

}
