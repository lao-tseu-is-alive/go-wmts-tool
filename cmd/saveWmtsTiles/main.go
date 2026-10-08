package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

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
	defaultMaxTileAge          = 24 * time.Hour   // with -skipExisting, tiles older than this are fetched again
	secondPassDelay            = 15 * time.Second // pause before retrying failed meta-tiles, to let the WMS server recover
	failedFileHeader           = "layer,zoom,startCol,startRow,metaTileSize"
	maxFailedListed            = 20 // failed meta-tiles listed at the end, the complete list is in the failed file
)

// metaTileTask defines a task to process a meta-tile.
type metaTileTask struct {
	zoomLevel int
	startCol  int
	startRow  int
	size      int // number of tiles per side of the meta-tile
}

// numTiles returns the number of png tiles in the meta-tile.
func (t metaTileTask) numTiles() int {
	return t.size * t.size
}

// desc returns the zoom and tile ranges covered by the meta-tile, for the logs.
func (t metaTileTask) desc() string {
	return wmts.MetaTileDesc(t.zoomLevel, t.startCol, t.startRow, t.size, t.size)
}

// countTiles returns the total number of png tiles in tasks.
func countTiles(tasks []metaTileTask) int {
	n := 0
	for _, t := range tasks {
		n += t.numTiles()
	}
	return n
}

// tileProcessor holds everything needed by the workers to process meta-tile tasks.
type tileProcessor struct {
	grid         *wmts.Grid
	layerConfig  wmts.LayerConfig
	basePath     string
	buffer       int
	maxRetries   int
	client       *http.Client
	numWorkers   int
	skipExisting bool
	maxTileAge   time.Duration
	verbose      bool
	l            golog.MyLogger
}

