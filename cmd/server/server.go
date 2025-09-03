package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/config"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/tools"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/version"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/wmts"
)

const (
	defaultPort                = 8000
	defaultServerIp            = "0.0.0.0"
	defaultWebRootDir          = "wmtsProxyFront/dist/"
	defaultWmtsUrlPrefix       = "tiles/1.0.0"
	defaultWmtsUrlStyle        = "default"
	defaultMaxClientTimeOutSec = 10
	defaultMaxIdleConn         = 100
	defaultMaxIdleConnPerHost  = 100
	defaultIdleConnTimeoutSec  = 90
	defaultBufferSize          = 50
	formatTraceRequest         = "[%s] %s '%s', IP: [%s],%s\n"
	defaultLogName             = "stderr"
)

type TileInfoResponse struct {
	Zoom   int       `json:"zoom,omitempty"`
	Col    int       `json:"col,omitempty"`
	Row    int       `json:"row,omitempty"`
	WmsUrl string    `json:"wms_url,omitempty"`
	BBox   []float64 `json:"bbox,omitempty"`
}

// content holds our static web server content.
//
//go:embed all:wmtsProxyFront/dist
var content embed.FS

// Extract parameter parsing to a separate function for better readability
func parseTileInfoByXYParams(r *http.Request) (layer string, zoom int, x, y float64, err error) {
	// 1. Get parameters using r.PathValue().  MUCH cleaner!
	layer = r.PathValue("layer")
	zoomStr := r.PathValue("zoom")
	xStr := r.PathValue("x")
	yStr := r.PathValue("y")

	// 2. Convert parameters to the correct types, with error handling.
	zoom, err = strconv.Atoi(zoomStr)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("invalid zoom level: %w", err)
	}
	x, err = strconv.ParseFloat(xStr, 64)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("invalid x coordinate: %w", err)
	}
	y, err = strconv.ParseFloat(yStr, 64)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("invalid y coordinate: %w", err)
	}

	return layer, zoom, x, y, nil
}

func parseTileParams(r *http.Request) (layer string, zoom, col, row int, err error) {
	layer = r.PathValue("layer")
	zoomStr := r.PathValue("zoom")
	colStr := r.PathValue("col")
	rowStr := r.PathValue("row")

	zoom, err = strconv.Atoi(zoomStr)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("invalid zoom level: %w", err)
	}

	col, err = strconv.Atoi(colStr)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("invalid column: %w", err)
	}

	row, err = strconv.Atoi(rowStr)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("invalid row: %w", err)
	}

	return layer, zoom, col, row, nil
}

func GetMyDefaultHandler(s *gohttp.Server, webRootDir string, content embed.FS) http.HandlerFunc {
	handlerName := "GetMyDefaultHandler"
	logger := s.GetLog()
	logger.Debug("Initial call to %s with webRootDir:%s", handlerName, webRootDir)
	// Create a subfolder filesystem to serve only the content of wmtsProxyFront/dist
	//subFS, err := fs.Sub(content, fmt.Sprintf("%s", defaultWebRootDir))
	subFS, err := fs.Sub(content, "wmtsProxyFront/dist")
	if err != nil {
		logger.Fatal("Error creating sub-filesystem: %v", err)
	}
	// Debug: List embedded files
	/*
		files, _ := fs.ReadDir(subFS, ".")
		for _, file := range files {
			logger.Debug("Embedded file: %s", file.Name())
		}
	*/
	// Create a file server handler for the embed filesystem
	handler := http.FileServer(http.FS(subFS))

	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug(formatTraceRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr, "")
		handler.ServeHTTP(w, r)
	}
}

func GetLayersInfoHandler(layers map[string]wmts.LayerConfig, l golog.MyLogger) http.HandlerFunc {
	handlerName := "GetLayersInfoHandler"
	l.Debug("Initial call to %s", handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug(formatTraceRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr, "")
		// Encode the response as JSON and send it.
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(layers); err != nil {
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
			return
		}
	}
}

