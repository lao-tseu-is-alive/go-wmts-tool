package wmts

// NewLausanneGrid creates and initializes a new WMTS Grid instance for Lausanne in Switzerland.
func NewLausanneGrid() *Grid {
	g := &Grid{
		Bbox: BBox{
			XMin: 2420000.0,
			YMin: 1030000.0,
			XMax: 2900000.0,
			YMax: 1350000.0,
		},
		SpatialREF:      2056,
		TileURLTemplate: "{zoom}/{tileRow}/{tileCol}.png",
		UNIT:            "meters",
		MetersPerUnit:   1,
		TileSize:        256,
		topLeftX:        2420000.0,
		topLeftY:        1350000.0,
		tileSize:        256,
		resolutions: map[int]map[string]float64{
			0: {"ScaleDenominator": 178571.42857142858, "cellSize": 50.0, "MatrixWidth": 38.0, "MatrixHeight": 25.0},
			1: {"ScaleDenominator": 71428.57142857143, "cellSize": 20.0, "MatrixWidth": 94.0, "MatrixHeight": 63.0},
			2: {"ScaleDenominator": 35714.28571428572, "cellSize": 10.0, "MatrixWidth": 188.0, "MatrixHeight": 125.0},
			3: {"ScaleDenominator": 17857.14285714286, "cellSize": 5.0, "MatrixWidth": 375.0, "MatrixHeight": 250.0},
			4: {"ScaleDenominator": 8928.57142857143, "cellSize": 2.5, "MatrixWidth": 750.0, "MatrixHeight": 500.0},
			5: {"ScaleDenominator": 3571.4285714285716, "cellSize": 1.0, "MatrixWidth": 1875.0, "MatrixHeight": 1250.0},
			6: {"ScaleDenominator": 1785.7142857142858, "cellSize": 0.5, "MatrixWidth": 3750.0, "MatrixHeight": 2500.0},
			7: {"ScaleDenominator": 892.8571428571429, "cellSize": 0.25, "MatrixWidth": 7500.0, "MatrixHeight": 5000.0},
			8: {"ScaleDenominator": 357.14285714285717, "cellSize": 0.1, "MatrixWidth": 18750.0, "MatrixHeight": 12500.0},
			9: {"ScaleDenominator": 178.57142857142858, "cellSize": 0.05, "MatrixWidth": 37500.0, "MatrixHeight": 25000.0},
		},
	}
	return g
}
