package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input merges mouse, touch and keyboard into one simple per-tick snapshot.
type Input struct {
	MX, MY            float64 // logical
	Pressed, Released bool
	Down              bool
	Touch             bool // the current pointer is a finger (no hover)
	Wheel             float64
	DragDY            float64 // vertical drag since last tick while Down
	Chars             []rune  // letters typed this tick (lowercase a-z)
	Enter, Backspace  bool
	Back              bool // Esc, or Android back
	consumed          bool
	Tick              int64
	touchID           ebiten.TouchID
	touching          bool
}

var in Input

// pendingBack is set by the Android back button.
var pendingBack bool

func repeatPressed(k ebiten.Key) bool {
	d := inpututil.KeyPressDuration(k)
	return d == 1 || (d > 24 && d%4 == 0)
}

func (i *Input) poll() {
	i.Tick++
	i.consumed = false
	i.Pressed, i.Released = false, false
	prevY := i.MY

	// touch
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		if !i.touching {
			i.touching, i.touchID = true, id
			x, y := ebiten.TouchPosition(id)
			i.MX, i.MY = float64(x)/S, float64(y)/S
			prevY = i.MY
			i.Pressed, i.Touch = true, true
		}
	}
	if i.touching {
		if inpututil.IsTouchJustReleased(i.touchID) {
			x, y := inpututil.TouchPositionInPreviousTick(i.touchID)
			i.MX, i.MY = float64(x)/S, float64(y)/S
			i.Released, i.touching = true, false
		} else {
			x, y := ebiten.TouchPosition(i.touchID)
			i.MX, i.MY = float64(x)/S, float64(y)/S
		}
		i.Down = i.touching
	} else if !i.Touch || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || mouseMoved() {
		x, y := ebiten.CursorPosition()
		i.MX, i.MY = float64(x)/S, float64(y)/S
		i.Pressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		i.Released = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
		i.Down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
		if i.Pressed {
			prevY = i.MY
		}
		i.Touch = false
	}
	i.DragDY = 0
	if i.Down && !i.Pressed {
		i.DragDY = i.MY - prevY
	}
	_, wy := ebiten.Wheel()
	i.Wheel = wy

	// keyboard
	i.Chars = i.Chars[:0]
	for _, r := range ebiten.AppendInputChars(nil) {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		if r >= 'a' && r <= 'z' {
			i.Chars = append(i.Chars, r)
		}
	}
	i.Enter = inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)
	i.Backspace = repeatPressed(ebiten.KeyBackspace)
	i.Back = inpututil.IsKeyJustPressed(ebiten.KeyEscape) || pendingBack
	pendingBack = false
}

var lastCX, lastCY int

func mouseMoved() bool {
	x, y := ebiten.CursorPosition()
	m := x != lastCX || y != lastCY
	lastCX, lastCY = x, y
	return m
}

// Click returns true once if a press happened inside r and wasn't consumed.
func (i *Input) Click(r Rect) bool {
	if i.Pressed && !i.consumed && r.Contains(i.MX, i.MY) {
		i.consumed = true
		return true
	}
	return false
}

// Consume swallows the current press so nothing underneath reacts to it.
func (i *Input) Consume() { i.consumed = true }

func (i *Input) Hover(r Rect) bool { return !i.Touch && r.Contains(i.MX, i.MY) }

var hoverAnim = map[string]float64{}

func animHover(id string, hovering bool) float64 {
	v := hoverAnim[id]
	t := 0.0
	if hovering {
		t = 1
	}
	v = approach(v, t, 0.25)
	hoverAnim[id] = v
	return v
}
