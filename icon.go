package main

// Iconițele aplicației (PWA), desenate la build: „Go” cu fontul Go Bold pe
// fundal turcoaz, deasupra a patru puncte în culorile celor patru niveluri.

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var (
	iconTop    = color.RGBA{0x00, 0xAD, 0xD8, 0xff} // turcoazul Go
	iconBottom = color.RGBA{0x00, 0x7D, 0x9C, 0xff}
	// aceleași culori ca --lvl-1 … --lvl-4 din style.css (tema întunecată, mai vii pe turcoaz)
	iconLevels = []color.RGBA{
		{0x4a, 0xde, 0x80, 0xff},
		{0xfb, 0xbf, 0x24, 0xff},
		{0xc4, 0xa1, 0xff, 0xff},
		{0xfb, 0x71, 0x85, 0xff},
	}
)

var goBold = func() *opentype.Font {
	f, err := opentype.Parse(gobold.TTF)
	if err != nil {
		panic(err)
	}
	return f
}()

// renderIcon desenează o iconiță pătrată de size×size pixeli. Pentru
// variantele „maskable”, conținutul e micșorat în zona sigură (cercul central
// de 80%), pe care sistemele de operare nu o taie niciodată.
func renderIcon(size int, maskable bool) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		t := float64(y) / float64(size-1)
		c := color.RGBA{
			R: lerp(iconTop.R, iconBottom.R, t),
			G: lerp(iconTop.G, iconBottom.G, t),
			B: lerp(iconTop.B, iconBottom.B, t),
			A: 0xff,
		}
		draw.Draw(img, image.Rect(0, y, size, y+1), image.NewUniform(c), image.Point{}, draw.Src)
	}

	scale := 1.0
	if maskable {
		scale = 0.78
	}
	s := float64(size) * scale

	face, err := opentype.NewFace(goBold, &opentype.FaceOptions{Size: s * 0.46, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	defer face.Close()

	const text = "Go"
	d := &font.Drawer{Dst: img, Src: image.White, Face: face}
	width := d.MeasureString(text)
	bounds, _ := font.BoundString(face, text)
	height := bounds.Max.Y - bounds.Min.Y
	// textul e centrat orizontal și puțin deasupra centrului, ca să facă loc punctelor
	top := fixed.I(size)/2 - height/2 - fixed.Int26_6(s*0.07*64)
	d.Dot = fixed.Point26_6{X: (fixed.I(size) - width) / 2, Y: top - bounds.Min.Y}
	d.DrawString(text)

	r := s * 0.045
	gap := s * 0.135
	cy := float64(size)/2 + s*0.25
	for i, c := range iconLevels {
		cx := float64(size)/2 + (float64(i)-1.5)*gap
		fillCircle(img, cx, cy, r, c)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func lerp(a, b uint8, t float64) uint8 {
	return uint8(math.Round(float64(a) + (float64(b)-float64(a))*t))
}

// fillCircle desenează un cerc plin, cu margini netezite (anti-aliasing).
func fillCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			dist := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			cover := math.Max(0, math.Min(1, r-dist+0.5))
			if cover == 0 {
				continue
			}
			bg := img.RGBAAt(x, y)
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(c.R)*cover + float64(bg.R)*(1-cover)),
				G: uint8(float64(c.G)*cover + float64(bg.G)*(1-cover)),
				B: uint8(float64(c.B)*cover + float64(bg.B)*(1-cover)),
				A: 0xff,
			})
		}
	}
}
