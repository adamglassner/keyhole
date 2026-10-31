package game

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	phInput = iota
	phReveal
	phUnlock
	phSlide
	phVault
	phSealed
	phDone
)

const (
	flipTicks    = 16.0
	staggerTicks = 10.0
)

type playScene struct {
	v       *Vault
	input   []byte
	phase   int
	pt      float64 // ticks in the current phase
	t       float64
	shakeT  float64
	pop     [6]float64
	lockA   [3]float64 // lock badge open animation
	parts   particles
	sounded map[int]bool

	back, help, stats, seeResult, toMenu Button
	keys                                 []kbKey
	pinRects                             []Rect
	pinLetters                           []byte

	// layout
	colX, colW float64
	topY       float64
	stripR     Rect
	meterR     Rect
	pinR       Rect
	boardR     Rect
	tile, gap  float64
	rows       int
	kbY, keyH  float64

	earnedKey bool
	newStyle  string
	finished  bool
	doorImg   *ebiten.Image
}

type kbKey struct {
	r     Rect
	label string
	c     byte // 0 = enter, 1 = backspace
	flash float64
}

func newPlayScene(v *Vault) *playScene {
	p := &playScene{v: v, sounded: map[int]bool{}}
	p.back = Button{ID: "p-back", Kind: BtnGhost, Icon: IconBack, Size: 22}
	p.help = Button{ID: "p-help", Kind: BtnGhost, Icon: IconHelp, Size: 22}
	p.stats = Button{ID: "p-stats", Kind: BtnGhost, Icon: IconStats, Size: 22}
	p.seeResult = Button{ID: "p-res", Kind: BtnGold, Label: "See result", Size: 18}
	p.toMenu = Button{ID: "p-menu", Kind: BtnDark, Label: "Menu", Size: 16}
	for i := 0; i < 3; i++ {
		if v.Locks[i].Opened {
			p.lockA[i] = 1
		}
	}
	if v.Done {
		p.phase = phDone
		p.finished = true
	} else if v.Cur > 0 && len(v.Lock().Guesses) == 0 {
		p.phase, p.pt = phSlide, 0
	}
	musicHold = false
	return p
}

func (p *playScene) title() string {
	if p.v.Kind == Daily {
		return "Vault #" + itoa(p.v.Number)
	}
	return "Practice Vault"
}

// ---- layout

func (p *playScene) layout() {
	pad := 12.0
	p.colW = math.Min(SW-2*pad, 520)
	p.colX = SW/2 - p.colW/2
	y := 6.0
	p.topY = y
	p.back.R = Rect{p.colX - 4, y, 46, 46}
	p.stats.R = Rect{p.colX + p.colW - 42, y, 46, 46}
	p.help.R = Rect{p.colX + p.colW - 88, y, 46, 46}
	y += 50
	p.stripR = Rect{p.colX, y, p.colW, 62}
	y += 66
	p.meterR = Rect{p.colX, y, p.colW, 26}
	y += 32
	p.pinR = Rect{p.colX, y, p.colW, 50}
	y += 58

	p.keyH = clamp(SH*0.064, 44, 58)
	kgap := 6.0
	kbH := 3*p.keyH + 2*kgap
	p.kbY = SH - kbH - 12
	p.layoutKeys(kgap)

	L := p.v.WordLen()
	boardTop, boardBot := y, p.kbY-12
	boardH := boardBot - boardTop
	p.gap = 6
	for p.rows = 6; p.rows >= 3; p.rows-- {
		p.tile = math.Min(64, math.Min((p.colW-5*p.gap)/6, (boardH-float64(p.rows-1)*p.gap)/float64(p.rows)))
		if p.tile >= 40 {
			break
		}
	}
	w := float64(L)*p.tile + float64(L-1)*p.gap
	h := float64(p.rows)*p.tile + float64(p.rows-1)*p.gap
	p.boardR = Rect{SW/2 - w/2, boardTop + math.Max(0, (boardH-h)/2), w, h}
}

