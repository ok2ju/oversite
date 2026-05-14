// Renders the Oversite aperture glyph into:
//   - build/appicon.png            (1024×1024, source for Wails build pipeline)
//   - build/darwin/iconfile.icns   (multi-resolution macOS icon)
//
// Mirrors the brand logo from frontend/src/components/brand/logo.tsx —
// a 5-blade aperture/iris pinwheel around a centered accent dot.
//
// Pure-stdlib rasterization — no external deps, no sips/iconutil.
// Each iconset slot is rendered at its native resolution so small icons
// (16×16) get crisp edges instead of downscaling artifacts.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

const (
	pngOutputPath  = "build/appicon.png"
	icnsOutputPath = "build/darwin/iconfile.icns"
)

var (
	bg     = color.NRGBA{R: 0x0B, G: 0x0D, B: 0x10, A: 0xFF}
	ink    = color.NRGBA{R: 0xE8, G: 0xEA, B: 0xED, A: 0xFF}
	accent = color.NRGBA{R: 0xE8, G: 0x9B, B: 0x2A, A: 0xFF}
)

func smoothstep(edge0, edge1, x float64) float64 {
	t := (x - edge0) / (edge1 - edge0)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

func diskCoverage(px, py, cx, cy, r float64) float64 {
	d := math.Hypot(px-cx, py-cy)
	return 1.0 - smoothstep(r-0.5, r+0.5, d)
}

func pointInTriangle(px, py, x0, y0, x1, y1, x2, y2 float64) bool {
	d1 := (px-x1)*(y0-y1) - (x0-x1)*(py-y1)
	d2 := (px-x2)*(y1-y2) - (x1-x2)*(py-y2)
	d3 := (px-x0)*(y2-y0) - (x2-x0)*(py-y0)
	hasNeg := d1 < 0 || d2 < 0 || d3 < 0
	hasPos := d1 > 0 || d2 > 0 || d3 > 0
	return !hasNeg || !hasPos
}

// 4×4 supersampling gives smooth blade edges without the cost of a full SDF.
func triangleCoverage(px, py, x0, y0, x1, y1, x2, y2 float64) float64 {
	const n = 4
	inside := 0
	for sy := 0; sy < n; sy++ {
		for sx := 0; sx < n; sx++ {
			qx := px - 0.5 + (float64(sx)+0.5)/float64(n)
			qy := py - 0.5 + (float64(sy)+0.5)/float64(n)
			if pointInTriangle(qx, qy, x0, y0, x1, y1, x2, y2) {
				inside++
			}
		}
	}
	return float64(inside) / float64(n*n)
}

func rotate(x, y, cx, cy, angleRad float64) (float64, float64) {
	s, c := math.Sin(angleRad), math.Cos(angleRad)
	dx, dy := x-cx, y-cy
	return cx + dx*c - dy*s, cy + dx*s + dy*c
}

func over(dst, src color.NRGBA, coverage float64) color.NRGBA {
	srcA := float64(src.A) / 255.0 * coverage
	if srcA <= 0 {
		return dst
	}
	dstA := float64(dst.A) / 255.0 * (1 - srcA)
	outA := srcA + dstA
	if outA <= 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8((float64(src.R)*srcA + float64(dst.R)*dstA) / outA),
		G: uint8((float64(src.G)*srcA + float64(dst.G)*dstA) / outA),
		B: uint8((float64(src.B)*srcA + float64(dst.B)*dstA) / outA),
		A: uint8(outA * 255),
	}
}

// renderAperture renders the 5-blade aperture glyph at any square resolution.
// Geometry mirrors the brand SVG's 0–64 viewBox; blade vertices are scaled
// to a centered inset square so the same logic works at 16px and 1024px.
func renderAperture(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	s := float64(size)
	inset := s * 0.76
	off := (s - inset) / 2.0
	cx := off + inset/2.0
	cy := off + inset/2.0

	scale := inset / 64.0
	toImg := func(x, y float64) (float64, float64) {
		return off + x*scale, off + y*scale
	}

	// Blade vertices from the brand SVG path "M 32 32 L 32 6 L 22 14 Z".
	bv0x, bv0y := toImg(32, 32)
	bv1x, bv1y := toImg(32, 6)
	bv2x, bv2y := toImg(22, 14)

	type tri struct{ ax, ay, bx, by, cx, cy float64 }
	blades := make([]tri, 5)
	for i := 0; i < 5; i++ {
		ang := float64(i) * 72.0 * math.Pi / 180.0
		b1x, b1y := rotate(bv1x, bv1y, cx, cy, ang)
		b2x, b2y := rotate(bv2x, bv2y, cx, cy, ang)
		blades[i] = tri{bv0x, bv0y, b1x, b1y, b2x, b2y}
	}

	dotR := inset * (4.5 / 64.0)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5

			pix := bg
			for _, t := range blades {
				pix = over(pix, ink, triangleCoverage(px, py, t.ax, t.ay, t.bx, t.by, t.cx, t.cy))
			}
			pix = over(pix, accent, diskCoverage(px, py, cx, cy, dotR))

			img.SetNRGBA(x, y, pix)
		}
	}
	return img
}

func encodePNG(img *image.NRGBA) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeICNS assembles a multi-resolution macOS .icns file. Each entry is a
// 4-byte type code, a big-endian uint32 length (header + payload), then the
// PNG data.
func writeICNS(path string, entries []icnsEntry) error {
	var body bytes.Buffer
	for _, e := range entries {
		body.WriteString(e.typ)
		_ = binary.Write(&body, binary.BigEndian, uint32(8+len(e.png)))
		body.Write(e.png)
	}

	var file bytes.Buffer
	file.WriteString("icns")
	_ = binary.Write(&file, binary.BigEndian, uint32(8+body.Len()))
	file.Write(body.Bytes())

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, file.Bytes(), 0o644)
}

type icnsEntry struct {
	typ string
	png []byte
}

// Resolution slots required for a complete macOS app icon. Each (size, type)
// pair is a distinct ICNS slot — same pixel size with different type codes
// signals retina vs. standard, which the OS uses to pick the right asset.
var slots = []struct {
	typ  string
	size int
}{
	{"icp4", 16},
	{"icp5", 32},
	{"ic07", 128},
	{"ic08", 256},
	{"ic09", 512},
	{"ic10", 1024}, // also serves 512@2x
	{"ic11", 32},   // 16@2x
	{"ic12", 64},   // 32@2x
	{"ic13", 256},  // 128@2x
	{"ic14", 512},  // 256@2x
}

func main() {
	cache := map[int][]byte{}
	pngFor := func(size int) ([]byte, error) {
		if data, ok := cache[size]; ok {
			return data, nil
		}
		img := renderAperture(size)
		data, err := encodePNG(img)
		if err != nil {
			return nil, err
		}
		cache[size] = data
		return data, nil
	}

	master, err := pngFor(1024)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render 1024: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(pngOutputPath, master, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", pngOutputPath, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (1024x1024)\n", pngOutputPath)

	entries := make([]icnsEntry, 0, len(slots))
	for _, s := range slots {
		data, err := pngFor(s.size)
		if err != nil {
			fmt.Fprintf(os.Stderr, "render %d: %v\n", s.size, err)
			os.Exit(1)
		}
		entries = append(entries, icnsEntry{typ: s.typ, png: data})
	}
	if err := writeICNS(icnsOutputPath, entries); err != nil {
		fmt.Fprintf(os.Stderr, "write icns: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d slots)\n", icnsOutputPath, len(slots))
}
