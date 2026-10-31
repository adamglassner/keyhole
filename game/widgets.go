package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type ButtonKind int

const (
	BtnGold ButtonKind = iota
	BtnDark
	BtnGhost
)

type Button struct {
	ID       string
	R        Rect
	Label    string
	Sub      string
	Kind     ButtonKind
	Size     float64
	Icon     func(dst *ebiten.Image, cx, cy, s float64, c color.RGBA)
	Disabled bool
	hover    float64
	press    float64
}

// Update returns true when the button was clicked.
func (b *Button) Update() bool {
	h := !b.Disabled && in.Hover(b.R)
	b.hover = animHover(b.ID, h)
	if h {
		ebiten.SetCursorShape(ebiten.CursorShapePointer)
	}
	b.press = approach(b.press, 0, 0.2)
	if !b.Disabled && in.Click(b.R) {
		b.press = 1
		playSFX("button")
		return true
	}
	return false
}

func (b *Button) Draw(dst *ebiten.Image) {
	r := b.R.Scale(1 - 0.03*b.press)
	size := b.Size
	if size == 0 {
		size = 17
	}
	rad := math.Min(14, r.H/2)
	st := style()
	var fg, subc color.RGBA
	switch b.Kind {
	case BtnGold:
		Shadow(dst, r, rad, 1)
		top, bot := lighten(st.Metal, 0.25+0.1*b.hover), darken(st.Metal, 0.15)
		if b.Disabled {
			top, bot = colSurf2, colSurface
		}
		FillRRV(dst, r, rad, top, bot)
		StrokeRR(dst, r, rad, 1.5, alpha(darken(st.Metal, 0.5), 0.9))
		fg, subc = colInk, alpha(colInk, 0.72)
	case BtnDark:
		FillRR(dst, r, rad, lerpColor(colSurface, colSurf2, b.hover))
		StrokeRR(dst, r, rad, 1.5, lerpColor(colBorder, st.Metal, b.hover*0.8))
		fg, subc = colText, colTextDim
	case BtnGhost:
		if b.hover > 0.01 {
			FillRR(dst, r, rad, alpha(colWhite, 0.06*b.hover))
		}
		fg, subc = lerpColor(colTextDim, colText, b.hover), colTextDim
	}
	if b.Disabled {
		fg = alpha(fg, 0.45)
	}
	cx := r.CX()
	tw := TextWidth(b.Label, Bold, size)
	if b.Icon != nil {
		iconS := size * 1.15
		if b.Label == "" {
			b.Icon(dst, cx, r.CY(), iconS, fg)
			return
		}
		iy := r.CY()
		if b.Sub != "" {
			iy = r.CY() - size*0.45
		}
		ix := cx - (tw+iconS+8)/2 + iconS/2
		b.Icon(dst, ix, iy, iconS, fg)
		cx = ix + iconS/2 + 8 + tw/2
	}
	if b.Sub != "" {
		Text(dst, b.Label, Bold, size, cx, r.CY()-size*0.45, fg, AlignCenter)
		Text(dst, b.Sub, Regular, size*0.72, r.CX(), r.CY()+size*0.68, subc, AlignCenter)
	} else {
		Text(dst, b.Label, Bold, size, cx, r.CY()+1, fg, AlignCenter)
	}
}

// Slider is a 0..1 volume slider.
type Slider struct {
	R        Rect
	dragging bool
}

func (s *Slider) Update(v *float64) bool {
	hit := Rect{s.R.X - 8, s.R.Y - 12, s.R.W + 16, s.R.H + 24}
	if in.Click(hit) {
		s.dragging = true
	}
	if s.dragging {
		if !in.Down {
			s.dragging = false
			playSFX("button")
		}
		*v = clamp((in.MX-s.R.X)/s.R.W, 0, 1)
		return true
	}
	return false
}

func (s *Slider) Draw(dst *ebiten.Image, v float64, muted bool) {
	st := style()
	FillRR(dst, Rect{s.R.X, s.R.CY() - 3, s.R.W, 6}, 3, colBorder)
	c := st.Metal
	if muted {
		c = colTextDim
	}
	FillRR(dst, Rect{s.R.X, s.R.CY() - 3, s.R.W * v, 6}, 3, c)
	FillCircle(dst, s.R.X+s.R.W*v, s.R.CY(), 10, lighten(c, 0.3))
	StrokeCircle(dst, s.R.X+s.R.W*v, s.R.CY(), 10, 1.5, darken(c, 0.4))
}

// Toggle draws an on/off switch and returns true when tapped.
func Toggle(dst *ebiten.Image, id string, r Rect, on bool) bool {
	clicked := false
	if dst == nil {
		if in.Click(r.Inset(-8)) {
			playSFX("button")
			clicked = true
		}
		return clicked
	}
	t := animHover("tg"+id, on)
	FillRR(dst, r, r.H/2, lerpColor(colBorder, style().Metal, t))
	x := r.X + r.H/2 + (r.W-r.H)*t
	FillCircle(dst, x, r.CY(), r.H/2-3, colText)
	return false
}