// passResult summarizes one pass of the worker pool over a list of meta-tile tasks.
type passResult struct {
	saved        int // meta-tiles
	skipped      int // meta-tiles
	tilesWritten int // png really written by the successful meta-tiles
	tilesSkipped int // fresh png kept with -skipExisting
	failed       []metaTileTask
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
	layerName := flag.String("layer", defaultLayer, "layer name in config file")
	zoomLevel := flag.Int("zoom", defaultZoomLevel, "zoom level")
	numWorkers := flag.Int("workers", defaultNumWorkers, "number of worker goroutines")
	metaTileSize := flag.Int("metatile", defaultMetaTileSize, "number of tiles size per request(e.g. 2 for a 2x2 meta-tile)")
	bufferFromEnv := config.GetBufferSizeFromEnvOrPanic(defaultBufferSize)
	// command line override
	buffer := flag.Int("buffer", bufferFromEnv, "buffer in pixel around tiles")
	clientTimeOut := flag.Int("ClientTimeOut", defaultMaxClientTimeOutSec, "client timeout in seconds")
	minZoom := flag.Int("minZoom", defaultZoomLevel, "min zoom level")
	maxZoom := flag.Int("maxZoom", defaultZoomLevel+1, "max zoom level")
	maxRetries := flag.Int("retries", tools.DefaultMaxRetries, "number of retries of a failed WMS request (with exponential backoff)")
	skipExisting := flag.Bool("skipExisting", false, "skip meta-tiles whose tiles all exist locally and are younger than -maxTileAge")
	maxTileAge := flag.Duration("maxTileAge", defaultMaxTileAge, "with -skipExisting, max age of an existing tile to be kept (e.g. 30m, 12h, 72h)")
	retryFile := flag.String("retryFile", "", "only process the failed meta-tiles listed in this file (written by a previous run), zoom flags are ignored")
	failedFile := flag.String("failedFile", "", "file where meta-tiles still failing at the end are written (default failed_<layer>_<timestamp>.csv)")

	flag.Parse()

	// Capture explicitly set flags
	flagsSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagsSet[f.Name] = true
	})

	if *numWorkers < 1 {
		l.Fatal("💥💥 -workers must be >= 1, got %d", *numWorkers)
	}
	if *metaTileSize < 1 {
		l.Fatal("💥💥 -metatile must be >= 1, got %d", *metaTileSize)
	}
	if *buffer < 0 || *buffer > 256 {
		l.Fatal("💥💥 -buffer must be between 0 and 256, got %d", *buffer)
	}
	if *clientTimeOut < 1 {
		l.Fatal("💥💥 -ClientTimeOut must be >= 1 second, got %d", *clientTimeOut)
	}
	if *maxRetries < 0 {
		l.Fatal("💥💥 -retries must be >= 0, got %d", *maxRetries)
	}
	if *maxTileAge <= 0 {
		l.Fatal("💥💥 -maxTileAge must be a positive duration, got %s", *maxTileAge)
	}
	if flagsSet["maxTileAge"] && !*skipExisting {
		l.Warn("Warning: -maxTileAge is ignored without -skipExisting")
	}

	l.Info("ℹ️ Reading config file: %s", *configFileName)
	config, err := wmts.ConfigFromYAML(*configFileName)
	if err != nil {
		l.Fatal("error loading %s layer config: %v", *configFileName, err)
	}
	basePath := config.Caches.Local.Folder
	layers := config.Layers
	// Check if there are layers loaded
	if len(layers) == 0 {
		l.Fatal("💥💥 no layers loaded from %s", *configFileName)
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

	p := &tileProcessor{
		grid:         myGrid,
		layerConfig:  layerConfig,
		basePath:     basePath,
		buffer:       *buffer,
		maxRetries:   *maxRetries,
		client:       tools.CreateHTTPClient(*clientTimeOut, defaultMaxIdleConn, defaultMaxIdleConnPerHost, defaultIdleConnTimeoutSec),
		numWorkers:   *numWorkers,
		skipExisting: *skipExisting,
		maxTileAge:   *maxTileAge,
		verbose:      *verbose,
		l:            l,
	}

	var stillFailing []metaTileTask

	if *retryFile != "" {
		if flagsSet["zoom"] || flagsSet["minZoom"] || flagsSet["maxZoom"] {
			l.Warn("Warning: zoom parameters are ignored because -retryFile is provided")
		}
		tasks, err := readFailedFile(*retryFile, *layerName)
		if err != nil {
			l.Fatal("💥💥 cannot read retry file: %v", err)
		}
		l.Info("ℹ️ %d meta-tiles to retry from %s", len(tasks), *retryFile)
		stillFailing = p.processTasks(fmt.Sprintf("Retrying failed tiles for layer %s from %s", *layerName, *retryFile), tasks)
	} else {
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
			tasks := zoomLevelTasks(myGrid, z, xMin, yMin, xMax, yMax, *metaTileSize, l)
			failed := p.processTasks(fmt.Sprintf("Processing tiles for layer %s, zoom %d", *layerName, z), tasks)
			stillFailing = append(stillFailing, failed...)
		}
	}

	if len(stillFailing) > 0 {
		fileName := *failedFile
		if fileName == "" {
			fileName = fmt.Sprintf("failed_%s_%s.csv", *layerName, time.Now().Format("20060102-150405"))
		}
		fmt.Fprintf(os.Stderr, "💥 %d meta-tiles (%d png) are still missing:\n", len(stillFailing), countTiles(stillFailing))
		for i, t := range stillFailing {
			if i == maxFailedListed {
				fmt.Fprintf(os.Stderr, "   ... and %d more\n", len(stillFailing)-maxFailedListed)
				break
			}
			fmt.Fprintf(os.Stderr, "   %s\n", t.desc())
		}
		if err := writeFailedFile(fileName, *layerName, stillFailing); err != nil {
			l.Error("💥💥 cannot write the list of failed meta-tiles: %v", err)
		} else {
			fmt.Fprintf(os.Stderr, "   they are listed in %s\n", fileName)
			fmt.Fprintf(os.Stderr, "   to fetch only them, run again with the same -config and -layer and: -retryFile %s\n", fileName)
		}
		os.Exit(1)
	}
	l.Info("🏁 All requested operations completed.")
}

// zoomLevelTasks returns the meta-tile tasks covering the bbox at the given zoom level.
func zoomLevelTasks(myGrid *wmts.Grid, zoomLevel int, xMin, yMin, xMax, yMax float64, metaTileSize int, l golog.MyLogger) []metaTileTask {
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
	l.Info("ℹ️ totalTiles: %d, ", (maxCol-minCol+1)*(maxRow-minRow+1))

	var tasks []metaTileTask
	for row := minRow; row <= maxRow; row += metaTileSize {
		for col := minCol; col <= maxCol; col += metaTileSize {
			tasks = append(tasks, metaTileTask{zoomLevel: zoomLevel, startCol: col, startRow: row, size: metaTileSize})
		}
	}
	return tasks
}

// processTasks runs all the tasks with the worker pool, then retries the failed ones
// in a second pass with a single worker. It returns the meta-tiles that still failed.
func (p *tileProcessor) processTasks(label string, tasks []metaTileTask) []metaTileTask {
	totalTiles := countTiles(tasks)
	bar := progressbar.Default(int64(totalTiles), label)

	res := p.runPass(tasks, p.numWorkers, bar)
	if len(res.failed) > 0 {
		p.l.Warn("⚠️ %d meta-tiles failed, second pass with one worker in %s", len(res.failed), secondPassDelay)
		time.Sleep(secondPassDelay)
		second := p.runPass(res.failed, 1, bar)
		res.saved += second.saved
		res.tilesWritten += second.tilesWritten
		res.failed = second.failed
	}
	// Exit keeps the real progress, Finish would show 100% even with missing tiles
	_ = bar.Exit()
	printSummary(os.Stdout, label, len(tasks), totalTiles, res)
	if accounted := res.tilesWritten + res.tilesSkipped + countTiles(res.failed); accounted != totalTiles {
		p.l.Error("💥 %s: %d png expected but %d accounted for (written + fresh + missing)", label, totalTiles, accounted)
	}
	return res.failed
}

