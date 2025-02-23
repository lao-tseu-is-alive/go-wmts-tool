package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/imgTools"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/tools"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/version"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/wmts"
	"image/png"
	"io/fs"
	"log"
	"net/http"
	"strconv"
)

const (
	APP               = "goCloudK8sCommonDemoServer"
	defaultPort       = 8000
	defaultServerIp   = "0.0.0.0"
	defaultWebRootDir = "front/dist/"
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

func getTileInfoByXYHandler(chGrid *wmts.Grid, l golog.MyLogger) http.HandlerFunc {
	handlerName := "getTileInfoByXYHandler"
	l.Debug("Initial call to %s", handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		gohttp.TraceRequest(handlerName, r, l)
		// 1. Get parameters using r.PathValue().  MUCH cleaner!
		zoomStr := r.PathValue("zoom")
		xStr := r.PathValue("x")
		yStr := r.PathValue("y")

		gutterStr := r.URL.Query().Get("gutter") // Get gutter as a string
		// Set default value for gutter
		gutter := 0

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
		if gutterStr != "" {
			gutter, err = strconv.Atoi(gutterStr)
			if err != nil {
				http.Error(w, "Invalid gutter value", http.StatusBadRequest)
				return
			}
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
		layers := "osm_bdcad_couleur_msgroup,planville_cs_autres_msgroup,planville_cs_bati_pol_sout,planville_marquage_msgroup,planville_od_objets_msgroup,planville_arbres_goeland_msgroup,planville_cs_bati_msgroup,planville_od_labels_msgroup"

		params := chGrid.GetWMSParams(*bbox, layers, gutter, int(chGrid.GetTileWidth()), int(chGrid.GetTileHeight()), "png") // Use GetTileWidth

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

func getTileImageHandler(chGrid *wmts.Grid, l golog.MyLogger) http.HandlerFunc {
	handlerName := "getTileImageHandler"
	return func(w http.ResponseWriter, r *http.Request) {
		gohttp.TraceRequest(handlerName, r, l)
		// 1. Get parameters using r.PathValue().  MUCH cleaner!
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
		// 4. check if tile exists
		if chGrid.IsValidTile(zoom, col, row) == false {
			errMsg := fmt.Sprintf("invalid tile request for zoom:%d, col:%d, row:%d", zoom, col, row)
			l.Error(errMsg)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		//chGrid.GetTileImage(zoom, col, row)
		// return the correct content type header for the image
		w.Header().Set("Content-Type", "image/png")
		img, err := imgTools.GetPngImg(0, 0, 250, 100, 256, 256)
		// Encode PNG to memory
		err = png.Encode(w, img)
		if err != nil {
			http.Error(w, "Error encoding PNG", http.StatusInternalServerError)
			return
		}
	}
}

func main() {
	l, err := golog.NewLogger("zap", golog.DebugLevel, APP)
	if err != nil {
		log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting App:'%s', ver:%s, build:%s, from: %s", APP, version.VERSION, version.Build, version.REPOSITORY)
	// Create a new grid
	myGrid := wmts.CreateNewLausanneGridFromEnvOrFail()
	myVersionReader := gohttp.NewSimpleVersionReader(APP, version.VERSION, version.REPOSITORY, version.Build)
	server := gohttp.CreateNewServerFromEnvOrFail(
		defaultPort,
		defaultServerIp,
		myVersionReader,
		l)
	mux := server.GetRouter()
	mux.Handle("GET /getTileByXY/{zoom}/{x}/{y}", gohttp.CorsMiddleware(getTileInfoByXYHandler(myGrid, l)))
	mux.Handle("GET /tile/{zoom}/{row}/{col}", gohttp.CorsMiddleware(getTileImageHandler(myGrid, l)))
	mux.Handle("GET /*", GetMyDefaultHandler(server, defaultWebRootDir, content))
	server.StartServer()
}