func (p *playScene) layoutKeys(g float64) {
	rows := []string{"qwertyuiop", "asdfghjkl", "+zxcvbnm-"}
	kw := (p.colW - 9*g) / 10
	if len(p.keys) == 0 {
		for _, r := range rows {
			for i := 0; i < len(r); i++ {
				k := kbKey{label: strings.ToUpper(string(r[i])), c: r[i]}
				switch r[i] {
				case '+':
					k.label, k.c = "ENTER", 0
				case '-':
					k.label, k.c = "", 1
				}
				p.keys = append(p.keys, k)
			}
		}
	}
	idx := 0
	for ri, r := range rows {
		y := p.kbY + float64(ri)*(p.keyH+g)
		widths := make([]float64, len(r))
		total := 0.0
		for i := 0; i < len(r); i++ {
			widths[i] = kw
			if r[i] == '+' || r[i] == '-' {
				widths[i] = kw*1.5 + g*0.5
			}
			total += widths[i]
		}
		total += float64(len(r)-1) * g
		x := SW/2 - total/2
		for i := 0; i < len(r); i++ {
			p.keys[idx].r = Rect{x, y, widths[i], p.keyH}
			x += widths[i] + g
			idx++
		}
	}
}

// ---- update

func (p *playScene) Update() {
	p.t++
	p.pt++
	p.layout()
	for i := range p.pop {
		p.pop[i] = approach(p.pop[i], 0, 0.25)
	}
	for i := range p.keys {
		p.keys[i].flash = approach(p.keys[i].flash, 0, 0.2)
	}
	if p.shakeT > 0 {
		p.shakeT--
	}
	for i := 0; i < 3; i++ {
		if p.v.Locks[i].Opened && (i < p.v.Cur || p.phase != phReveal) {
			p.lockA[i] = approach(p.lockA[i], 1, 0.08)
		}
	}
	p.parts.update()

	if p.back.Update() || in.Back {
		writeSave()
		game.SetScene(newMenuScene())
		return
	}
	if p.help.Update() {
		game.Push(newHelpPanel())
	}
	if p.stats.Update() {
		game.Push(newStatsPanel())
	}

	switch p.phase {
	case phInput:
		p.updateInput()
	case phReveal:
		p.updateReveal()
	case phUnlock:
		if p.pt == 1 {
			playSFX("unlock")
			cx := p.stripR.X + p.stripR.W*(float64(p.v.Cur)+0.5)/3
			p.parts.burst(cx, p.stripR.CY(), 26, 4, style().MetalHi)
		}
		if p.pt > 70 {
			if p.v.Done {
				p.phase, p.pt = phVault, 0
			} else {
				p.v.Advance()
				writeSave()
				p.input = p.input[:0]
				p.phase, p.pt = phSlide, 0
				if p.v.Lock().PinTests > 1 {
					game.Toast("Clean open! 2 free pin tests")
				}
			}
		}
	case phSlide:
		if p.pt > 26 {
			p.phase, p.pt = phInput, 0
		}
		p.updatePins()
	case phVault:
		p.updateVault()
	case phSealed:
		if p.pt == 1 {
			playSFX("fail")
		}
		if p.pt > 110 {
			p.finish()
		}
	case phDone:
		if p.seeResult.Update() {
			game.Push(newResultPanel(p))
		}
		if p.toMenu.Update() {
			game.SetScene(newMenuScene())
		}
	}
}

func (p *playScene) updateInput() {
	L := p.v.WordLen()
	p.updatePins()
	typeChar := func(c byte) {
		if len(p.input) < L {
			p.input = append(p.input, c)
			p.pop[len(p.input)-1] = 1
			playSFX("type")
		}
	}
	for i := range p.keys {
		k := &p.keys[i]
		if in.Click(k.r) {
			k.flash = 1
			switch k.c {
			case 0:
				p.submit()
			case 1:
				p.backspace()
			default:
				typeChar(k.c)
			}
		}
	}
	for _, r := range in.Chars {
		typeChar(byte(r))
		p.flashKey(byte(r))
	}
	if in.Backspace {
		p.backspace()
		p.flashKey(1)
	}
	if in.Enter {
		p.flashKey(0)
		p.submit()
	}
}