// ---- icons, drawn with vectors so they stay sharp at any size

func IconBack(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	w := s * 0.11
	Line(dst, cx+s*0.18, cy-s*0.32, cx-s*0.16, cy, w, c)
	Line(dst, cx-s*0.16, cy, cx+s*0.18, cy+s*0.32, w, c)
}

func IconClose(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	d := s * 0.28
	Line(dst, cx-d, cy-d, cx+d, cy+d, s*0.1, c)
	Line(dst, cx+d, cy-d, cx-d, cy+d, s*0.1, c)
}

func IconHelp(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	StrokeCircle(dst, cx, cy, s*0.42, s*0.08, c)
	Text(dst, "?", Bold, s*0.62, cx, cy+s*0.02, c, AlignCenter)
}

func IconStats(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	w := s * 0.18
	for i, h := range []float64{0.4, 0.75, 0.55} {
		x := cx - s*0.3 + float64(i)*s*0.3
		FillRR(dst, Rect{x - w/2, cy + s*0.36 - s*h, w, s * h}, w*0.3, c)
	}
}

func IconGear(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	for i := 0; i < 8; i++ {
		a := float64(i) * math.Pi / 4
		Line(dst, cx+math.Cos(a)*s*0.22, cy+math.Sin(a)*s*0.22, cx+math.Cos(a)*s*0.42, cy+math.Sin(a)*s*0.42, s*0.16, c)
	}
	FillCircle(dst, cx, cy, s*0.3, c)
	FillCircle(dst, cx, cy, s*0.12, colSurface)
}

func IconShare(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	w := s * 0.09
	Line(dst, cx, cy+s*0.12, cx, cy-s*0.38, w, c)
	Line(dst, cx-s*0.17, cy-s*0.22, cx, cy-s*0.38, w, c)
	Line(dst, cx+s*0.17, cy-s*0.22, cx, cy-s*0.38, w, c)
	Line(dst, cx-s*0.3, cy-s*0.05, cx-s*0.3, cy+s*0.38, w, c)
	Line(dst, cx-s*0.3, cy+s*0.38, cx+s*0.3, cy+s*0.38, w, c)
	Line(dst, cx+s*0.3, cy+s*0.38, cx+s*0.3, cy-s*0.05, w, c)
}

func IconKey(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	StrokeCircle(dst, cx-s*0.22, cy, s*0.17, s*0.1, c)
	Line(dst, cx-s*0.05, cy, cx+s*0.42, cy, s*0.1, c)
	Line(dst, cx+s*0.28, cy, cx+s*0.28, cy+s*0.16, s*0.09, c)
	Line(dst, cx+s*0.4, cy, cx+s*0.4, cy+s*0.2, s*0.09, c)
}

func IconFlame(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	Poly(dst, c, cx, cy-s*0.45, cx+s*0.3, cy, cx+s*0.25, cy+s*0.3, cx, cy+s*0.42, cx-s*0.25, cy+s*0.3, cx-s*0.3, cy+s*0.02, cx-s*0.12, cy-s*0.1)
	FillCircle(dst, cx, cy+s*0.2, s*0.12, darken(c, 0.45))
}

func IconPlay(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	Poly(dst, c, cx-s*0.22, cy-s*0.32, cx+s*0.3, cy, cx-s*0.22, cy+s*0.32)
}

func IconDice(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	r := Rect{cx - s*0.36, cy - s*0.36, s * 0.72, s * 0.72}
	StrokeRR(dst, r, s*0.12, s*0.08, c)
	for _, p := range [][2]float64{{-0.17, -0.17}, {0, 0}, {0.17, 0.17}} {
		FillCircle(dst, cx+p[0]*s, cy+p[1]*s, s*0.065, c)
	}
}

func IconZen(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	// a calm lotus: three petals over a line
	Poly(dst, c, cx, cy-s*0.38, cx+s*0.13, cy, cx, cy+s*0.14, cx-s*0.13, cy)
	Poly(dst, c, cx-s*0.42, cy-s*0.12, cx-s*0.06, cy+s*0.02, cx-s*0.08, cy+s*0.16, cx-s*0.36, cy+s*0.08)
	Poly(dst, c, cx+s*0.42, cy-s*0.12, cx+s*0.06, cy+s*0.02, cx+s*0.08, cy+s*0.16, cx+s*0.36, cy+s*0.08)
	Line(dst, cx-s*0.4, cy+s*0.3, cx+s*0.4, cy+s*0.3, s*0.08, c)
}

func IconTheme(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	// a tag
	Poly(dst, c, cx-s*0.38, cy-s*0.3, cx+s*0.12, cy-s*0.3, cx+s*0.42, cy, cx+s*0.12, cy+s*0.3, cx-s*0.38, cy+s*0.3)
	FillCircle(dst, cx-s*0.16, cy, s*0.08, colSurface)
}

func IconLock(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	drawPadlock(dst, cx, cy+s*0.08, s*0.75, c, darken(c, 0.5), 0, false)
}
