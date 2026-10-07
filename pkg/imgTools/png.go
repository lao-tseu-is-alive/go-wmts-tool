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

// subImager is implemented by all the standard image types (RGBA, NRGBA, Paletted, ...).
type subImager interface {
	SubImage(r image.Rectangle) image.Image
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
	si, ok := img.(subImager)
	if !ok {
		return nil, fmt.Errorf("image type %T does not support SubImage", img)
	}

	numCols := imgWidth / tileWidth
	numRows := imgHeight / tileHeight
	tiles := make([]image.Image, 0, numCols*numRows)

	// Iterate over the image and extract each tile.
	for y := 0; y < numRows; y++ {
		for x := 0; x < numCols; x++ {
			// bounds.Min is not (0,0) when img is itself a sub-image (e.g. returned by CropImage)
			tileRect := image.Rect(x*tileWidth, y*tileHeight, (x+1)*tileWidth, (y+1)*tileHeight).Add(bounds.Min)

			// SubImage returns an image that shares pixels with the original.
			// This is efficient as it avoids copying pixel data.
			tiles = append(tiles, si.SubImage(tileRect))
		}
	}

	return tiles, nil
}

// CropImage removes a border of buffer pixels around bufferedImage.
// The returned image shares its pixels with bufferedImage (no copy, no color conversion),
// so its bounds do not start at (0,0).
func CropImage(bufferedImage image.Image, buffer int, l golog.MyLogger) image.Image {
	l.Debug("CropImage bufferedImg wxh = %v , buffer : %d", bufferedImage.Bounds(), buffer)
	cropRect := bufferedImage.Bounds().Inset(buffer)
	if si, ok := bufferedImage.(subImager); ok {
		return si.SubImage(cropRect)
	}
	// fallback for image types without SubImage: copy the cropped area into a new image
	l.Debug("image type %T has no SubImage, copying the cropped area", bufferedImage)
	croppedImage := image.NewNRGBA(image.Rect(0, 0, cropRect.Dx(), cropRect.Dy()))
	draw.Draw(croppedImage, croppedImage.Bounds(), bufferedImage, cropRect.Min, draw.Src)
	return croppedImage
}
