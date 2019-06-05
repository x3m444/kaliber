/*
   Copyright © 2019 M.Watermann, 10247 Berlin, Germany
                  All rights reserved
               EMail : <support@mwat.de>
*/

package kaliber

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestTDocument_Cover(t *testing.T) {
	SetCalibreLibraryPath("/var/opt/Calibre/")
	d1 := TDocument{
		ID:   7628,
		path: calibreLibraryPath + "/Spiegel/Der Spiegel (2019-06-01) 23_2019 (7628)",
	}
	w1 := "/cover/7628/cover.jpg"
	d2 := TDocument{
		ID:   6730,
		path: calibreLibraryPath + "/John Scalzi/Zoe's Tale (6730)",
	}
	w2 := "/cover/6730/cover.jpg"
	tests := []struct {
		name   string
		fields TDocument
		want   string
	}{
		// TODO: Add test cases.
		{" 1", d1, w1},
		{" 2", d2, w2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &tt.fields
			if got := doc.Cover(); got != tt.want {
				t.Errorf("TDocument.Cover() = '%s',\nwant '%s'", got, tt.want)
			}
		})
	}
} // TestTDocument_Cover()

func TestTDocument_coverAbs(t *testing.T) {
	SetCalibreLibraryPath("/var/opt/Calibre/")
	d1 := TDocument{
		ID:   7628,
		path: "Spiegel/Der Spiegel (2019-06-01) 23_2019 (7628)",
	}
	w1 := d1.path + "/cover.jpg"

	d2 := TDocument{
		ID:   6730,
		path: "John Scalzi/Zoe's Tale (6730)",
	}
	w2 := d2.path + "/cover.jpg"
	w3 := filepath.Join(calibreLibraryPath, w1)
	w4 := filepath.Join(calibreLibraryPath, w2)
	type args struct {
		aRelative bool
	}
	tests := []struct {
		name    string
		fields  TDocument
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{" 1", d1, args{true}, w1, false},
		{" 2", d2, args{true}, w2, false},
		{" 3", d1, args{false}, w3, false},
		{" 4", d2, args{false}, w4, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &tt.fields
			got, err := doc.coverAbs(tt.args.aRelative)
			if (err != nil) != tt.wantErr {
				t.Errorf("TDocument.coverAbs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TDocument.coverAbs() = '%s',\nwant '%s'", got, tt.want)
			}
		})
	}
} // TestTDocument_coverAbs()

func TestTDocument_Filename(t *testing.T) {
	SetCalibreLibraryPath("/var/opt/Calibre/")
	d1 := TDocument{
		formats: &tFormatList{
			TEntity{
				Name: "AZW3",
			},
			TEntity{
				Name: "EPUB",
			},
			TEntity{
				Name: "PDF",
			},
		},
		path: "John Scalzi/Zoe's Tale (6730)",
	}
	w1 := filepath.Join(d1.path, "Zoe's Tale - John Scalzi.azw3")
	type args struct {
		aFormat   string
		aRelative bool
	}
	tests := []struct {
		name   string
		fields TDocument
		args   args
		want   string
	}{
		// TODO: Add test cases.
		{" 1", d1, args{"azw3", true}, w1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &tt.fields
			if got := doc.Filename(tt.args.aFormat, tt.args.aRelative); got != tt.want {
				t.Errorf("TDocument.Filename() = '%s',\nwant '%s'", got, tt.want)
			}
		})
	}
} // TestTDocument_Filename

func TestTDocument_Filenames(t *testing.T) {
	SetCalibreLibraryPath("/var/opt/Calibre/")
	d1 := TDocument{
		formats: &tFormatList{
			TEntity{
				Name: "PDF",
			},
		},
		path: "Spiegel/Der Spiegel (2019-06-01) 23_2019 (7628)",
	}
	w1 := &TPathList{
		"PDF": "/var/opt/Calibre/Spiegel/Der Spiegel (2019-06-01) 23_2019 (7628)/Der Spiegel (2019-06-01) 23_2019 - Spiegel.pdf",
	}
	d2 := TDocument{
		formats: &tFormatList{
			TEntity{
				Name: "AZW3",
			},
			TEntity{
				Name: "EPUB",
			},
			TEntity{
				Name: "PDF",
			},
		},
		path: "John Scalzi/Zoe's Tale (6730)",
	}
	w2 := &TPathList{
		"AZW3": "/var/opt/Calibre/John Scalzi/Zoe's Tale (6730)/Zoe's Tale - John Scalzi.azw3",
		"EPUB": "/var/opt/Calibre/John Scalzi/Zoe's Tale (6730)/Zoe's Tale - John Scalzi.epub",
		"PDF":  "/var/opt/Calibre/John Scalzi/Zoe's Tale (6730)/Zoe's Tale - John Scalzi.pdf",
	}
	tests := []struct {
		name   string
		fields TDocument
		want   *TPathList
	}{
		// TODO: Add test cases.
		{" 1", d1, w1},
		{" 2", d2, w2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &tt.fields
			if got := doc.Filenames(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TDocument.Filenames() = %v,\nwant %v", got, tt.want)
			}
		})
	}
} // TestTDocument_Filenames()

func TestTDocument_Formats(t *testing.T) {
	SetCalibreLibraryPath("/var/opt/Calibre/")
	d1 := TDocument{
		ID: 1,
		formats: &tFormatList{
			TEntity{
				ID:   2,
				Name: "PDF",
			},
		},
		path: "Spiegel/Der Spiegel (2019-06-01) 23_2019 (7628)",
	}
	w1 := &TEntityList{
		TEntity{
			ID:   2,
			Name: "PDF",
			URL:  "/format/1/PDF",
		},
	}
	tests := []struct {
		name   string
		fields TDocument
		want   *TEntityList
	}{
		// TODO: Add test cases.
		{" 1", d1, w1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &tt.fields
			if got := doc.Formats(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TDocument.Formats() = %v,\nwant %v", got, tt.want)
			}
		})
	}
} // TestTDocument_Formats()

func TestTDocument_setPages(t *testing.T) {
	SetCalibreLibraryPath("/var/opt/Calibre/")
	d1 := TDocument{
		path: "Spiegel/Der Spiegel (2019-06-01) 23_2019 (7628)",
	}
	d2 := TDocument{
		path: "John Scalzi/Zoe's Tale (6730)",
	}
	tests := []struct {
		name   string
		fields TDocument
		want   int
	}{
		// TODO: Add test cases.
		{" 1", d1, 130},
		{" 2", d2, 569},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &tt.fields
			if got := doc.setPages(); got != tt.want {
				t.Errorf("TDocument.setPages() = %v, want %v", got, tt.want)
			}
		})
	}
} // TestTDocument_setPages()