package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/tools"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/version"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/wmts"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	defaultPort                = 8000
	defaultServerIp            = "0.0.0.0"
	defaultWebRootDir          = "front/dist/"
	defaultWmtsUrlPrefix       = "tiles/1.0.0"
	defaultWmtsUrlStyle        = "default"
	defaultWmtsUrlYear         = "2021"
	defaultWmtsMatrixSet       = "swissgrid_05"
	defaultMaxClientTimeOutSec = 10
	defaultMaxIdleConn         = 100
	defaultMaxIdleConnPerHost  = 100
	defaultIdleConnTimeoutSec  = 90
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
//go:embed all:front/dist
var content embed.FS

func GetMyDefaultHandler(s *gohttp.Server, webRootDir string, content embed.FS) http.HandlerFunc {
	handlerName := "GetMyDefaultHandler"
	logger := s.GetLog()
	logger.Debug("Initial call to %s with webRootDir:%s", handlerName, webRootDir)
	// Create a subfolder filesystem to serve only the content of front/dist
	subFS, err := fs.Sub(content, "front/dist")
	if err != nil {
		logger.Fatal("Error creating sub-filesystem: %v", err)
	}

	// Create a file server handler for the embed filesystem
	handler := http.FileServer(http.FS(subFS))

	return func(w http.ResponseWriter, r *http.Request) {
		gohttp.TraceRequest(handlerName, r, logger)
		handler.ServeHTTP(w, r)
	}
}

