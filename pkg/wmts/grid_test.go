package wmts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsMetaTileFresh(t *testing.T) {
	basePath := t.TempDir()
	lc := LayerConfig{Name: "test_layer"}
	lc.WMTSURLPrefix, lc.WMTSURLStyle, lc.WMTSDimensionYear, lc.WMTSMatrixSet = "1.0.0", "default", "2021", "swissgrid_05"
	g := &Grid{}
	const zoom, startCol, startRow, size = 9, 9224, 15436, 2
	maxAge := 24 * time.Hour

	touch := func(row, col int, modTime time.Time) {
		t.Helper()
		path := lc.TileImgPath(basePath, zoom, row, col)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("png"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}

	if g.IsMetaTileFresh(zoom, startCol, startRow, size, size, lc, basePath, maxAge) {
		t.Errorf("empty cache must not be fresh")
	}
	now := time.Now()
	touch(startRow, startCol, now)
	touch(startRow, startCol+1, now)
	touch(startRow+1, startCol, now)
	if g.IsMetaTileFresh(zoom, startCol, startRow, size, size, lc, basePath, maxAge) {
		t.Errorf("meta-tile with a missing tile must not be fresh")
	}
	touch(startRow+1, startCol+1, now.Add(-30*24*time.Hour))
	if g.IsMetaTileFresh(zoom, startCol, startRow, size, size, lc, basePath, maxAge) {
		t.Errorf("meta-tile with an outdated tile must not be fresh")
	}
	touch(startRow+1, startCol+1, now.Add(-time.Hour))
	if !g.IsMetaTileFresh(zoom, startCol, startRow, size, size, lc, basePath, maxAge) {
		t.Errorf("meta-tile with all tiles recent must be fresh")
	}
}
