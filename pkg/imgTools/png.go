package imgTools

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
)

// GeneratePng creates a PNG image with the specified color and dimensions.
func GeneratePng(red, green, blue, alpha uint8, w, h int) ([]byte, error) {
	// Create the image once.
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	blueColor := color.RGBA{R: red, G: green, B: blue, A: alpha}
	for x := 0; x < 256; x++ {
		for y := 0; y < 256; y++ {
			img.Set(x, y, blueColor)
		}
	}

	var pngImage bytes.Buffer
	err := png.Encode(&pngImage, img) // Encode to memory.
	if err != nil {
		return nil, err
	}
	return pngImage.Bytes(), nil
}

// GenerateFastPng creates a PNG with a solid color, optimized for speed.
func GenerateFastPng(red, green, blue, alpha uint8, w, h int) ([]byte, error) {
	// Create a buffer and fill it with the same color efficiently
	pixelData := make([]byte, w*h*4) // 4 bytes per pixel (RGBA)

	// Encode the color once and copy it in chunks
	colorRGBA := []byte{red, green, blue, alpha}
	for i := 0; i < len(pixelData); i += 4 {
		copy(pixelData[i:i+4], colorRGBA)
	}

	// Create the image using the filled buffer
	img := &image.RGBA{
		Pix:    pixelData,
		Stride: 4 * w,
		Rect:   image.Rect(0, 0, w, h),
	}

	// Encode PNG to memory
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GetPngImg  creates a PNG with a solid color, optimized for speed.
func GetPngImg(red, green, blue, alpha uint8, w, h int) (*image.RGBA, error) {
	// Create a buffer and fill it with the same color efficiently
	pixelData := make([]byte, w*h*4) // 4 bytes per pixel (RGBA)

	// Encode the color once and copy it in chunks
	colorRGBA := []byte{red, green, blue, alpha}
	for i := 0; i < len(pixelData); i += 4 {
		copy(pixelData[i:i+4], colorRGBA)
	}

	// Create the image using the filled buffer
	return &image.RGBA{
		Pix:    pixelData,
		Stride: 4 * w,
		Rect:   image.Rect(0, 0, w, h),
	}, nil

}

// SplitImage splits an image into tiles of a specified width and height.
// It returns a slice of images, each representing a tile.
func SplitImage(img image.Image, tileWidth, tileHeight int) ([]image.Image, error) {
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Check if the image can be evenly divided into tiles of the given dimensions.
	if imgWidth%tileWidth != 0 || imgHeight%tileHeight != 0 {
		return nil, fmt.Errorf("image dimensions (%d x %d) are not perfectly divisible by tile dimensions (%d x %d)", imgWidth, imgHeight, tileWidth, tileHeight)
	}

	numCols := imgWidth / tileWidth
	numRows := imgHeight / tileHeight
	tiles := make([]image.Image, 0, numCols*numRows)

	// Iterate over the image and extract each tile.
	for y := 0; y < numRows; y++ {
		for x := 0; x < numCols; x++ {
			tileRect := image.Rect(x*tileWidth, y*tileHeight, (x+1)*tileWidth, (y+1)*tileHeight)

			// SubImage returns an image that shares pixels with the original.
			// This is efficient as it avoids copying pixel data.
			tile := img.(interface {
				SubImage(r image.Rectangle) image.Image
			}).SubImage(tileRect)
			tiles = append(tiles, tile)
		}
	}

	return tiles, nil
}

func CropImage(bufferedImage image.Image, buffer int, l golog.MyLogger) image.Image {
	l.Debug("CropImage bufferedImg wxh = %v , buffer : %d", bufferedImage.Bounds(), buffer)
	originalWidth := bufferedImage.Bounds().Dx() - (buffer * 2)
	originalHeight := bufferedImage.Bounds().Dy() - (buffer * 2)
	l.Debug("creating new image wxh = %d x %d", originalWidth, originalHeight)
	// Create a new blank image with the original dimensions
	croppedImage := image.NewRGBA(image.Rect(0, 0, originalWidth, originalHeight))

	// Define the rectangle to crop from the buffered image
	cropRect := image.Rect(buffer, buffer, buffer+originalWidth, buffer+originalHeight)

	// Draw the cropped section onto the new image
	ZP := image.Point{
		X: buffer,
		Y: buffer,
	}
	l.Debug("about to draw  %v , %v", cropRect.Min, ZP)
	draw.Draw(croppedImage, croppedImage.Bounds(), bufferedImage, cropRect.Min, draw.Src)
	return croppedImage
}
