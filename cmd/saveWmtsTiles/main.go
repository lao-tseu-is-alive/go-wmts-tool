package main

import (
	"flag"
	"fmt"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/tools"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/version"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/wmts"
	"github.com/schollz/progressbar/v3"
	"log"
	"sync"
)

//saveWmtsTiles allows saving all png tiles for a given zoom level and layer

const (
	APP                        = "saveWmtsTiles"
	defaultWmtsConfig          = "wmtsConfig.yaml"
	defaultLayer               = "fonds_geo_osm_bdcad_couleur"
	defaultZoomLevel           = 2
	defaultMaxClientTimeOutSec = 10
	defaultMaxIdleConn         = 100
	defaultMaxIdleConnPerHost  = 100
	defaultIdleConnTimeoutSec  = 90
	defaultNumWorkers          = 4 // Default number of workers
)

type tileTask struct {
	zoomLevel int
	col       int
	row       int
}

func main() {
	l, err := golog.NewLogger("zap", golog.DebugLevel, fmt.Sprintf("%s:", APP))
	if err != nil {
		log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", APP, version.VERSION, version.Build, version.REPOSITORY)
	// get the YAML config file name received from the config parameter
	configFileName := flag.String("config", defaultWmtsConfig, "config file name")
	verbose := flag.Bool("verbose", false, "verbose output")
	layerName := flag.String("layer", defaultLayer, "config file name")
	zoomLevel := flag.Int("zoom", defaultZoomLevel, "zoom level")
	numWorkers := flag.Int("workers", defaultNumWorkers, "number of worker goroutines")
	flag.Parse()

	l.Info("ℹ️ Using zoom level : %d", *zoomLevel)
	l.Info("ℹ️ Using layer : %s", *layerName)
	l.Info("ℹ️ Reading config file: %s", *configFileName)
	config, err := wmts.ConfigFromYAML(*configFileName)
	if err != nil {
		l.Fatal("error loading %s layer config: %v", *configFileName, err)
	}
	basePath := config.Caches.Local.Folder
	layers := config.Layers
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
		if *verbose {
			wmts.PrintLayerInfo(layer)
		}
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
	// Get tile boundaries
	numCols := myGrid.GetMaxNumCols(*zoomLevel)
	numRows := myGrid.GetMaxNumRows(*zoomLevel)
	l.Info("ℹ️ will generate %d rows and %d cols for zoom level %d", numRows, numCols, *zoomLevel)
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
	// Calculate total tiles
	totalTiles := (maxCol - minCol + 1) * (maxRow - minRow + 1)

	// Initialize progress bar
	bar := progressbar.Default(int64(totalTiles), fmt.Sprintf("Processing tiles for layer %s, zoom %d", *layerName, *zoomLevel))

	// Create a channel for tasks
	tasks := make(chan tileTask, totalTiles)
	var wg sync.WaitGroup

	// Channel to track completed tasks
	done := make(chan struct{}, totalTiles)

	// Start a worker pool
	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for row := minRow; row <= maxRow; row++ {
				for task := range tasks {
					imagePath, err := myGrid.SaveTileImage(task.zoomLevel, task.col, task.row, layers[*layerName], basePath, client)
					if err != nil {
						l.Error("💥 Worker %d: SaveTileImage for zoom:%d, tile (row:%d, col:%d) trying to save image to: %s got error: %v", workerID, task.zoomLevel, task.row, task.col, imagePath, err)
					} else {
						if *verbose {
							l.Info("ℹ️ Worker %d: zoom:%d, tile (row:%3d, col:%3d) saved image to: %s", workerID, task.zoomLevel, task.row, task.col, imagePath)
						}
						done <- struct{}{} // Signal task completion
					}
				}
			}
		}(i)
	}
	// Start a goroutine to update the progress bar
	go func() {
		for range done {
			bar.Add(1) // Increment progress bar
		}
	}()
	// Enqueue tasks
	for row := minRow; row <= maxRow; row++ {
		for col := minCol; col <= maxCol; col++ {
			tasks <- tileTask{zoomLevel: *zoomLevel, col: col, row: row}
		}
	}

	// Close the tasks channel and wait for workers to finish
	close(tasks)
	wg.Wait()
	// Close done channel and wait for progress bar to finish
	close(done)
	bar.Finish()
	l.Info("ℹ️ All tiles processed successfully")
}