func (p *playScene) flashKey(c byte) {
	for i := range p.keys {
		if p.keys[i].c == c {
			p.keys[i].flash = 1
		}
	}
}

func (p *playScene) backspace() {
	if len(p.input) > 0 {
		p.input = p.input[:len(p.input)-1]
		playSFX("erase")
	}
}

func (p *playScene) submit() {
	L := p.v.WordLen()
	res, _ := p.v.Guess(string(p.input), isValid)
	switch res {
	case GuessTooShort:
		p.reject("Needs " + itoa(L) + " letters")
	case GuessNotWord:
		p.reject("Not in the word list")
	case GuessRepeat:
		p.reject("You already tried that")
	case GuessOK:
		p.input = p.input[:0]
		p.phase, p.pt = phReveal, 0
		p.sounded = map[int]bool{}
		writeSave()
	}
}

func (p *playScene) reject(msg string) {
	p.shakeT = 22
	playSFX("invalid")
	game.Toast(msg)
}

func (p *playScene) updatePins() {
	p.pinLetters = p.v.CarriedPins()
	if p.v.PinTestsLeft() == 0 {
		return
	}
	for i, r := range p.pinRects {
		if i < len(p.pinLetters) && in.Click(r) {
			res, ok := p.v.TestPin(p.pinLetters[i])
			if ok {
				writeSave()
				if res.Regions == 0 {
					playSFX("silent")
				} else {
					playSFX("pin")
				}
				game.Toast(strings.ToUpper(string(res.Letter)) + " is " + res.Describe())
			}
			return
		}
	}
}

func (p *playScene) revealProgress(i int) float64 {
	return clamp((p.pt-float64(i)*staggerTicks)/flipTicks, 0, 1)
}

func (p *playScene) updateReveal() {
	l := p.v.Lock()
	row := len(l.Guesses) - 1
	marks := l.Marks(row)
	for i := range marks {
		if p.revealProgress(i) >= 0.5 && !p.sounded[i] {
			p.sounded[i] = true
			switch marks[i] {
			case Click:
				playSFX("click")
			case Wiggle:
				playSFX("wiggle")
			default:
				playSFX("silent")
			}
		}
	}
	end := float64(len(marks)-1)*staggerTicks + flipTicks + 8
	if p.pt >= end {
		switch {
		case l.Opened:
			p.phase, p.pt = phUnlock, 0
			if !p.v.Done {
				game.Toast("Lock " + itoa(p.v.Cur+1) + " opened!")
			}
		case p.v.Done:
			p.phase, p.pt = phSealed, 0
			game.Toast("Out of guesses. The vault stays sealed")
		default:
			p.phase, p.pt = phInput, 0
			if p.v.Left() <= 3 {
				playSFX("warn")
			}
		}
	}
}

func (p *playScene) updateVault() {
	if p.pt < 72 && int(p.pt)%12 == 0 {
		playSFXv("tick", 0.8)
	}
	if p.pt == 74 {
		playSFX("vault")
	}
	if p.pt == 96 {
		r := math.Min(SW, SH) * 0.3
		st := style()
		p.parts.burst(SW/2, SH/2, 70, 7, st.MetalHi)
		p.parts.burst(SW/2, SH/2, 40, 5, colGoldHi)
		_ = r
	}
	if p.pt > 175 || (p.pt > 100 && in.Pressed) {
		p.finish()
	}
}

func (p *playScene) finish() {
	if p.finished {
		return
	}
	p.finished = true
	before := save.Stats.GoldKeys
	p.earnedKey = recordFinish(p.v)
	for _, s := range lockStyles {
		if s.Keys > before && s.Keys <= save.Stats.GoldKeys {
			p.newStyle = s.Name
		}
	}
	if p.earnedKey {
		playSFX("key")
	}
	p.phase, p.pt = phDone, 0
	game.Push(newResultPanel(p))
}

// ---- draw

