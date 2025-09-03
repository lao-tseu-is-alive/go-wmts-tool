package config

import (
	"fmt"
	"os"
	"strconv"
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

// GetBufferSizeFromEnvOrPanic returns the buffer to use in WMS queries
func GetBufferSizeFromEnvOrPanic(defaultBuffer int) int {
	buffer := defaultBuffer
	var err error
	val, exist := os.LookupEnv("BUFFER_SIZE")
	if exist {
		buffer, err = strconv.Atoi(val)
		if err != nil {
			panic(fmt.Errorf("💥💥 ERROR:  ENV BUFFER_SIZE should contain a valid integer. %v", err))
		}
	}
	if buffer < 0 || buffer > 256 {
		panic(fmt.Errorf("💥💥 ERROR: BUFFER_SIZE should contain an integer between 0 and 256 inclusive. Err: %v", err))
	}
	return buffer
}