// printSummary writes the meta-tiles and png counts of a processed zoom level (or retry file).
func printSummary(w io.Writer, label string, numMetaTiles, totalTiles int, res passResult) {
	fmt.Fprintf(w, "\n%s: %d meta-tiles, %d png expected\n", label, numMetaTiles, totalTiles)
	fmt.Fprintf(w, "  saved   : %6d meta-tiles, %8d png written\n", res.saved, res.tilesWritten)
	fmt.Fprintf(w, "  skipped : %6d meta-tiles, %8d png fresh\n", res.skipped, res.tilesSkipped)
	fmt.Fprintf(w, "  failed  : %6d meta-tiles, %8d png missing\n", len(res.failed), countTiles(res.failed))
}

// runPass distributes the tasks to numWorkers goroutines and collects the results.
func (p *tileProcessor) runPass(tasks []metaTileTask, numWorkers int, bar *progressbar.ProgressBar) passResult {
	var (
		res passResult
		mu  sync.Mutex
		wg  sync.WaitGroup
	)
	taskCh := make(chan metaTileTask)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range taskCh {
				if p.skipExisting && p.grid.IsMetaTileFresh(task.zoomLevel, task.startCol, task.startRow, task.size, task.size, p.layerConfig, p.basePath, p.maxTileAge) {
					mu.Lock()
					res.skipped++
					res.tilesSkipped += task.numTiles()
					mu.Unlock()
					_ = bar.Add(task.numTiles())
					continue
				}
				written, err := p.grid.SaveTilesFromMetaTile(task.zoomLevel, task.startCol, task.startRow, task.size, task.size, p.buffer, p.maxRetries, p.layerConfig, p.basePath, p.client)
				if err != nil {
					p.l.Error("💥 Worker %d: meta-tile %s failed: %v", workerID, task.desc(), err)
					mu.Lock()
					res.failed = append(res.failed, task)
					mu.Unlock()
					continue
				}
				if p.verbose {
					p.l.Info("ℹ️ Worker %d: meta-tile %s saved", workerID, task.desc())
				}
				mu.Lock()
				res.saved++
				res.tilesWritten += written
				mu.Unlock()
				_ = bar.Add(task.numTiles())
			}
		}(i)
	}

	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)
	wg.Wait()
	return res
}

// writeFailedFile saves the failed meta-tiles as csv, to be processed again later with -retryFile.
func writeFailedFile(path, layerName string, tasks []metaTileTask) error {
	return tools.WriteFileAtomic(path, func(w io.Writer) error {
		if _, err := fmt.Fprintf(w, "# %s\n", failedFileHeader); err != nil {
			return err
		}
		cw := csv.NewWriter(w)
		for _, t := range tasks {
			record := []string{layerName, strconv.Itoa(t.zoomLevel), strconv.Itoa(t.startCol), strconv.Itoa(t.startRow), strconv.Itoa(t.size)}
			if err := cw.Write(record); err != nil {
				return err
			}
		}
		cw.Flush()
		return cw.Error()
	})
}

// readFailedFile loads the meta-tiles written by writeFailedFile and checks they belong to layerName.
func readFailedFile(path, layerName string) ([]metaTileTask, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cr := csv.NewReader(f)
	cr.Comment = '#'
	cr.FieldsPerRecord = 5
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid content in %s: %w", path, err)
	}

	tasks := make([]metaTileTask, 0, len(records))
	for i, r := range records {
		if r[0] != layerName {
			return nil, fmt.Errorf("%s record %d is for layer %q, not for -layer %q", path, i+1, r[0], layerName)
		}
		var values [4]int
		for j, s := range r[1:] {
			v, err := strconv.Atoi(s)
			if err != nil || v < 0 {
				return nil, fmt.Errorf("%s record %d: invalid value %q, expected %s", path, i+1, s, failedFileHeader)
			}
			values[j] = v
		}
		if values[3] < 1 {
			return nil, fmt.Errorf("%s record %d: metaTileSize must be >= 1", path, i+1)
		}
		tasks = append(tasks, metaTileTask{zoomLevel: values[0], startCol: values[1], startRow: values[2], size: values[3]})
	}
	return tasks, nil
}