func getTileInfoByXYHandler(chGrid *wmts.Grid, layers map[string]wmts.LayerConfig, l golog.MyLogger) http.HandlerFunc {
	handlerName := "getTileInfoByXYHandler"
	l.Debug("Initial call to %s", handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		gohttp.TraceRequest(handlerName, r, l)
		// 1. Get parameters using r.PathValue().  MUCH cleaner!
		layerStr := r.PathValue("layer")
		zoomStr := r.PathValue("zoom")
		xStr := r.PathValue("x")
		yStr := r.PathValue("y")

		// 2. Convert parameters to the correct types, with error handling.
		zoom, err := strconv.Atoi(zoomStr)
		if err != nil {
			http.Error(w, "Invalid zoom level", http.StatusBadRequest)
			return
		}
		x, err := strconv.ParseFloat(xStr, 64)
		if err != nil {
			http.Error(w, "Invalid x coordinate", http.StatusBadRequest)
			return
		}
		y, err := strconv.ParseFloat(yStr, 64)
		if err != nil {
			http.Error(w, "Invalid y coordinate", http.StatusBadRequest)
			return
		}

		// Look up layer config
		layerConfig, exists := layers[layerStr]
		if !exists {
			l.Error("invalid layer request: %s", layerStr)
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

		params := chGrid.GetWMSParams(*bbox, layers, int(chGrid.GetTileWidth()), int(chGrid.GetTileHeight()), "png") // Use GetTileWidth

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

func getTileImageHandler(chGrid *wmts.Grid, layers map[string]wmts.LayerConfig, l golog.MyLogger) http.HandlerFunc {
	handlerName := "getTileImageHandler"
	l.Debug("Initial call to %s", handlerName)
	client := tools.CreateHTTPClient(defaultMaxClientTimeOutSec, defaultMaxIdleConn, defaultMaxIdleConnPerHost, defaultIdleConnTimeoutSec)
	return func(w http.ResponseWriter, r *http.Request) {
		gohttp.TraceRequest(handlerName, r, l)
		// 1. Get parameters using r.PathValue().  MUCH cleaner!
		layerStr := r.PathValue("layer")
		zoomStr := r.PathValue("zoom")
		colStr := r.PathValue("col")
		rowStr := r.PathValue("row")
		// 2. Convert parameters to the correct types, with error handling.
		zoom, err := strconv.Atoi(zoomStr)
		if err != nil {
			http.Error(w, "Invalid zoom level", http.StatusBadRequest)
			return
		}
		col, err := strconv.Atoi(colStr)
		if err != nil {
			http.Error(w, "Invalid column", http.StatusBadRequest)
			return
		}
		row, err := strconv.Atoi(rowStr)
		if err != nil {
			http.Error(w, "Invalid row", http.StatusBadRequest)
			return
		}
		// Look up layer config
		layerConfig, exists := layers[layerStr]
		if !exists {
			l.Error("invalid layer request: %s", layerStr)
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
		params := chGrid.GetWMSParams(*bbox, layers, int(chGrid.GetTileWidth()), int(chGrid.GetTileHeight()), "png") // Use GetTileWidth

		// 5. Build the WMS URL.
		wmsURL := fmt.Sprintf("%s?%s%s", chGrid.WmsBackendUrl, chGrid.WmsStartParams, tools.BuildQueryString(params))

		prefix := defaultWmtsUrlPrefix
		style := defaultWmtsUrlStyle
		imgPath := wmts.GetWmtsImgPath(prefix, layerStr, style, defaultWmtsUrlYear, defaultWmtsMatrixSet, "png", zoom, row, col)
		// check if tile is in cache
		_, err = os.Stat(imgPath)
		if err != nil {
			l.Debug("file %s is not in cache, downloading: %s", imgPath, wmsURL)
			err = tools.GetPngFromUrl(client, wmsURL, imgPath, 2)
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
		// get the size of the file to add content-length header
		fileInfo, err := file.Stat()
		if err != nil {
			errMsg := fmt.Sprintf("error %v, doing file.Stat : %s", err, imgPath)
			l.Error(errMsg)
			http.Error(w, errMsg, http.StatusInternalServerError)
			return
		}
		// return the correct content type header for the image
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
		// write the img to the response
		_, err = io.Copy(w, file)
		if err != nil {
			http.Error(w, "Error reading png img", http.StatusInternalServerError)
			return
		}
	}
}

func main() {
	l, err := golog.NewLogger("zap", golog.DebugLevel, version.APP)
	if err != nil {
		log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", version.APP, version.VERSION, version.Build, version.REPOSITORY)
	// Create a new grid
	myGrid := wmts.CreateNewLausanneGridFromEnvOrFail()

	configPath := "config.yaml"
	layers, err := wmts.LoadLayerConfigFromYAML(configPath)
	if err != nil {
		l.Fatal("error loading %s layer config: %v", configPath, err)
	}
	// Print loaded layers for info
	for name, layer := range layers {
		l.Debug("Layer: %s\n", name)
		l.Debug("  Title: %s\n", layer.Title)
		l.Debug("  WMS Backend URL: %s\n", layer.WMSBackendURL)
		l.Debug("  WMS Layers: %s\n", layer.WMSLayers)
		l.Debug("  BBox: %v\n", layer.BBox)
		l.Debug("  WMTS URL Prefix: %s\n", layer.WMTSURLPrefix)
		l.Debug("  Image MIME Type: %s\n\n", layer.ImageMIMEType)
	}

	myVersionReader := gohttp.NewSimpleVersionReader(version.APP, version.VERSION, version.REPOSITORY, version.Build)
	server := gohttp.CreateNewServerFromEnvOrFail(
		defaultPort,
		defaultServerIp,
		myVersionReader,
		l)
	mux := server.GetRouter()
	mux.Handle("GET /getTileByXY/{layer}/{zoom}/{x}/{y}", gohttp.CorsMiddleware(getTileInfoByXYHandler(myGrid, layers, l)))
	wmtsUrlTemplate := fmt.Sprintf("/%s/{layer}/%s/{year}/{matrixSet}/{zoom}/{row}/{col}", defaultWmtsUrlPrefix, defaultWmtsUrlStyle)
	l.Info("tiles url template: %s", wmtsUrlTemplate)
	// wmtsUrlTemplate := "/tiles/1.0.0/{layer}/default/{year}/{matrixSet}/{zoom}/{row}/{col}"
	mux.Handle(fmt.Sprintf("GET %s", wmtsUrlTemplate), gohttp.CorsMiddleware(getTileImageHandler(myGrid, layers, l)))
	mux.Handle("GET /*", GetMyDefaultHandler(server, defaultWebRootDir, content))
	server.StartServer()
}
