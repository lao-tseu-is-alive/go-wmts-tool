package config

import (
	"fmt"
	"os"
)

// GetWmsBackendUrlFromEnvOrPanic returns a string to be used with WMS based on the content of the env variable
// WMS_BACKEND_URL : should exist and contain a string with your WMS backend URL or this function will panic
func GetWmsBackendUrlFromEnvOrPanic() string {
	val, exist := os.LookupEnv("WMS_BACKEND_URL")
	if !exist {
		panic("💥💥 ERROR: ENV WMS_BACKEND_URL should contain your WMS backend URL.")
	}
	return fmt.Sprintf("%s", val)
}

// GetWmsBackendPrefixFromEnvOrPanic returns a string to be used with WMS based on the content of the env variable
// WMS_BACKEND_PREFIX : should exist and contain a string with your WMS backend prefix or this function will panic
func GetWmsBackendPrefixFromEnvOrPanic() string {
	val, exist := os.LookupEnv("WMS_BACKEND_PREFIX")
	if !exist {
		panic("💥💥 ERROR: ENV WMS_BACKEND_PREFIX should contain your WMS backend prefix.")
	}
	return fmt.Sprintf("%s", val)
}

// GetLayersConfigPathFromEnvOrPanic returns a string to be used with WMS based on the content of the env variable
// LAYERS_CONFIG_PATH : should exist and contain a string with your WMS backend prefix or this function
func GetLayersConfigPathFromEnvOrPanic() string {
	val, exist := os.LookupEnv("LAYERS_CONFIG_PATH")
	if !exist {
		panic("💥💥 ERROR: ENV LAYERS_CONFIG_PATH should contain your WMS layers configuration.")
	}
	return fmt.Sprintf("%s", val)
}