func getTileInfoByXYHandler(chGrid *wmts.Grid, layers map[string]wmts.LayerConfig, l golog.MyLogger) http.HandlerFunc {
	handlerName := "getTileInfoByXYHandler"
	buffer := config.GetBufferSizeFromEnvOrPanic(defaultBufferSize)
	l.Debug("Initial call to %s, buffer size: %d", handlerName, buffer)
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug(formatTraceRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr, "")
		layer, zoom, x, y, err := parseTileInfoByXYParams(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		l.Info("getTileInfoByXYHandler: layer:%s, zoom:%d, x:%f, y:%f", layer, zoom, x, y)

		// Look up layer config
		layerConfig, exists := layers[layer]
		if !exists {
			l.Error("invalid layer request: %s", layer)
			// Maybe try using structured logging if logger supports it:
			// l.Error("invalid layer request", "layer", layer, "remote_addr", r.RemoteAddr)
			http.Error(w, "Invalid layer", http.StatusBadRequest)
			return
		}

		// 4. Perform calculations, handling potential errors from the lausanne wmts grid package.
		col, row, err := chGrid.GetTile(x, y, zoom)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest) // Forward the error message
			return
		}

		bbox, err := chGrid.GetTileBBox(zoom, col, row)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		layers := layerConfig.WMSLayers

		params := chGrid.GetWMSParams(*bbox, layers, int(chGrid.GetTileWidth()), int(chGrid.GetTileHeight()), buffer, "png") // Use GetTileWidth

		// 5. Build the WMS URL.
		wmsURL := fmt.Sprintf("%s?%s%s", chGrid.WmsBackendUrl, chGrid.WmsStartParams, tools.BuildQueryString(params))

		bboxArray := bbox.ToArray()

		// 6. Create the response.
		tileInfo := TileInfoResponse{
			Zoom:   zoom,
			Col:    col,
			Row:    row,
			WmsUrl: wmsURL,
			BBox:   bboxArray,
		}

		// 7. Encode the response as JSON and send it.
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(tileInfo); err != nil {
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
			return
		}
	}
}