func (p *playScene) Draw(dst *ebiten.Image) {
	p.layout()
	drawBackground(dst)
	st := style()
	mp := marksPalette()

	// top bar
	p.back.Draw(dst)
	p.help.Draw(dst)
	p.stats.Draw(dst)
	Text(dst, p.title(), Bold, 18, SW/2, p.topY+23, colText, AlignCenter)

	p.drawStrip(dst, st)
	p.drawMeter(dst, st)
	p.drawPins(dst, st, mp)
	p.drawBoard(dst, st, mp)

	if p.phase == phDone {
		bw := math.Min(360, p.colW)
		p.seeResult.R = Rect{SW/2 - bw/2, p.kbY + 4, bw, 58}
		p.toMenu.R = Rect{SW/2 - bw/2, p.kbY + 4 + 58 + 12, bw, 48}
		p.seeResult.Draw(dst)
		p.toMenu.Draw(dst)
	} else {
		p.drawKeyboard(dst, st, mp)
	}
	if p.phase == phVault {
		p.drawVault(dst, st)
	}
	p.parts.draw(dst)
}

func (p *playScene) drawStrip(dst *ebiten.Image, st *LockStyle) {
	w := p.stripR.W / 3
	for i := 0; i < 3; i++ {
		cx := p.stripR.X + w*(float64(i)+0.5)
		cy := p.stripR.Y + 22
		l := &p.v.Locks[i]
		cur := i == p.v.Cur && !p.v.Done
		a := p.lockA[i]
		body := lerpColor(hex("#3A4648"), st.Metal, a)
		if cur {
			pulse := 0.5 + 0.5*math.Sin(p.t/14)
			RadialGlow(dst, cx, cy, 34, st.MetalHi, 0.18+0.14*pulse)
			body = lerpColor(lerpColor(hex("#55666A"), st.Metal, 0.45), st.Metal, a)
		}
		if p.v.Done && !p.v.Opened && i == p.v.Cur {
			body = darken(colDanger, 0.25)
		}
		bounce := 0.0
		if p.phase == phUnlock && i == p.v.Cur {
			bounce = math.Sin(clamp(p.pt/22, 0, 1)*math.Pi) * 6
		}
		drawPadlock(dst, cx, cy-bounce, 34, body, darken(body, 0.55), a, true)
		label := itoa(len(l.Answer)) + " letters"
		col := colTextDim
		if l.Opened {
			label = plural(len(l.Guesses), "guess", "guesses")
			col = st.MetalHi
		} else if cur {
			col = colText
		}
		Text(dst, label, Medium, 12, cx, p.stripR.Y+52, col, AlignCenter)
		if i < 2 {
			x := p.stripR.X + w*float64(i+1)
			for d := -1.0; d <= 1; d++ {
				FillCircle(dst, x+d*7, cy+2, 1.6, alpha(colTextDim, 0.5))
			}
		}
	}
}

func (p *playScene) drawMeter(dst *ebiten.Image, st *LockStyle) {
	r := p.meterR
	left := p.v.Left()
	label := plural(left, "guess left", "guesses left")
	lc := colText
	if left <= 3 && !p.v.Done {
		lc = lerpColor(colDanger, colText, 0.5+0.5*math.Sin(p.t/8))
	}
	Text(dst, label, Bold, 14, r.X, r.CY(), lc, AlignLeft)
	px := r.X + 120
	pw := (r.Right() - px - float64(TotalGuesses-1)*4) / TotalGuesses
	for i := 0; i < TotalGuesses; i++ {
		pr := Rect{px + float64(i)*(pw+4), r.Y + 4, pw, r.H - 8}
		if i < left {
			c := st.Metal
			if left <= 3 {
				c = colDanger
			}
			FillRRV(dst, pr, 3, lighten(c, 0.25), darken(c, 0.15))
		} else {
			FillRR(dst, pr, 3, alpha(colBlack, 0.35))
			StrokeRR(dst, pr, 3, 1, alpha(colBorder, 0.8))
		}
	}
}

