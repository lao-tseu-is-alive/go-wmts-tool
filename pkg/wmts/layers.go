package wmts

import (
	"fmt"
)

// LayerDefaultValues holds the default configuration values for layers
type LayerDefaultValues struct {
	WMSBackendURL             string    `yaml:"wms_backend_url"`
	WMSBackendPrefix          string    `yaml:"wms_backend_prefix"`
	WMTSBBox                  []float64 `yaml:"wmts_bbox"`
	WMTSURLPrefix             string    `yaml:"wmts_url_prefix"`
	WMTSURLStyle              string    `yaml:"wmts_url_style"`
	WMTSDimensionName         string    `yaml:"wmts_dimension_name"`
	WMTSDimensionYear         string    `yaml:"wmts_dimension_year"`
	WMTSMatrixSet             string    `yaml:"wmts_matrix_set"`
	ImageExtension            string    `yaml:"image_extension"`
	ImageMIMEType             string    `yaml:"image_mime_type"`
	EmptyTileDetectionSize    int       `yaml:"empty_tile_detection_size"`
	EmptyTileDetectionMD5Hash string    `yaml:"empty_tile_detection_md5_hash"`
}

// LayerConfig represents the configuration for a single layer
type LayerConfig struct {
	LayerDefaultValues `yaml:",inline"`
	WMSLayers          string `yaml:"wms_layers"`
	Name               string `yaml:"layer_name"`
	Title              string `yaml:"layer_title"`
	Abstract           string `yaml:"abstract"`
}

func PrintLayerInfo(layer LayerConfig) {
	fmt.Printf("  Title: %s\n", layer.Title)
	fmt.Printf("  WMS Backend URL: %s\n", layer.WMSBackendURL)
	fmt.Printf("  WMS Backend prefix: %s\n", layer.WMSBackendPrefix)
	fmt.Printf("  WMTS BBox: [%7.1f, %7.1f, %7.1f, %7.1f]\n", layer.WMTSBBox[0], layer.WMTSBBox[1], layer.WMTSBBox[2], layer.WMTSBBox[3])
	fmt.Printf("  WMTS URL prefix: %s\n", layer.WMTSURLPrefix)
	fmt.Printf("  WMTS URL Style: %s\n", layer.WMTSURLStyle)
	fmt.Printf("  WMTS Dimension Name: %s\n", layer.WMTSDimensionName)
	fmt.Printf("  WMTS Dimension Year: %s\n", layer.WMTSDimensionYear)
	fmt.Printf("  WMTS Matrix Set: %s\n", layer.WMTSMatrixSet)
	fmt.Printf("  WMS Layers: %s\n", layer.WMSLayers)
	fmt.Printf("  Image Extension: %s\n", layer.ImageExtension)
	fmt.Printf("  Image MIME Type: %s\n", layer.ImageMIMEType)
	fmt.Printf("  Empty Tile Detection Size: %d\n", layer.EmptyTileDetectionSize)
	fmt.Printf("  Empty Tile Detection MD5 Hash: %s\n", layer.EmptyTileDetectionMD5Hash)
	fmt.Println("-------------------------------------------")
}
