package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/config"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/tools"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/version"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/wmts"
	"github.com/schollz/progressbar/v3"
)

// saveWmtsTiles allows saving all png tiles for a given zoom level and layer

const (
	APP                        = "saveWmtsTiles"
	defaultWmtsConfig          = "wmtsConfig.yaml"
	defaultLayer               = "fonds_geo_osm_bdcad_couleur"
	defaultZoomLevel           = 3
	defaultMaxClientTimeOutSec = 30
	defaultMaxIdleConn         = 100
	defaultMaxIdleConnPerHost  = 100
	defaultIdleConnTimeoutSec  = 90
	defaultNumWorkers          = 4 // Default number of workers
	defaultMetaTileSize        = 4 // Number of tiles per side in a meta-tile (e.g., 2 for a 2x2 meta-tile)
	defaultBufferSize          = 50
	defaultLogName             = "stderr"
)

// metaTileTask defines a task to process a meta-tile.
type metaTileTask struct {
	zoomLevel int
	startCol  int
	startRow  int
}

func main() {
	l, err := golog.NewLogger(
		"simple",
		config.GetLogWriterFromEnvOrPanic(defaultLogName),
		config.GetLogLevelFromEnvOrPanic(golog.WarnLevel),
		fmt.Sprintf("%s:", APP),
	)
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
	ptrMetaTileSize := flag.Int("metatile", defaultMetaTileSize, "number of tiles size per request(e.g. 2 for a 2x2 meta-tile) default is 4 ")
	metaTileSize := *ptrMetaTileSize
	bufferFromEnv := config.GetBufferSizeFromEnvOrPanic(defaultBufferSize)
	// command line override
	ptrBuffer := flag.Int("buffer", bufferFromEnv, "buffer in pixel around  tiles (default is 50)")
	buffer := *ptrBuffer

	// New parameters
	clientTimeOut := flag.Int("ClientTimeOut", defaultMaxClientTimeOutSec, "client timeout in seconds")
	minZoom := flag.Int("minZoom", defaultZoomLevel, "min zoom level")
	maxZoom := flag.Int("maxZoom", defaultZoomLevel+1, "max zoom level")

	flag.Parse()

	// Capture explicitly set flags
	flagsSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagsSet[f.Name] = true
	})

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
	myGrid := wmts.CreateNewLausanneGridFromEnvOrFail(wmsBackEndUrl, wmsStartParams, l)

	client := tools.CreateHTTPClient(*clientTimeOut, defaultMaxIdleConn, defaultMaxIdleConnPerHost, defaultIdleConnTimeoutSec)

	var zoomsToProcess []int

	// Logic to handle zoom parameters
	if flagsSet["minZoom"] && flagsSet["maxZoom"] {
		if flagsSet["zoom"] {
			l.Warn("Warning: zoomLevel parameter is ignored because minZoom and maxZoom are provided")
		}
		// Validate and add range
		start := *minZoom
		end := *maxZoom
		l.Info("ℹ️ Range requested: %d to %d (Grid Min: %d, Max: %d)", start, end, myGrid.MinZoom(), myGrid.MaxZoom())

		for z := start; z <= end; z++ {
			if z < myGrid.MinZoom() || z > myGrid.MaxZoom() {
				l.Warn("Skipping zoom level %d: outside of grid capabilities [%d, %d]", z, myGrid.MinZoom(), myGrid.MaxZoom())
				continue
			}
			zoomsToProcess = append(zoomsToProcess, z)
		}
	} else {
		// Default or direct zoom usage
		// "if no one of the 3 ... are given we work like now" -> yes, defaults.
		l.Info("ℹ️ Single zoom level requested: %d", *zoomLevel)
		zoomsToProcess = append(zoomsToProcess, *zoomLevel)
	}

	for _, z := range zoomsToProcess {
		l.Info("=======================================================================")
		l.Info("🚀 Processing Zoom Level: %d", z)
		l.Info("=======================================================================")
		processZoomLevel(z, *layerName, myGrid, xMin, yMin, xMax, yMax, metaTileSize, buffer, layerConfig, basePath, client, *numWorkers, *verbose, l)
	}

	l.Info("🏁 All requested operations completed.")
}

func processZoomLevel(
	zoomLevel int,
	layerName string,
	myGrid *wmts.Grid,
	xMin, yMin, xMax, yMax float64,
	metaTileSize int,
	buffer int,
	layerConfig wmts.LayerConfig,
	basePath string,
	client *http.Client,
	numWorkers int,
	verbose bool,
	l golog.MyLogger,
) {
	// Get tile boundaries
	minCol, maxRow, err := myGrid.GetTile(xMin, yMin, zoomLevel)
	if err != nil {
		l.Fatal("💥💥 GetTile(%f, %f, %d) got error: %v", xMin, yMin, zoomLevel, err)
	}
	maxCol, minRow, err := myGrid.GetTile(xMax, yMax, zoomLevel)
	if err != nil {
		l.Fatal("💥💥 GetTile(%f, %f, %d) got error: %v", xMax, yMax, zoomLevel, err)
	}
	l.Info("ℹ️ minCol: %d, minRow: %d", minCol, minRow)
	l.Info("ℹ️ maxCol: %d, maxRow: %d", maxCol, maxRow)

	// Calculate total tiles
	totalTiles := (maxCol - minCol + 1) * (maxRow - minRow + 1)

	// Initialize progress bar
	bar := progressbar.Default(int64(totalTiles), fmt.Sprintf("Processing tiles for layer %s, zoom %d", layerName, zoomLevel))

	// Create a channel for tasks. The channel now holds metaTileTask.
	tasks := make(chan metaTileTask, (totalTiles)/(metaTileSize*metaTileSize)+1)
	var wg sync.WaitGroup

	// Channel to track completed tasks
	done := make(chan struct{}, totalTiles)

	// Start a worker pool. Each worker now processes a meta-tile.
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range tasks {
				err := myGrid.SaveTilesFromMetaTile(task.zoomLevel, task.startCol, task.startRow, metaTileSize, metaTileSize, buffer, layerConfig, basePath, client)
				if err != nil {
					l.Error("💥 Worker %d: SaveTilesFromMetaTile for zoom:%d, meta-tile at (row:%d, col:%d) failed: %v", workerID, task.zoomLevel, task.startRow, task.startCol, err)
				} else {
					if verbose {
						l.Info("ℹ️ Worker %d: zoom:%d, meta-tile at (row:%d, col:%d) saved", workerID, task.zoomLevel, task.startRow, task.startCol)
					}
					// Signal completion for each tile in the meta-tile
					for j := 0; j < metaTileSize*metaTileSize; j++ {
						done <- struct{}{}
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

	// Enqueue meta-tile tasks
	for row := minRow; row <= maxRow; row += metaTileSize {
		for col := minCol; col <= maxCol; col += metaTileSize {
			tasks <- metaTileTask{zoomLevel: zoomLevel, startCol: col, startRow: row}
		}
	}

	// Close the tasks channel and wait for workers to finish
	close(tasks)
	wg.Wait()
	// Close done channel and wait for progress bar to finish
	close(done)
	bar.Finish()
	l.Info("ℹ️ Zoom %d processed successfully", zoomLevel)
}