func (p *playScene) drawPins(dst *ebiten.Image, st *LockStyle, mp markColors) {
	r := p.pinR
	FillRR(dst, r, 12, alpha(colBlack, 0.28))
	StrokeRR(dst, r, 12, 1, alpha(colBorder, 0.7))
	p.pinRects = p.pinRects[:0]
	if p.v.Cur == 0 && !(p.v.Done && p.v.Opened) {
		Text(dst, "Open this lock and its letters become pins", Regular, 13, r.CX(), r.CY(), colTextDim, AlignCenter)
		return
	}
	if p.v.Done && p.v.Opened {
		Text(dst, "All three locks open", Medium, 14, r.CX(), r.CY(), st.MetalHi, AlignCenter)
		return
	}
	TextSpaced(dst, "PINS", Bold, 11, r.X+14, r.CY(), 2, colTextDim, AlignLeft)
	l := p.v.Lock()
	tested := map[byte]PinResult{}
	for _, pr := range l.Pins {
		tested[pr.Letter] = pr
	}
	// every letter of the last answer, tested ones first
	prev := p.v.Locks[p.v.Cur-1].Answer
	var letters []byte
	for _, pr := range l.Pins {
		letters = append(letters, pr.Letter)
	}
	for _, c := range p.v.CarriedPins() {
		letters = append(letters, c)
	}
	_ = prev
	canTest := p.v.PinTestsLeft() > 0
	cs := 34.0
	x := r.X + 62
	for _, c := range letters {
		cr := Rect{x, r.CY() - cs/2, cs, cs}
		res, done := tested[c]
		ch := strings.ToUpper(string(c))
		switch {
		case done && res.Regions != 0:
			FillRRV(dst, cr, 8, lighten(mp.Click, 0.15), darken(mp.Click, 0.1))
			Text(dst, ch, Bold, 16, cr.CX(), cr.CY()-3, mp.ClickText, AlignCenter)
			for k, bit := range []int{RegionFront, RegionMiddle, RegionBack} {
				c := alpha(mp.ClickText, 0.25)
				if res.Regions&bit != 0 {
					c = mp.ClickText
				}
				FillRR(dst, Rect{cr.X + 6 + float64(k)*8, cr.Bottom() - 7, 6, 3}, 1.5, c)
			}
		case done:
			FillRR(dst, cr, 8, mp.Silent)
			Text(dst, ch, Bold, 16, cr.CX(), cr.CY(), alpha(mp.SilentText, 0.6), AlignCenter)
			Line(dst, cr.X+7, cr.Bottom()-7, cr.Right()-7, cr.Y+7, 2, alpha(colDanger, 0.9))
		case canTest:
			pulse := 0.5 + 0.5*math.Sin(p.t/10)
			h := animHover("pin"+ch, in.Hover(cr))
			FillRR(dst, cr, 8, lerpColor(colSurface, colSurf2, h))
			StrokeRR(dst, cr, 8, 2, lerpColor(st.MetalLo, st.MetalHi, pulse*0.6+h*0.4))
			Text(dst, ch, Bold, 16, cr.CX(), cr.CY(), colText, AlignCenter)
			p.pinRects = append(p.pinRects, cr)
		default:
			FillRR(dst, cr, 8, alpha(colBlack, 0.2))
			StrokeRR(dst, cr, 8, 1, colBorder)
			Text(dst, ch, Bold, 16, cr.CX(), cr.CY(), alpha(colTextDim, 0.7), AlignCenter)
		}
		x += cs + 6
	}
	// hint on the right if there's room
	hint := ""
	switch {
	case canTest:
		hint = "Tap to test · " + itoa(p.v.PinTestsLeft()) + " free"
	case len(l.Pins) > 0:
		last := l.Pins[len(l.Pins)-1]
		hint = strings.ToUpper(string(last.Letter)) + ": " + last.Describe()
	case len(l.Guesses) > 0:
		hint = "Tests end at your first guess"
	}
	if hint != "" {
		if x+TextWidth(hint, Regular, 12)+14 < r.Right() {
			Text(dst, hint, Regular, 12, r.Right()-12, r.CY(), colTextDim, AlignRight)
		}
	}
}

