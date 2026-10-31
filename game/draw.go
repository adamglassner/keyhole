package game

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// S is the device scale factor: the UI is laid out in logical units and multiplied by S.
var S = 1.0

// useAA turns on anti-aliasing for shapes. High-density screens (most phones)
// don't need it, and skipping it keeps drawing fast.
var useAA = true

var whiteImg = func() *ebiten.Image {
	img := ebiten.NewImage(3, 3)
	img.Fill(color.White)
	return img
}()
var whiteSub = whiteImg.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

type Rect struct{ X, Y, W, H float64 }

func (r Rect) Contains(x, y float64) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}
func (r Rect) Inset(d float64) Rect     { return Rect{r.X + d, r.Y + d, r.W - 2*d, r.H - 2*d} }
func (r Rect) CX() float64              { return r.X + r.W/2 }
func (r Rect) CY() float64              { return r.Y + r.H/2 }
func (r Rect) Right() float64           { return r.X + r.W }
func (r Rect) Bottom() float64          { return r.Y + r.H }
func (r Rect) Move(dx, dy float64) Rect { return Rect{r.X + dx, r.Y + dy, r.W, r.H} }
func (r Rect) Scale(k float64) Rect {
	return Rect{r.CX() - r.W*k/2, r.CY() - r.H*k/2, r.W * k, r.H * k}
}

func rrPath(x, y, w, h, r float64) *vector.Path {
	p := &vector.Path{}
	r = math.Max(0, math.Min(r, math.Min(w, h)/2))
	fx, fy, fw, fh, fr := float32(x), float32(y), float32(w), float32(h), float32(r)
	if r <= 0.01 {
		p.MoveTo(fx, fy)
		p.LineTo(fx+fw, fy)
		p.LineTo(fx+fw, fy+fh)
		p.LineTo(fx, fy+fh)
		p.Close()
		return p
	}
	p.MoveTo(fx+fr, fy)
	p.LineTo(fx+fw-fr, fy)
	p.Arc(fx+fw-fr, fy+fr, fr, -math.Pi/2, 0, vector.Clockwise)
	p.LineTo(fx+fw, fy+fh-fr)
	p.Arc(fx+fw-fr, fy+fh-fr, fr, 0, math.Pi/2, vector.Clockwise)
	p.LineTo(fx+fr, fy+fh)
	p.Arc(fx+fr, fy+fh-fr, fr, math.Pi/2, math.Pi, vector.Clockwise)
	p.LineTo(fx, fy+fr)
	p.Arc(fx+fr, fy+fr, fr, math.Pi, 3*math.Pi/2, vector.Clockwise)
	p.Close()
	return p
}

func colorVerts(vs []ebiten.Vertex, c color.RGBA) {
	cr, cg, cb, ca := float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	for i := range vs {
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = cr, cg, cb, ca
	}
}

func fillPath(dst *ebiten.Image, p *vector.Path, c color.RGBA) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	colorVerts(vs, c)
	dst.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: useAA, FillRule: ebiten.FillRuleFillAll})
}

// fillPathV fills a path with a vertical gradient between top and bottom (device units).
func fillPathV(dst *ebiten.Image, p *vector.Path, top, bot color.RGBA, y0, y1 float64) {
	vs, is := p.AppendVerticesAndIndicesForFilling(nil, nil)
	for i := range vs {
		t := clamp((float64(vs[i].DstY)-y0)/math.Max(1, y1-y0), 0, 1)
		c := lerpColor(top, bot, t)
		vs[i].SrcX, vs[i].SrcY = 1, 1
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(c.R)/255, float32(c.G)/255, float32(c.B)/255, float32(c.A)/255
	}
	dst.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: useAA, FillRule: ebiten.FillRuleFillAll})
}

func strokePath(dst *ebiten.Image, p *vector.Path, width float64, c color.RGBA) {
	sop := &vector.StrokeOptions{Width: float32(width), LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound}
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, sop)
	colorVerts(vs, c)
	dst.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: useAA})
}

// ---- logical-unit helpers (multiplied by S)

func FillRR(dst *ebiten.Image, r Rect, rad float64, c color.RGBA) {
	fillPath(dst, rrPath(r.X*S, r.Y*S, r.W*S, r.H*S, rad*S), c)
}

func FillRRV(dst *ebiten.Image, r Rect, rad float64, top, bot color.RGBA) {
	fillPathV(dst, rrPath(r.X*S, r.Y*S, r.W*S, r.H*S, rad*S), top, bot, r.Y*S, r.Bottom()*S)
}

