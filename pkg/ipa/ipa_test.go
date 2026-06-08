package ipa

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/iineva/ipa-server/pkg/seekbuf"
)

func TestReadPlistInfo(t *testing.T) {

	printMemUsage()

	fileName := "test_data/ipa.ipa"
	// fileName := "/Users/steven/Downloads/TikTok (18.5.0) Unicorn v4.9.ipa"
	f, err := os.Open(fileName)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}

	buf, err := seekbuf.Open(f, seekbuf.MemoryMode)
	if err != nil {
		t.Fatal(err)
	}
	defer buf.Close()

	info, err := Parse(buf, fi.Size())
	if err != nil {
		t.Fatal(err)
	}
	if info == nil {
		t.Fatal(errors.New("parse error"))
	}
	printMemUsage()
	// log.Printf("%+v", info)
}

func TestIconSize(t *testing.T) {

	data := map[string]int{
		"Payload/UnicornApp.app/AppIcon_TikTok29x29@3x.png":          87,
		"Payload/UnicornApp.app/AppIcon_TikTok40x40@2x.png":          80,
		"Payload/UnicornApp.app/AppIcon_TikTok60x60@3x.png":          180,
		"Payload/UnicornApp.app/AppIcon_TikTok60x60@2x.png":          120,
		"Payload/UnicornApp.app/AppIcon_TikTok40x40@3x.png":          120,
		"Payload/UnicornApp.app/AppIcon_TikTok29x29@2x.png":          58,
		"Payload/UnicornApp.app/AppIcon_TikTok83.5x83.5@2x~ipad.png": 167,
		"Payload/UnicornApp.app/AppIcon_TikTok20x20@3x.png":          60,
		"Payload/UnicornApp.app/AppIcon_TikTok76x76~ipad.png":        76,
		"Payload/UnicornApp.app/AppIcon_TikTok20x20@2x.png":          40,
		"Payload/UnicornApp.app/AppIcon_TikTok76x76@2x~ipad.png":     152,
	}
	for k, v := range data {
		size, err := iconSize(k)
		if err != nil {
			t.Fatal(err)
		}
		if size != v {
			t.Fatal(errors.New("size error"))
		}
	}
}

func printMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// TestParseCgBIIcon guards against the regression where png.Decode advances the
// buffer and the CgBI fallback (ipaPng.Decode) is then handed a non-zero offset,
// failing with "not a PNG file" and forcing the expensive Assets.car fallback.
func TestParseCgBIIcon(t *testing.T) {
	data, err := os.ReadFile("test_data/cgbi_icon.png")
	if err != nil {
		t.Fatal(err)
	}

	// wrap the CgBI png in a zip so we can get a *zip.File
	zbuf := &bytes.Buffer{}
	zw := zip.NewWriter(zbuf)
	w, err := zw.Create("Payload/App.app/AppIcon60x60@2x.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(zbuf.Bytes()), int64(zbuf.Len()))
	if err != nil {
		t.Fatal(err)
	}

	img, err := parseIconImage(zr.File[0])
	if err != nil {
		t.Fatalf("parseIconImage failed on CgBI png: %v", err)
	}
	if img == nil {
		t.Fatal("expected decoded icon, got nil")
	}
	if b := img.Bounds(); b.Dx() != 120 || b.Dy() != 120 {
		t.Fatalf("unexpected icon bounds: %v", b)
	}
}