type tileState int

const (
	tEmpty tileState = iota
	tTyped
	tMarked
	tAnswer
)

func drawTile(dst *ebiten.Image, r Rect, ch byte, ts tileState, m Mark, flip float64, mp markColors) {
	sy := 1.0
	showMark := ts == tMarked || ts == tAnswer
	if flip > 0 && flip < 1 {
		sy = math.Abs(math.Cos(flip * math.Pi))
		if flip < 0.5 {
			showMark = false
			if ts == tMarked {
				ts = tTyped
			}
		}
	}
	rr := Rect{r.X, r.CY() - r.H*sy/2, r.W, math.Max(1, r.H*sy)}
	rad := r.W * 0.14
	letter := strings.ToUpper(string(ch))
	var fg color.RGBA
	switch {
	case showMark && ts == tAnswer:
		FillRR(dst, rr, rad, darken(colDanger, 0.55))
		StrokeRR(dst, rr, rad, 2, colDanger)
		fg = colText
	case showMark:
		var c color.RGBA
		switch m {
		case Click:
			c, fg = mp.Click, mp.ClickText
		case Wiggle:
			c, fg = mp.Wiggle, mp.WiggleText
		default:
			c, fg = mp.Silent, mp.SilentText
		}
		FillRR(dst, rr.Move(0, 2), rad, alpha(colBlack, 0.35))
		FillRRV(dst, rr, rad, lighten(c, 0.14), darken(c, 0.12))
		if m != Silent {
			FillRR(dst, Rect{rr.X + rr.W*0.12, rr.Y + rr.H*0.08, rr.W * 0.76, rr.H * 0.12}, rad*0.5, alpha(colWhite, 0.2))
		}
	case ts == tTyped:
		FillRR(dst, rr, rad, colSurface)
		StrokeRR(dst, rr, rad, 2, hex("#7F908F"))
		fg = colText
	default:
		FillRR(dst, rr, rad, alpha(colBlack, 0.25))
		StrokeRR(dst, rr, rad, 1.5, alpha(colBorder, 0.9))
	}
	if ch != 0 && sy > 0.2 {
		Text(dst, letter, Bold, r.W*0.52, r.CX(), r.CY()+r.W*0.03, fg, AlignCenter)
	}
}

func (p *playScene) drawBoard(dst *ebiten.Image, st *LockStyle, mp markColors) {
	l := p.v.Lock()
	L := len(l.Answer)
	type row struct {
		word  string
		state tileState
		idx   int // guess index, -1 for input
	}
	var rows []row
	for i, g := range l.Guesses {
		rows = append(rows, row{g, tMarked, i})
	}
	showInput := p.phase == phInput || p.phase == phSlide
	if showInput {
		rows = append(rows, row{string(p.input), tTyped, -1})
	}
	if p.phase == phSealed || (p.phase == phDone && !p.v.Opened) {
		if p.phase == phDone || p.pt > 30 {
			rows = append(rows, row{l.Answer, tAnswer, -2})
		}
	}
	hidden := 0
	if len(rows) > p.rows {
		hidden = len(rows) - p.rows
		rows = rows[hidden:]
	}
	empty := 0
	if !p.v.Done {
		empty = p.rows - len(rows)
		budget := p.v.Left()
		if showInput {
			budget--
		}
		if empty > budget {
			empty = budget
		}
	}
	offX := 0.0
	alphaBoard := 1.0
	if p.phase == phSlide {
		k := easeOut(p.pt / 26)
		offX = (1 - k) * 80
		alphaBoard = k
	}
	if p.shakeT > 0 {
		offX += math.Sin(p.shakeT*1.3) * p.shakeT * 0.5
	}
	_ = alphaBoard
	y := p.boardR.Y
	if hidden > 0 {
		Text(dst, "+"+itoa(hidden)+" earlier", Regular, 11, p.boardR.CX(), y-8, colTextDim, AlignCenter)
	}
	revealRow := len(l.Guesses) - 1
	for ri, rw := range rows {
		for i := 0; i < L; i++ {
			x := p.boardR.X + float64(i)*(p.tile+p.gap) + offX
			r := Rect{x, y, p.tile, p.tile}
			var ch byte
			if i < len(rw.word) {
				ch = rw.word[i]
			}
			var m Mark
			flip := 0.0
			ts := rw.state
			if rw.state == tMarked {
				m = l.Marks(rw.idx)[i]
				if p.phase == phReveal && rw.idx == revealRow {
					flip = p.revealProgress(i)
					if flip == 0 {
						flip = 0.001
					}
				}
				if p.phase == phUnlock && rw.idx == revealRow {
					jump := math.Sin(clamp((p.pt-float64(i)*5)/18, 0, 1)*math.Pi) * 10
					r = r.Move(0, -jump)
				}
			}
			if rw.state == tTyped {
				if ch == 0 {
					ts = tEmpty
				} else if i < len(p.pop) {
					r = r.Scale(1 + 0.08*p.pop[i])
				}
			}
			if rw.state == tAnswer {
				flip = clamp((p.pt-30-float64(i)*6)/14, 0, 1)
				if p.phase == phDone {
					flip = 1
				}
				if flip >= 1 {
					flip = 0
				}
			}
			drawTile(dst, r, ch, ts, m, flip, mp)
		}
		_ = ri
		y += p.tile + p.gap
	}
	for e := 0; e < empty; e++ {
		for i := 0; i < L; i++ {
			x := p.boardR.X + float64(i)*(p.tile+p.gap) + offX
			drawTile(dst, Rect{x, y, p.tile, p.tile}, 0, tEmpty, Silent, 0, mp)
		}
		y += p.tile + p.gap
	}
}

