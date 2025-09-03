package wmts

import "github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"

// NewLausanneGrid creates and initializes a new WMTS Grid instance for Lausanne in Switzerland.
func NewLausanneGrid(wmsBackEndUrl, wmsStartParams string, l golog.MyLogger) *Grid {
	if wmsBackEndUrl == "" {
		panic("💥💥 panic in NewLausanneGrid : wmsBackEndUrl cannot be empty")
	}
	if l == nil {
		panic("💥💥 panic in NewLausanneGrid : logger cannot be nil")
	}
	g := &Grid{
		Bbox: BBox{
			XMin: 2420000.0,
			YMin: 1030000.0,
			XMax: 2900000.0,
			YMax: 1350000.0,
		},
		SpatialREF:      DefaultSpatialRef,
		TileURLTemplate: "{zoom}/{tileRow}/{tileCol}.png",
		UNIT:            "meters",
		MetersPerUnit:   1,
		TileSize:        DefaultTileSize,
		topLeftX:        2420000.0,
		topLeftY:        1350000.0,
		WmsBackendUrl:   wmsBackEndUrl,
		WmsStartParams:  wmsStartParams,
		resolutions: map[int]Resolution{
			0: {ScaleDenominator: 178571.42857142858, CellSize: 50.0, MatrixWidth: 38.0, MatrixHeight: 25},
			1: {ScaleDenominator: 71428.57142857143, CellSize: 20.0, MatrixWidth: 94.0, MatrixHeight: 63},
			2: {ScaleDenominator: 35714.28571428572, CellSize: 10.0, MatrixWidth: 188.0, MatrixHeight: 125},
			3: {ScaleDenominator: 17857.14285714286, CellSize: 5.0, MatrixWidth: 375.0, MatrixHeight: 250},
			4: {ScaleDenominator: 8928.57142857143, CellSize: 2.5, MatrixWidth: 750.0, MatrixHeight: 500},
			5: {ScaleDenominator: 3571.4285714285716, CellSize: 1.0, MatrixWidth: 1875.0, MatrixHeight: 1250},
			6: {ScaleDenominator: 1785.7142857142858, CellSize: 0.5, MatrixWidth: 3750.0, MatrixHeight: 2500},
			7: {ScaleDenominator: 892.8571428571429, CellSize: 0.25, MatrixWidth: 7500.0, MatrixHeight: 5000},
			8: {ScaleDenominator: 357.14285714285717, CellSize: 0.1, MatrixWidth: 18750.0, MatrixHeight: 12500},
			9: {ScaleDenominator: 178.57142857142858, CellSize: 0.05, MatrixWidth: 37500.0, MatrixHeight: 25000},
		},
		l: l,
	}
	return g
}

func CreateNewLausanneGridFromEnvOrFail(wmsBackEndUrl, wmsStartParams string, l golog.MyLogger) *Grid {
	return NewLausanneGrid(wmsBackEndUrl, wmsStartParams, l)

}
