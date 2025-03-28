package wmts

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// LayerDefaultValues holds the default configuration values for layers
type LayerDefaultValues struct {
	WMSBackendURL    string `yaml:"wms_backend_url"`
	WMSBackendPrefix string `yaml:"wms_backend_prefix"`
	WMTSURLPrefix    string `yaml:"wmts_url_prefix"`
	WMTSURLStyle     string `yaml:"wmts_url_style"`
	WMTSURLYear      string `yaml:"wmts_url_year"`
	WMTSMatrixSet    string `yaml:"wmts_matrix_set"`
	ImageExtension   string `yaml:"image_extension"`
	ImageMIMEType    string `yaml:"image_mime_type"`
}

// LayerConfig represents the configuration for a single layer
type LayerConfig struct {
	LayerDefaultValues `yaml:",inline"`
	WMSLayers          string    `yaml:"wms_layers"`
	Title              string    `yaml:"title"`
	Abstract           string    `yaml:"abstract"`
	BBox               []float64 `yaml:"bbox"`
}

// Config holds the entire YAML structure
type Config struct {
	LayerDefaultValues *LayerDefaultValues    `yaml:"layer_default_values"`
	Layers             map[string]LayerConfig `yaml:"layers"`
}

// LoadLayerConfigFromYAML reads a YAML file and returns a map of layer configurations
func LoadLayerConfigFromYAML(filePath string) (map[string]LayerConfig, error) {
	// Read the YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %v", err)
	}
	// Unmarshal YAML into Config struct
	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %v", err)
	}

	// Apply defaults to each layer where applicable
	layers := make(map[string]LayerConfig)
	for name, layer := range config.Layers {
		// If LayerDefaultValues is not explicitly set in the layer, use the global defaults
		if layer.WMSBackendURL == "" && config.LayerDefaultValues != nil {
			layer.LayerDefaultValues = *config.LayerDefaultValues
		}
		layers[name] = layer
	}

	return layers, nil
}