func (p *playScene) drawKeyboard(dst *ebiten.Image, st *LockStyle, mp markColors) {
	ks := p.v.KeyState()
	ready := len(p.input) == p.v.WordLen() && p.phase == phInput
	for i := range p.keys {
		k := &p.keys[i]
		r := k.r.Scale(1 - 0.05*k.flash)
		h := animHover("k"+k.label+itoa(int(k.c)), in.Hover(k.r) && p.phase == phInput)
		bg := lerpColor(hex("#4A5A5C"), hex("#5A6C6E"), h)
		fg := colText
		switch k.c {
		case 0:
			if ready {
				bg = lerpColor(st.Metal, st.MetalHi, h*0.4)
				fg = colInk
			}
		case 1:
		default:
			switch ks[k.c] {
			case 4:
				bg, fg = mp.Click, mp.ClickText
			case 3:
				bg, fg = mp.Wiggle, mp.WiggleText
			case 2:
				bg = lerpColor(hex("#4A5A5C"), mp.Click, 0.35)
			case 1:
				bg, fg = hex("#1E292B"), alpha(colTextDim, 0.55)
			}
		}
		FillRR(dst, r.Move(0, 2), 7, alpha(colBlack, 0.35))
		FillRRV(dst, r, 7, lighten(bg, 0.06+0.1*k.flash), darken(bg, 0.08))
		if k.c == 0 || k.c >= 'a' && ks[k.c] == 2 {
			if k.c != 0 {
				StrokeRR(dst, r, 7, 2, mp.Click)
			}
		}
		switch k.c {
		case 1:
			drawBackspaceIcon(dst, r.CX(), r.CY(), r.H*0.42, fg)
		case 0:
			Text(dst, k.label, Bold, math.Min(13, r.W*0.2), r.CX(), r.CY()+1, fg, AlignCenter)
		default:
			Text(dst, k.label, Bold, math.Min(20, r.H*0.38), r.CX(), r.CY()+1, fg, AlignCenter)
		}
	}
}

func drawBackspaceIcon(dst *ebiten.Image, cx, cy, s float64, c color.RGBA) {
	w := s * 1.1
	x0 := cx - w/2
	Poly(dst, c, x0, cy, x0+w*0.3, cy-s*0.42, x0+w, cy-s*0.42, x0+w, cy+s*0.42, x0+w*0.3, cy+s*0.42)
	d := s * 0.18
	mx := x0 + w*0.62
	bg := hex("#4A5A5C")
	Line(dst, mx-d, cy-d, mx+d, cy+d, s*0.1, bg)
	Line(dst, mx+d, cy-d, mx-d, cy+d, s*0.1, bg)
}

