package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFailedFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed.csv")
	want := []metaTileTask{
		{zoomLevel: 9, startCol: 9224, startRow: 15436, size: 4},
		{zoomLevel: 8, startCol: 4512, startRow: 7590, size: 2},
	}
	if err := writeFailedFile(path, "fonds_geo_osm_bdcad_gris", want); err != nil {
		t.Fatalf("writeFailedFile: %v", err)
	}
	got, err := readFailedFile(path, "fonds_geo_osm_bdcad_gris")
	if err != nil {
		t.Fatalf("readFailedFile: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if _, err := readFailedFile(path, "another_layer"); err == nil {
		t.Errorf("expected an error when the layer does not match")
	}
}

func TestReadFailedFileRejectsInvalidContent(t *testing.T) {
	for name, content := range map[string]string{
		"missing column":    "layer,9,1,2\n",
		"not a number":      "layer,9,x,2,4\n",
		"negative value":    "layer,9,-1,2,4\n",
		"zero metatilesize": "layer,9,1,2,0\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "failed.csv")
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}
			if _, err := readFailedFile(path, "layer"); err == nil {
				t.Errorf("expected an error for %q", content)
			}
		})
	}
}

func TestPrintSummary(t *testing.T) {
	res := passResult{
		saved:        10,
		skipped:      3,
		tilesWritten: 160,
		tilesSkipped: 48,
		failed:       []metaTileTask{{zoomLevel: 9, startCol: 9192, startRow: 15432, size: 4}},
	}
	var buf bytes.Buffer
	printSummary(&buf, "zoom 9", 14, 224, res)
	for _, want := range []string{
		"zoom 9: 14 meta-tiles, 224 png expected",
		"saved   :     10 meta-tiles,      160 png written",
		"skipped :      3 meta-tiles,       48 png fresh",
		"failed  :      1 meta-tiles,       16 png missing",
	} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("summary does not contain %q:\n%s", want, buf.String())
		}
	}
}