func StrokeRR(dst *ebiten.Image, r Rect, rad, width float64, c color.RGBA) {
	strokePath(dst, rrPath(r.X*S, r.Y*S, r.W*S, r.H*S, rad*S), width*S, c)
}

func FillRect(dst *ebiten.Image, r Rect, c color.RGBA) {
	vector.DrawFilledRect(dst, float32(r.X*S), float32(r.Y*S), float32(r.W*S), float32(r.H*S), c, false)
}

// GradientRect fills r with a vertical gradient.
func GradientRect(dst *ebiten.Image, r Rect, top, bot color.RGBA) {
	x0, y0, x1, y1 := float32(r.X*S), float32(r.Y*S), float32(r.Right()*S), float32(r.Bottom()*S)
	v := func(x, y float32, c color.RGBA) ebiten.Vertex {
		return ebiten.Vertex{DstX: x, DstY: y, SrcX: 1, SrcY: 1, ColorR: float32(c.R) / 255, ColorG: float32(c.G) / 255, ColorB: float32(c.B) / 255, ColorA: float32(c.A) / 255}
	}
	vs := []ebiten.Vertex{v(x0, y0, top), v(x1, y0, top), v(x1, y1, bot), v(x0, y1, bot)}
	dst.DrawTriangles(vs, []uint16{0, 1, 2, 0, 2, 3}, whiteSub, &ebiten.DrawTrianglesOptions{})
}

// RadialGlow draws a soft round glow (center color fading to transparent).
func RadialGlow(dst *ebiten.Image, cx, cy, r float64, c color.RGBA, strength float64) {
	const seg = 40
	vs := make([]ebiten.Vertex, 0, seg+2)
	is := make([]uint16, 0, seg*3)
	a := float32(clamp(strength, 0, 1))
	vs = append(vs, ebiten.Vertex{DstX: float32(cx * S), DstY: float32(cy * S), SrcX: 1, SrcY: 1,
		ColorR: float32(c.R) / 255 * a, ColorG: float32(c.G) / 255 * a, ColorB: float32(c.B) / 255 * a, ColorA: a})
	for i := 0; i <= seg; i++ {
		t := float64(i) / seg * 2 * math.Pi
		vs = append(vs, ebiten.Vertex{DstX: float32((cx + math.Cos(t)*r) * S), DstY: float32((cy + math.Sin(t)*r) * S), SrcX: 1, SrcY: 1})
		if i > 0 {
			is = append(is, 0, uint16(i), uint16(i+1))
		}
	}
	dst.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: useAA})
}

func FillCircle(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	vector.DrawFilledCircle(dst, float32(x*S), float32(y*S), float32(r*S), c, useAA)
}

func StrokeCircle(dst *ebiten.Image, x, y, r, w float64, c color.RGBA) {
	vector.StrokeCircle(dst, float32(x*S), float32(y*S), float32(r*S), float32(w*S), c, useAA)
}

func Line(dst *ebiten.Image, x0, y0, x1, y1, w float64, c color.RGBA) {
	vector.StrokeLine(dst, float32(x0*S), float32(y0*S), float32(x1*S), float32(y1*S), float32(w*S), c, useAA)
}

// Poly fills a polygon given in logical units.
func Poly(dst *ebiten.Image, c color.RGBA, pts ...float64) {
	p := &vector.Path{}
	for i := 0; i+1 < len(pts); i += 2 {
		if i == 0 {
			p.MoveTo(float32(pts[i]*S), float32(pts[i+1]*S))
		} else {
			p.LineTo(float32(pts[i]*S), float32(pts[i+1]*S))
		}
	}
	p.Close()
	fillPath(dst, p, c)
}

// Shadow draws a soft drop shadow under a rounded rect.
func Shadow(dst *ebiten.Image, r Rect, rad float64, strength float64) {
	for i := 4; i >= 1; i-- {
		d := float64(i) * 2.5
		FillRR(dst, Rect{r.X - d*0.5, r.Y + d*0.6, r.W + d, r.H + d}, rad+d, alpha(colBlack, 0.12*strength))
	}
}

// ---- text

type Weight int

const (
	Regular Weight = iota
	Medium
	Bold
)

var fontSources [3]*text.GoTextFaceSource

type faceKey struct {
	w    Weight
	size float64
	s    float64
}

var faceCache = map[faceKey]*text.GoTextFace{}

func Face(w Weight, size float64) *text.GoTextFace {
	k := faceKey{w, size, S}
	if f, ok := faceCache[k]; ok {
		return f
	}
	f := &text.GoTextFace{Source: fontSources[w], Size: size * S}
	faceCache[k] = f
	return f
}

type Align int

