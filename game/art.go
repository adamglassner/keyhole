package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	bgImg         *ebiten.Image
	bgW, bgH, bgS float64
	bgStyle       *LockStyle
)

// drawBackground paints the steel-plate wall behind everything. It is rendered
// once into an image and reused, so each frame costs a single copy.
func drawBackground(dst *ebiten.Image) {
	st := style()
	if bgImg == nil || bgW != SW || bgH != SH || bgS != S || bgStyle != st {
		if bgImg != nil {
			bgImg.Deallocate()
		}
		bgImg = ebiten.NewImage(int(math.Ceil(SW*S)), int(math.Ceil(SH*S)))
		bgW, bgH, bgS, bgStyle = SW, SH, S, st
		paintBackground(bgImg, st)
	}
	dst.DrawImage(bgImg, nil)
}

func paintBackground(dst *ebiten.Image, st *LockStyle) {
	GradientRect(dst, Rect{0, 0, SW, SH}, st.BgTop, st.BgBot)
	RadialGlow(dst, SW/2, SH*0.38, math.Max(SW, SH)*0.6, lighten(st.BgTop, 0.06), 0.9)
	// plate seams and rivets
	seam := alpha(colWhite, 0.035)
	for _, fy := range []float64{0.22, 0.78} {
		y := math.Round(SH * fy)
		FillRect(dst, Rect{0, y, SW, 2}, seam)
		FillRect(dst, Rect{0, y + 2, SW, 1}, alpha(colBlack, 0.25))
	}
	if SW < 640 {
		return
	}
	for _, fx := range []float64{0.06, 0.94} {
		x := math.Round(SW * fx)
		FillRect(dst, Rect{x, 0, 2, SH}, seam)
	}
	for _, fx := range []float64{0.06, 0.94} {
		for _, fy := range []float64{0.22, 0.78} {
			drawRivet(dst, SW*fx+1, SH*fy+1, 4.5, st.Plate)
		}
	}
}

func drawRivet(dst *ebiten.Image, x, y, r float64, c color.RGBA) {
	FillCircle(dst, x+0.8, y+1.2, r, alpha(colBlack, 0.4))
	FillCircle(dst, x, y, r, lighten(c, 0.12))
	FillCircle(dst, x-r*0.3, y-r*0.3, r*0.35, alpha(colWhite, 0.25))
}

// drawDial draws the round vault door: a ticked metal ring, rivets and a glowing keyhole.
// rot spins the tick ring; glow (0..1) brightens the keyhole.
func drawDial(dst *ebiten.Image, cx, cy, r, rot, glow float64, st *LockStyle) {
	FillCircle(dst, cx, cy+r*0.04, r*1.03, alpha(colBlack, 0.45))
	FillCircle(dst, cx, cy, r, st.MetalLo)
	FillCircle(dst, cx-r*0.01, cy-r*0.015, r*0.985, st.Metal)
	// bevel highlight on the upper-left
	RadialGlow(dst, cx-r*0.35, cy-r*0.4, r*0.9, st.MetalHi, 0.35)
	// ticks
	for i := 0; i < 60; i++ {
		a := float64(i)*math.Pi/30 + rot
		r1 := r * 0.9
		w := 1.4
		if i%5 == 0 {
			r1, w = r*0.85, 2.6
		}
		Line(dst, cx+math.Cos(a)*r1, cy+math.Sin(a)*r1, cx+math.Cos(a)*r*0.97, cy+math.Sin(a)*r*0.97, w*r/130, alpha(st.MetalLo, 0.9))
	}
	// inner plate
	FillCircle(dst, cx, cy, r*0.82, darken(st.MetalLo, 0.3))
	FillCircle(dst, cx, cy, r*0.79, st.Plate)
	RadialGlow(dst, cx-r*0.2, cy-r*0.25, r*0.8, lighten(st.Plate, 0.25), 0.6)
	for i := 0; i < 12; i++ {
		a := float64(i)*math.Pi/6 + math.Pi/12
		drawRivet(dst, cx+math.Cos(a)*r*0.68, cy+math.Sin(a)*r*0.68, r*0.04, st.Metal)
	}
	StrokeCircle(dst, cx, cy, r*0.47, r*0.03, alpha(st.Metal, 0.75))
	// keyhole light
	RadialGlow(dst, cx, cy, r*0.75, st.MetalHi, 0.25+0.5*glow)
	drawKeyhole(dst, cx, cy-r*0.05, r*0.22, lerpColor(hex("#FFD98A"), hex("#FFF4D6"), glow))
}

