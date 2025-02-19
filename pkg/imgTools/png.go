package imgTools

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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