func getTileImageHandler(chGrid *wmts.Grid, layers map[string]wmts.LayerConfig, basePath string, l golog.MyLogger) http.HandlerFunc {
	handlerName := "getTileImageHandler"
	buffer := config.GetBufferSizeFromEnvOrPanic(defaultBufferSize)
	l.Debug("Initial call to %s, buffer size: %d", handlerName, buffer)
	client := tools.CreateHTTPClient(defaultMaxClientTimeOutSec, defaultMaxIdleConn, defaultMaxIdleConnPerHost, defaultIdleConnTimeoutSec)
	return func(w http.ResponseWriter, r *http.Request) {
		l.Debug(formatTraceRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr, "")
		layer, zoom, col, row, err := parseTileParams(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		l.Info("getTileImageHandler: layer:%s, zoom:%d, col:%d, row:%d", layer, zoom, col, row)
		// Look up layer config
		layerConfig, exists := layers[layer]
		if !exists {
			l.Error("invalid layer request: %s", layer)
			http.Error(w, "Invalid layer", http.StatusBadRequest)
			return
		}

		// 4. check if tile exists
		if chGrid.IsValidTile(zoom, col, row) == false {
			errMsg := fmt.Sprintf("invalid tile request for zoom:%d, col:%d, row:%d", zoom, col, row)
			l.Error(errMsg)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}

		bbox, err := chGrid.GetTileBBox(zoom, col, row)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		layers := layerConfig.WMSLayers
		params := chGrid.GetWMSParams(*bbox, layers, int(chGrid.GetTileWidth()), int(chGrid.GetTileHeight()), buffer, "png") // Use GetTileWidth

		// 5. Build the WMS URL.
		wmsURL := fmt.Sprintf("%s?%s%s", chGrid.WmsBackendUrl, chGrid.WmsStartParams, tools.BuildQueryString(params))

		imgPath := wmts.GetWmtsImgPath(basePath, layerConfig.WMTSURLPrefix, layerConfig.Name, layerConfig.WMTSURLStyle, layerConfig.WMTSDimensionYear, layerConfig.WMTSMatrixSet, "png", zoom, row, col)
		// check if tile is in cache
		_, err = os.Stat(imgPath)
		if err != nil {
			l.Debug("file %s is not in cache, downloading: %s", imgPath, wmsURL)
			err = tools.GetPngFromUrl(client, wmsURL, imgPath, buffer, 2, l)
			if err != nil {
				errMsg := fmt.Sprintf("error in GetPngFromUrl tile  zoom:%d, col:%d, row:%d", zoom, col, row)
				l.Error(errMsg)
				http.Error(w, errMsg, http.StatusInternalServerError)
				return
			}
		}
		// Open the image file
		l.Debug("reading local tile %s", imgPath)
		file, err := os.Open(imgPath)
		if err != nil {
			errMsg := fmt.Sprintf("error doing os.Open(imgPath:%s)", imgPath)
			l.Error(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
		defer file.Close()
		// Using http.ServeContent to efficiently serve the file content.
		// This function handles a number of important HTTP features automatically:
		// - Caching: It supports `If-Modified-Since` and `If-None-Match` headers,
		//   allowing the browser to use a cached version of the file and
		//   avoiding unnecessary data transfer.
		// - Range Requests: It correctly handles `Range` headers, which allows
		//   clients to request specific portions of the file.
		// - Content Headers: It sets the correct `Content-Type` and `Content-Length` headers
		//   for the response.
		//
		// We pass a `time.Now()` as the `modtime` because the file is dynamically generated
		// and we want to prevent clients from caching it for too long, as its content
		// might change in the future.
		http.ServeContent(w, r, filepath.Base(imgPath), time.Now(), file)

	}
}

func main() {
	l, err := golog.NewLogger(
		"simple",
		config.GetLogWriterFromEnvOrPanic(defaultLogName),
		config.GetLogLevelFromEnvOrPanic(golog.InfoLevel),
		fmt.Sprintf("%s:", version.APP),
	)
	if err != nil {
		log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", version.APP, version.VERSION, version.Build, version.REPOSITORY)

	configPath := config.GetLayersConfigPathFromEnvOrPanic()
	myConfig, err := wmts.ConfigFromYAML(configPath)
	if err != nil {
		l.Fatal("error loading %s layer config: %v", configPath, err)
	}
	basePath := myConfig.Caches.Local.Folder
	layers := myConfig.Layers
	// Check if there are layers loaded
	if len(layers) == 0 {
		l.Fatal("no layers loaded from %s", configPath)
	}
	// Print loaded layers for info
	firstLayer := ""
	for name, layer := range layers {
		firstLayer = name
		wmts.PrintLayerInfo(layer)
	}

	wmsBackEndUrl := layers[firstLayer].WMSBackendURL
	wmsStartParams := layers[firstLayer].WMSBackendPrefix

	// Create a new grid
	myGrid := wmts.CreateNewLausanneGridFromEnvOrFail(wmsBackEndUrl, wmsStartParams, l)

	myVersionReader := gohttp.NewSimpleVersionReader(version.APP, version.VERSION, version.REPOSITORY, version.Build)
	server := gohttp.CreateNewServerFromEnvOrFail(
		defaultPort,
		defaultServerIp,
		myVersionReader,
		l)
	mux := server.GetRouter()

	mux.Handle("GET /layersInfo", gohttp.CorsMiddleware(GetLayersInfoHandler(layers, l)))

	// route to retrieve information about a tile surrounding the given coordinates
	mux.Handle("GET /getTileByXY/{layer}/{zoom}/{x}/{y}", gohttp.CorsMiddleware(getTileInfoByXYHandler(myGrid, layers, l)))

	wmtsUrlTemplate := fmt.Sprintf("/%s/{layer}/%s/{year}/{matrixSet}/{zoom}/{row}/{col}", defaultWmtsUrlPrefix, defaultWmtsUrlStyle)
	l.Debug("tiles url template: %s", wmtsUrlTemplate)
	mux.Handle(fmt.Sprintf("GET %s", wmtsUrlTemplate), gohttp.CorsMiddleware(getTileImageHandler(myGrid, layers, basePath, l)))

	mux.HandleFunc("GET /", GetMyDefaultHandler(server, defaultWebRootDir, content))
	server.StartServer()
}