// drawVault plays the big finale: the dial spins, then the door swings open on golden light.
func (p *playScene) drawVault(dst *ebiten.Image, st *LockStyle) {
	t := p.pt
	FillRect(dst, Rect{0, 0, SW, SH}, alpha(st.BgBot, clamp(t/20, 0, 0.88)))
	r := math.Min(SW, SH) * 0.3
	cx, cy := SW/2, SH/2
	appear := easeBack(t / 24)
	if p.doorImg == nil || p.doorImg.Bounds().Dx() != int(math.Ceil(2.2*r*S)) {
		sz := int(math.Ceil(2.2 * r * S))
		p.doorImg = ebiten.NewImage(sz, sz)
	}
	spin := easeInOut(clamp((t-10)/62, 0, 1)) * math.Pi * 3
	open := easeInOut(clamp((t-74)/50, 0, 1))
	// light behind the door
	if t > 60 {
		// the doorway: a heavy frame around a lit room full of gold
		fa := clamp((t-60)/14, 0, 1)
		FillCircle(dst, cx, cy, r*1.08, alpha(darken(st.MetalLo, 0.2), fa))
		FillCircle(dst, cx, cy, r*0.97, alpha(hex("#2A1C08"), fa))
		RadialGlow(dst, cx, cy-r*0.1, r*0.95, hex("#FFE7A6"), open*0.95)
		bar := func(x, y, w float64) {
			h := w * 0.36
			Poly(dst, alpha(darken(colGold, 0.15), open), x-w/2, y+h, x+w/2, y+h, x+w*0.38, y, x-w*0.38, y)
			Poly(dst, alpha(lighten(colGold, 0.35), open), x-w*0.38, y, x+w*0.38, y, x+w*0.3, y+h*0.25, x-w*0.3, y+h*0.25)
		}
		bw := r * 0.42
		by := cy + r*0.38
		for i := -1; i <= 1; i++ {
			bar(cx+float64(i)*bw*0.95, by, bw)
		}
		bar(cx-bw*0.48, by-bw*0.34, bw)
		bar(cx+bw*0.48, by-bw*0.34, bw)
		bar(cx, by-bw*0.68, bw)
	}
	if open > 0 {
		RadialGlow(dst, cx, cy, r*(1.2+open*1.4), st.MetalHi, open*0.55)
	}
	p.doorImg.Clear()
	oldS := S
	// draw the dial into the offscreen image at its own origin
	drawDialAt(p.doorImg, 1.1*r, 1.1*r, r, spin, clamp(t/70, 0, 1), st)
	S = oldS
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	w := float64(p.doorImg.Bounds().Dx())
	op.GeoM.Translate(-w/2, -w/2)
	op.GeoM.Scale(appear, appear)
	// swing: squash horizontally around the left hinge
	sx := 1 - open*0.88
	op.GeoM.Translate(w/2*appear, 0)
	op.GeoM.Scale(sx, 1)
	op.GeoM.Translate(-w/2*appear, 0)
	op.GeoM.Translate(cx*S-open*r*0.15*S, cy*S)
	op.ColorScale.Scale(float32(1-open*0.35), float32(1-open*0.35), float32(1-open*0.35), 1)
	dst.DrawImage(p.doorImg, op)
	if t > 100 {
		a := clamp((t-100)/20, 0, 1)
		TextSpaced(dst, "VAULT OPEN", Bold, 34, cx, cy+r*1.25, 5, alpha(st.MetalHi, a), AlignCenter)
	}
}

// drawDialAt draws the dial into an offscreen image (device pixels = logical * S).
func drawDialAt(dst *ebiten.Image, cx, cy, r, rot, glow float64, st *LockStyle) {
	drawDial(dst, cx, cy, r, rot, glow, st)
}
