package main

import (
	"encoding/json"
	"fmt"
	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/wmts"
	"log"
	"net/http"
	"strconv"
)

const WMSBackendUrl = "https://carto.lausanne.ch/mapserv_proxy"
const WMSStartParams = "ogcserver=source+for+image%2Fpng&"

type TileInfoResponse struct {
	Zoom   int       `json:"zoom,omitempty"`
	Col    int       `json:"col,omitempty"`
	Row    int       `json:"row,omitempty"`
	WmsUrl string    `json:"wms_url,omitempty"`
	BBox   []float64 `json:"bbox,omitempty"`
}

// corsMiddleware adds CORS headers to allow requests from any origin.
// THIS IS FOR DEVELOPMENT ONLY.  See later examples for production-ready solutions.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")                                // Allow any origin
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE") // Allowed methods
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")     // Allowed headers

		// Handle preflight requests (OPTIONS).  Important for POST, PUT, DELETE, and requests with custom headers.
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getTileInfoByXYHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Get parameters using r.PathValue().  MUCH cleaner!
	zoomStr := r.PathValue("zoom")
	xStr := r.PathValue("x")
	yStr := r.PathValue("y")

	// 1. Extract parameters from the URL path.
	/*
		zoomStr := r.URL.Query().Get("zoom")
		xStr := r.URL.Query().Get("x")
		yStr := r.URL.Query().Get("y")
	*/
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

	// 3. Instantiate LausanneGrid.
	chGrid := wmts.NewLausanneGrid()

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
	wmsURL := fmt.Sprintf("%s?%s%s", WMSBackendUrl, WMSStartParams, buildQueryString(params))

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

// buildQueryString builds a URL query string from a map of parameters.
func buildQueryString(params map[string]string) string {
	queryString := ""
	first := true
	for k, v := range params {
		if !first {
			queryString += "&"
		}
		queryString += fmt.Sprintf("%s=%s", k, v)
		first = false
	}
	return queryString
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /getTileByXY/{zoom}/{x}/{y}", getTileInfoByXYHandler) //Register the handler
	// Wrap the mux with the CORS middleware.  This is the key change.
	handler := corsMiddleware(mux)
	port := 8000
	log.Printf("Server listening on port %d...", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), handler))
}