const (
	AlignCenter Align = iota
	AlignLeft
	AlignRight
)

// Text draws a single line. y is the vertical center of the line.
func Text(dst *ebiten.Image, s string, w Weight, size float64, x, y float64, c color.RGBA, al Align) {
	f := Face(w, size)
	op := &text.DrawOptions{}
	op.GeoM.Translate(x*S, y*S)
	switch al {
	case AlignCenter:
		op.PrimaryAlign = text.AlignCenter
	case AlignRight:
		op.PrimaryAlign = text.AlignEnd
	}
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, f, op)
}

// TextSpaced draws text with extra letter spacing (for titles).
func TextSpaced(dst *ebiten.Image, s string, w Weight, size, x, y, spacing float64, c color.RGBA, al Align) {
	total := SpacedWidth(s, w, size, spacing)
	switch al {
	case AlignCenter:
		x -= total / 2
	case AlignRight:
		x -= total
	}
	for _, r := range s {
		ch := string(r)
		Text(dst, ch, w, size, x, y, c, AlignLeft)
		x += TextWidth(ch, w, size) + spacing
	}
}

func SpacedWidth(s string, w Weight, size, spacing float64) float64 {
	t := 0.0
	n := 0
	for _, r := range s {
		t += TextWidth(string(r), w, size)
		n++
	}
	if n > 1 {
		t += spacing * float64(n-1)
	}
	return t
}

type measureKey struct {
	s    string
	w    Weight
	size float64
	sc   float64
}

type wrapKey struct {
	s          string
	w          Weight
	size, maxW float64
	sc         float64
}

var (
	widthCache = map[measureKey]float64{}
	wrapCache  = map[wrapKey][]string{}
)

// TextWidth measures a line of text. Results are cached: shaping text is slow.
func TextWidth(s string, w Weight, size float64) float64 {
	k := measureKey{s, w, size, S}
	if v, ok := widthCache[k]; ok {
		return v
	}
	if len(widthCache) > 4000 {
		widthCache = map[measureKey]float64{}
	}
	wd, _ := text.Measure(s, Face(w, size), 0)
	widthCache[k] = wd / S
	return wd / S
}

// WrapText splits s into lines that fit maxW (cached).
func WrapText(s string, w Weight, size, maxW float64) []string {
	k := wrapKey{s, w, size, maxW, S}
	if v, ok := wrapCache[k]; ok {
		return v
	}
	if len(wrapCache) > 500 {
		wrapCache = map[wrapKey][]string{}
	}
	v := wrapText(s, w, size, maxW)
	wrapCache[k] = v
	return v
}

func wrapText(s string, w Weight, size, maxW float64) []string {
	var lines []string
	for _, para := range splitLines(s) {
		cur := ""
		for _, wd := range splitWords(para) {
			try := wd
			if cur != "" {
				try = cur + " " + wd
			}
			if TextWidth(try, w, size) > maxW && cur != "" {
				lines = append(lines, cur)
				cur = wd
			} else {
				cur = try
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

func splitLines(s string) []string {
	var out []string
	cur := []rune{}
	for _, r := range s {
		if r == '\n' {
			out = append(out, string(cur))
			cur = cur[:0]
			continue
		}
		cur = append(cur, r)
	}
	return append(out, string(cur))
}

func splitWords(s string) []string {
	var out []string
	cur := []rune{}
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if len(cur) > 0 {
				out = append(out, string(cur))
				cur = cur[:0]
			}
			continue
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		out = append(out, string(cur))
	}
	return out
}

// TextWrapped draws wrapped text starting at y (top) and returns the height used.
func TextWrapped(dst *ebiten.Image, s string, w Weight, size, x, y, maxW, lineH float64, c color.RGBA, al Align) float64 {
	lines := WrapText(s, w, size, maxW)
	for i, l := range lines {
		lx := x
		if al == AlignCenter {
			lx = x + maxW/2
		}
		if dst != nil {
			Text(dst, l, w, size, lx, y+float64(i)*lineH+lineH/2, c, al)
		}
	}
	return float64(len(lines)) * lineH
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

func approach(cur, target, speed float64) float64 {
	return cur + (target-cur)*clamp(speed, 0, 1)
}

func easeOut(t float64) float64   { t = clamp(t, 0, 1); return 1 - (1-t)*(1-t)*(1-t) }
func easeInOut(t float64) float64 { t = clamp(t, 0, 1); return t * t * (3 - 2*t) }
func easeBack(t float64) float64 {
	t = clamp(t, 0, 1)
	const c1, c3 = 1.70158, 2.70158
	return 1 + c3*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}
