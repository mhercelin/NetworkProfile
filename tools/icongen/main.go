//go:build ignore

// Command icongen draws the application icon and writes build/windows/icon.ico.
//
//	go run tools/icongen/main.go
//
// The icon is drawn rather than stored so it can be adjusted by editing numbers
// here. Shapes are filled rather than stroked: at 16 pixels, the size the
// notification area actually uses, an outlined glyph turns into grey mush.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
)

// Everything is laid out on a 256-unit square and scaled down per size.
const design = 256.0

// Supersampling gives the anti-aliasing; 4x4 per pixel is plenty at 16px.
const samples = 4

var (
	badgeTop    = color.RGBA{0x17, 0xA2, 0xAC, 0xFF}
	badgeBottom = color.RGBA{0x0A, 0x6F, 0x82, 0xFF}
	glyph       = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
)

type rect struct {
	x0, y0, x1, y1, r float64
}

func (s rect) contains(x, y float64) bool {
	dx := math.Max(math.Max(s.x0+s.r-x, 0), x-(s.x1-s.r))
	dy := math.Max(math.Max(s.y0+s.r-y, 0), y-(s.y1-s.r))
	return dx*dx+dy*dy <= s.r*s.r
}

var badge = rect{0, 0, 256, 256, 56}

// Three nodes joined by a bus: one above, two below, the shape of a small
// network. Stems overlap the bar they meet so no seam shows at any size.
var glyphShapes = []rect{
	{100, 40, 156, 96, 16},   // top node
	{26, 154, 82, 210, 16},   // bottom left node
	{174, 154, 230, 210, 16}, // bottom right node
	{120, 88, 136, 132, 8},   // stem down from the top node
	{46, 118, 210, 134, 8},   // the bus
	{46, 118, 62, 166, 8},    // drop to the left node
	{194, 118, 210, 166, 8},  // drop to the right node
}

func draw(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scale := design / float64(size)
	total := float64(samples * samples)

	for py := range size {
		for px := range size {
			var r, g, b, a float64

			for sy := range samples {
				for sx := range samples {
					x := (float64(px) + (float64(sx)+0.5)/samples) * scale
					y := (float64(py) + (float64(sy)+0.5)/samples) * scale
					if !badge.contains(x, y) {
						continue
					}

					c := gradientAt(y)
					for _, shape := range glyphShapes {
						if shape.contains(x, y) {
							c = glyph
							break
						}
					}
					r += float64(c.R)
					g += float64(c.G)
					b += float64(c.B)
					a++
				}
			}

			if a == 0 {
				continue
			}
			// Colour is averaged over the covered samples only, so edge pixels
			// keep the shape's colour and fade in alpha rather than to black.
			img.SetRGBA(px, py, color.RGBA{
				R: uint8(math.Round(r / a)),
				G: uint8(math.Round(g / a)),
				B: uint8(math.Round(b / a)),
				A: uint8(math.Round(a / total * 255)),
			})
		}
	}
	return img
}

func gradientAt(y float64) color.RGBA {
	t := y / design
	lerp := func(from, to uint8) uint8 {
		return uint8(math.Round(float64(from) + (float64(to)-float64(from))*t))
	}
	return color.RGBA{
		R: lerp(badgeTop.R, badgeBottom.R),
		G: lerp(badgeTop.G, badgeBottom.G),
		B: lerp(badgeTop.B, badgeBottom.B),
		A: 0xFF,
	}
}

// Above this size entries are stored as PNG, which keeps the file small; at or
// below it they are stored as plain DIBs, the form every version of Windows and
// every icon-loading path reads without question. Mixing the two this way is
// what icon editors produce.
const pngFrom = 64

// encodeDIB writes the BITMAPINFOHEADER form of an icon entry: a double-height
// header, bottom-up BGRA rows, and the legacy AND mask that must be present
// even when the alpha channel is what actually decides transparency.
func encodeDIB(img *image.RGBA) []byte {
	size := img.Bounds().Dx()
	var out bytes.Buffer

	binary.Write(&out, binary.LittleEndian, uint32(40))    // header size
	binary.Write(&out, binary.LittleEndian, int32(size))   // width
	binary.Write(&out, binary.LittleEndian, int32(size*2)) // height: XOR + AND
	binary.Write(&out, binary.LittleEndian, uint16(1))     // planes
	binary.Write(&out, binary.LittleEndian, uint16(32))    // bits per pixel
	binary.Write(&out, binary.LittleEndian, uint32(0))     // BI_RGB
	binary.Write(&out, binary.LittleEndian, uint32(size*size*4))
	for range 4 {
		binary.Write(&out, binary.LittleEndian, uint32(0)) // resolution, palette
	}

	for y := size - 1; y >= 0; y-- {
		for x := range size {
			c := img.RGBAAt(x, y)
			out.Write([]byte{c.B, c.G, c.R, c.A})
		}
	}

	maskRow := ((size + 31) / 32) * 4 // each row padded to 4 bytes
	out.Write(make([]byte, maskRow*size))

	return out.Bytes()
}

func writeICO(path string, sizes []int) error {
	var payloads [][]byte
	for _, size := range sizes {
		img := draw(size)
		if size < pngFrom {
			payloads = append(payloads, encodeDIB(img))
			continue
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		payloads = append(payloads, buf.Bytes())
	}

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint16(0)) // reserved
	binary.Write(&out, binary.LittleEndian, uint16(1)) // type: icon
	binary.Write(&out, binary.LittleEndian, uint16(len(sizes)))

	const headerSize = 6
	const entrySize = 16
	offset := headerSize + entrySize*len(sizes)

	for i, size := range sizes {
		// 256 is stored as 0: the field is a single byte.
		dimension := byte(size)
		out.WriteByte(dimension)
		out.WriteByte(dimension)
		out.WriteByte(0)                                    // palette size, none
		out.WriteByte(0)                                    // reserved
		binary.Write(&out, binary.LittleEndian, uint16(1))  // colour planes
		binary.Write(&out, binary.LittleEndian, uint16(32)) // bits per pixel
		binary.Write(&out, binary.LittleEndian, uint32(len(payloads[i])))
		binary.Write(&out, binary.LittleEndian, uint32(offset))
		offset += len(payloads[i])
	}
	for _, payload := range payloads {
		out.Write(payload)
	}

	return os.WriteFile(path, out.Bytes(), 0o644)
}

func main() {
	sizes := []int{16, 20, 24, 32, 40, 48, 64, 128, 256}
	path := filepath.Join("build", "windows", "icon.ico")

	if err := writeICO(path, sizes); err != nil {
		log.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s écrit : %d tailles, %d octets", path, len(sizes), info.Size())
}
