package engine

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf8"
)

// Chixie 3.0.224's A1 uploader sends UTF-8 text as page_<n>.gcode via
// FTPS STOR. It does not pack a 3MF or declare slicer/filament metadata.
func TestHandwritingExportUsesPlainGCodeWithoutFilamentPreparation(t *testing.T) {
	page := Page{Number: 4, Strokes: []Stroke{{{X: 20, Y: 30}, {X: 70, Y: 80}}}}
	doc := &Document{Settings: DefaultLayout(), Pages: []Page{page}}
	for _, mode := range []string{"manual", "auto"} {
		t.Run(mode, func(t *testing.T) {
			writing := testWriting()
			writing.OriginMode = mode
			data, err := ExportPage(doc, 0, writing)
			if err != nil {
				t.Fatal(err)
			}
			if !utf8.Valid(data) || bytes.HasPrefix(data, []byte("PK")) || !bytes.HasPrefix(data, []byte("; A1 handwriting / page 4\n")) {
				t.Fatal("A1 upload must be a plain UTF-8 G-code file")
			}
			if strings.Contains(strings.ToLower(string(data)), "filament") || bytes.Contains(data, []byte("HEADER_BLOCK_START")) {
				t.Fatal("writing file declares a slicer/filament job")
			}
			for _, line := range strings.Split(string(data), "\n") {
				fields := strings.Fields(strings.SplitN(line, ";", 2)[0])
				if len(fields) == 0 {
					continue
				}
				command := fields[0]
				if strings.HasPrefix(command, "T") || strings.HasPrefix(command, "M620") || strings.HasPrefix(command, "M621") || command == "M109" || command == "M190" || command == "G28" {
					t.Fatalf("unexpected printer preparation: %s", line)
				}
				for _, field := range fields[1:] {
					if (command == "G0" || command == "G1") && strings.HasPrefix(field, "E") {
						t.Fatalf("unexpected extrusion: %s", line)
					}
					if (command == "M104" || command == "M140") && strings.HasPrefix(field, "S") && field != "S0" {
						t.Fatalf("unexpected heating: %s", line)
					}
				}
			}
			if !bytes.Contains(data, []byte("G1 Z7.000")) || !bytes.Contains(data, []byte("G1 Z3.400")) || !bytes.Contains(data, []byte("G1 X")) {
				t.Fatal("lost calibrated pen motion")
			}
		})
	}
}

func TestAllPagesZIPContainsStandaloneGCodeAndSkipsBlankPages(t *testing.T) {
	doc := &Document{Settings: DefaultLayout(), Pages: []Page{
		{Number: 1, Strokes: []Stroke{{{X: 20, Y: 30}, {X: 40, Y: 50}}}},
		{Number: 2},
		{Number: 3, Strokes: []Stroke{{{X: 60, Y: 70}, {X: 80, Y: 90}}}},
	}}
	data, err := ExportAll(doc, testWriting())
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 2 {
		t.Fatal("blank page was not skipped")
	}
	for ordinal, index := range []int{0, 2} {
		file := archive.File[ordinal]
		if file.Name != fmt.Sprintf("page-%03d.gcode", index+1) {
			t.Fatal("lost source page number or wrong file format", file.Name)
		}
		r, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			t.Fatal(err)
		}
		want, err := ExportPage(doc, index, testWriting())
		if err != nil || !bytes.Equal(body, want) {
			t.Fatal("batch export changed the page's G-code", err)
		}
	}
}
