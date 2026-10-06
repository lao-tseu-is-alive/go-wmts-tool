package main

import (
	"os"
	"path/filepath"
	"reflect"
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
