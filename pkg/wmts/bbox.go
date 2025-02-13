package wmts

import "fmt"

// BBox represents a bounding box.
type BBox struct {
	XMin float64
	YMin float64
	XMax float64
	YMax float64
}

// NewBBoxFromArray creates a new BBox from an array of 4 float64 values. XMin, YMin, XMax, YMax
func NewBBoxFromArray(arr []float64) (*BBox, error) {
	if len(arr) != 4 {
		return nil, fmt.Errorf("invalid array length. Expected 4, got %d", len(arr))
	}
	if arr[0] > arr[2] || arr[1] > arr[3] {
		return nil, fmt.Errorf("invalid bounding box values. XMin must be less than XMax and YMin must be less than YMax")
	}
	return &BBox{XMin: arr[0], YMin: arr[1], XMax: arr[2], YMax: arr[3]}, nil
}

// NewBBox creates a new BBox from the given values.
func NewBBox(xMin, yMin, xMax, yMax float64) (*BBox, error) {
	if xMin > xMax || yMin > yMax {
		return nil, fmt.Errorf("invalid bounding box values. XMin must be less than XMax and YMin must be less than YMax")
	}
	return &BBox{XMin: xMin, YMin: yMin, XMax: xMax, YMax: yMax}, nil
}

// String returns a string representation of the BBox.
func (b *BBox) String() string {
	return fmt.Sprintf("%f,%f,%f,%f", b.XMin, b.YMin, b.XMax, b.YMax)
}

// ToArray returns the BBox as an array of 4 float64 values.
func (b *BBox) ToArray() []float64 {
	return []float64{b.XMin, b.YMin, b.XMax, b.YMax}
}

// Width returns the width of the BBox.
func (b *BBox) Width() float64 {
	return b.XMax - b.XMin
}

// Height returns the height of the BBox.
func (b *BBox) Height() float64 {
	return b.YMax - b.YMin
}

// Area returns the area of the BBox
func (b *BBox) Area() float64 {
	return b.Width() * b.Height()
}

// Intersects checks if two bounding boxes overlap.
func (b *BBox) Intersects(other BBox) bool {
	return !(b.XMax < other.XMin || b.XMin > other.XMax ||
		b.YMax < other.YMin || b.YMin > other.YMax)
}

// Contains checks if the bounding box 'b' completely contains the 'other' bounding box.
func (b *BBox) Contains(other BBox) bool {
	return b.XMin <= other.XMin && b.XMax >= other.XMax && b.YMin <= other.YMin && b.YMax >= other.YMax
}

// Expand expands the bounding box by a given amount in all directions.
func (b *BBox) Expand(amount float64) {
	b.XMin -= amount
	b.YMin -= amount
	b.XMax += amount
	b.YMax += amount
}
