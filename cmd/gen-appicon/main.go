// Renders the Oversite reticle into:
//   - build/appicon.png            (1024×1024, source for Wails build pipeline)
//   - build/darwin/iconfile.icns   (multi-resolution macOS icon)
//
// Pure-stdlib SDF rasterization — no external deps, no sips/iconutil.
// Each iconset slot is rendered at its native resolution so small icons
// (16×16) get crisp strokes instead of downscaling artifacts.
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
	accent = color.NRGBA{R: 0xF1, G: 0x8A, B: 0x4B, A: 0xFF}
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

func ringCoverage(px, py, cx, cy, r, sw float64) float64 {
	d := math.Hypot(px-cx, py-cy)
	half := sw / 2.0
	return 1.0 - smoothstep(half-0.5, half+0.5, math.Abs(d-r))
}

func rectCoverage(px, py, x0, y0, x1, y1 float64) float64 {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	xCov := smoothstep(x0-0.5, x0+0.5, px) - smoothstep(x1-0.5, x1+0.5, px)
	yCov := smoothstep(y0-0.5, y0+0.5, py) - smoothstep(y1-0.5, y1+0.5, py)
	return xCov * yCov
}

func vlineCoverage(px, py, x, y0, y1, sw float64) float64 {
	half := sw / 2.0
	return rectCoverage(px, py, x-half, y0, x+half, y1)
}

func hlineCoverage(px, py, y, x0, x1, sw float64) float64 {
	half := sw / 2.0
	return rectCoverage(px, py, x0, y-half, x1, y+half)
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

// renderReticle renders the reticle at any square resolution. Geometry is
// expressed as fractions of size, so the same SDF works at 16px and 1024px.
func renderReticle(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	s := float64(size)
	inset := s * 0.75
	off := (s - inset) / 2.0

	stroke := s * 0.039
	if stroke < 1.6 {
		stroke = 1.6
	}
	innerStroke := stroke * 0.7

	cx := off + inset/2.0
	cy := off + inset/2.0
	rOuter := (inset-stroke)/2.0 - inset*0.04
	rInner := rOuter * 0.42
	tickEnd := inset * 0.22
	dotR := inset * 0.08

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5

			pix := bg
			pix = over(pix, ink, ringCoverage(px, py, cx, cy, rOuter, stroke))
			pix = over(pix, ink, ringCoverage(px, py, cx, cy, rInner, innerStroke)*0.55)
			pix = over(pix, ink, vlineCoverage(px, py, cx, off, off+tickEnd, stroke))
			pix = over(pix, ink, vlineCoverage(px, py, cx, off+inset-tickEnd, off+inset, stroke))
			pix = over(pix, ink, hlineCoverage(px, py, cy, off, off+tickEnd, stroke))
			pix = over(pix, ink, hlineCoverage(px, py, cy, off+inset-tickEnd, off+inset, stroke))
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
		img := renderReticle(size)
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
