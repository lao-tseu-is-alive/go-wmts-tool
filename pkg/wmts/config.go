package wmts

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// CacheConfig holds the configuration for a single cache
type CacheConfig struct {
	CacheType string `yaml:"cache_type"`
	Folder    string `yaml:"folder"`
}

// Caches holds the configuration for all defined caches
type Caches struct {
	Local CacheConfig `yaml:"local"`
}

// Config holds the entire YAML structure
type Config struct {
	Caches             *Caches                `yaml:"caches"`
	LayerDefaultValues *LayerDefaultValues    `yaml:"layer_default_values"`
	Layers             map[string]LayerConfig `yaml:"layers"`
}

// ConfigFromYAML reads a YAML file and returns a map of layer configurations
func ConfigFromYAML(filePath string) (*Config, error) {
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
	myConfig := &Config{
		Caches:             config.Caches,
		LayerDefaultValues: config.LayerDefaultValues,
		Layers:             layers,
	}

	return myConfig, nil
}
