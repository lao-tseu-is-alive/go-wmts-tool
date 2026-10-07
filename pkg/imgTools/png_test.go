package imgTools

import (
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/lao-tseu-is-alive/go-wmts-tool/pkg/golog"
)

func TestCropThenSplit(t *testing.T) {
	l, err := golog.NewLogger("simple", os.Stderr, golog.ErrorLevel, "test:")
	if err != nil {
		t.Fatal(err)
	}
	const buffer, tileSize, n = 5, 8, 3
	size := tileSize*n + 2*buffer
	// each pixel encodes its own position, and some are semi-transparent
	src := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 200, A: uint8(100 + (x+y)%156)})
		}
	}

	cropped := CropImage(src, buffer, l)
	if got := cropped.Bounds().Size(); got != image.Pt(tileSize*n, tileSize*n) {
		t.Fatalf("cropped size: got %v, want %dx%d", got, tileSize*n, tileSize*n)
	}
	tiles, err := SplitImage(cropped, tileSize, tileSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(tiles) != n*n {
		t.Fatalf("got %d tiles, want %d", len(tiles), n*n)
	}
	for i, tile := range tiles {
		b := tile.Bounds()
		if b.Dx() != tileSize || b.Dy() != tileSize {
			t.Fatalf("tile %d size: got %v", i, b.Size())
		}
		row, col := i/n, i%n
		for dy := 0; dy < tileSize; dy++ {
			for dx := 0; dx < tileSize; dx++ {
				srcX, srcY := buffer+col*tileSize+dx, buffer+row*tileSize+dy
				want := src.NRGBAAt(srcX, srcY)
				got := color.NRGBAModel.Convert(tile.At(b.Min.X+dx, b.Min.Y+dy)).(color.NRGBA)
				if got != want {
					t.Fatalf("tile %d pixel (%d,%d): got %v, want %v (source pixel %d,%d)", i, dx, dy, got, want, srcX, srcY)
				}
			}
		}
	}
}