// drawKeyhole draws a keyhole shape of height ~2.4*s centered at (cx, cy).
func drawKeyhole(dst *ebiten.Image, cx, cy, s float64, light color.RGBA) {
	dark := hex("#120A02")
	FillCircle(dst, cx, cy-s*0.45, s*0.68, dark)
	Poly(dst, dark, cx-s*0.36, cy, cx+s*0.36, cy, cx+s*0.62, cy+s*1.45, cx-s*0.62, cy+s*1.45)
	FillCircle(dst, cx, cy-s*0.45, s*0.48, light)
	Poly(dst, light, cx-s*0.22, cy-s*0.1, cx+s*0.22, cy-s*0.1, cx+s*0.4, cy+s*1.25, cx-s*0.4, cy+s*1.25)
	FillCircle(dst, cx, cy-s*0.55, s*0.2, alpha(colWhite, 0.7))
}

// drawPadlock draws a padlock about s tall. open (0..1) lifts and swings the shackle.
func drawPadlock(dst *ebiten.Image, cx, cy, s float64, body, dark color.RGBA, open float64, hole bool) {
	bw, bh := s*0.78, s*0.58
	by := cy - bh*0.1
	lift := s * 0.22 * easeOut(open)
	sw := s * 0.48
	thick := s * 0.11
	// shackle: an upside-down U
	p := &vector.Path{}
	x0, x1 := (cx-sw/2)*S, (cx+sw/2)*S
	top := (by - s*0.3 - lift) * S
	p.MoveTo(float32(x0), float32((by+bh*0.1-lift*0.2)*S))
	p.LineTo(float32(x0), float32(top+sw/2*S))
	p.Arc(float32(cx*S), float32(top+sw/2*S), float32(sw/2*S), math.Pi, 2*math.Pi, vector.Clockwise)
	rightBottom := by + bh*0.1
	if open > 0 {
		rightBottom = by - s*0.05 - lift
	}
	p.LineTo(float32(x1), float32(rightBottom*S))
	sop := &vector.StrokeOptions{Width: float32(thick * S), LineCap: vector.LineCapRound, LineJoin: vector.LineJoinRound}
	vs, is := p.AppendVerticesAndIndicesForStroke(nil, nil, sop)
	colorVerts(vs, lighten(dark, 0.25))
	dst.DrawTriangles(vs, is, whiteSub, &ebiten.DrawTrianglesOptions{AntiAlias: useAA})
	// body
	r := Rect{cx - bw/2, by, bw, bh}
	FillRR(dst, r.Move(0, s*0.03), s*0.1, alpha(colBlack, 0.35))
	FillRRV(dst, r, s*0.1, lighten(body, 0.2), darken(body, 0.12))
	FillRR(dst, Rect{r.X + bw*0.1, r.Y + bh*0.1, bw * 0.8, bh * 0.14}, s*0.04, alpha(colWhite, 0.22))
	if hole {
		drawKeyhole(dst, cx, by+bh*0.45, s*0.09, dark)
	}
}

// ---- particles

type particle struct {
	x, y, vx, vy, life, max, size float64
	c                             color.RGBA
}

type particles []*particle

func (ps *particles) burst(x, y float64, n int, speed float64, c color.RGBA) {
	for i := 0; i < n; i++ {
		a := rand.Float64() * 2 * math.Pi
		v := speed * (0.35 + 0.65*rand.Float64())
		life := 40 + 50*rand.Float64()
		*ps = append(*ps, &particle{x, y, math.Cos(a) * v, math.Sin(a)*v - speed*0.4, life, life, 2 + 3*rand.Float64(), c})
	}
}

func (ps *particles) update() {
	out := (*ps)[:0]
	for _, p := range *ps {
		p.x += p.vx
		p.y += p.vy
		p.vy += 0.12
		p.vx *= 0.985
		p.life--
		if p.life > 0 {
			out = append(out, p)
		}
	}
	*ps = out
}

func (ps particles) draw(dst *ebiten.Image) {
	for _, p := range ps {
		a := clamp(p.life/p.max*1.4, 0, 1)
		FillCircle(dst, p.x, p.y, p.size*(0.5+0.5*a), alpha(p.c, a))
	}
}
