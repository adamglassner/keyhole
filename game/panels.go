package game

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// panel is a modal card. Content is drawn immediate-mode: content() runs once
// with dst == nil to handle input, then again with dst set to draw.
type panel struct {
	title    string
	maxW     float64
	content  func(dst *ebiten.Image, x, y, w float64) float64
	close    Button
	anim     float64
	closing  bool
	scroll   float64
	contentH float64
	card     Rect
	body     Rect
}

func newPanel(title string, maxW float64) *panel {
	return &panel{title: title, maxW: maxW, close: Button{ID: "pn-close-" + title, Kind: BtnGhost, Icon: IconClose, Size: 22}}
}

func (p *panel) layout() {
	w := math.Min(p.maxW, SW-20)
	head := 58.0
	h := math.Min(p.contentH+head+18, SH-28)
	if p.contentH == 0 {
		h = math.Min(500, SH-28)
	}
	p.card = Rect{SW/2 - w/2, SH/2 - h/2, w, h}
	p.body = Rect{p.card.X, p.card.Y + head, w, h - head - 8}
	p.close.R = Rect{p.card.Right() - 52, p.card.Y + 6, 46, 46}
	maxScroll := math.Max(0, p.contentH-p.body.H)
	p.scroll = clamp(p.scroll, 0, maxScroll)
}

func (p *panel) Dismiss() { p.closing = true }

func (p *panel) Update() bool {
	if p.closing {
		p.anim = approach(p.anim, 0, 0.35)
		return p.anim > 0.05
	}
	p.anim = approach(p.anim, 1, 0.3)
	p.layout()
	if p.close.Update() || in.Back {
		p.closing = true
		return true
	}
	if in.Pressed && !p.card.Contains(in.MX, in.MY) && !in.consumed {
		in.Consume()
		p.closing = true
		return true
	}
	if p.body.Contains(in.MX, in.MY) {
		p.scroll -= in.Wheel * 40
		if in.Down && math.Abs(in.DragDY) > 0 {
			p.scroll -= in.DragDY
		}
	}
	p.layout()
	pad := 22.0
	if in.Pressed && !p.body.Contains(in.MX, in.MY) {
		// taps on the header shouldn't hit scrolled content
		saved := in.Pressed
		in.Pressed = false
		p.contentH = p.content(nil, p.body.X+pad, p.body.Y-p.scroll, p.body.W-2*pad)
		in.Pressed = saved
	} else {
		p.contentH = p.content(nil, p.body.X+pad, p.body.Y-p.scroll, p.body.W-2*pad)
	}
	return true
}

func (p *panel) Draw(dst *ebiten.Image) {
	p.layout()
	a := easeOut(p.anim)
	FillRect(dst, Rect{0, 0, SW, SH}, alpha(colBlack, 0.62*a))
	off := (1 - a) * 24
	card := p.card.Move(0, off)
	Shadow(dst, card, 18, 2*a)
	st := style()
	FillRRV(dst, card, 18, lighten(st.BgTop, 0.05), st.BgBot)
	StrokeRR(dst, card, 18, 1.5, alpha(st.Metal, 0.55))
	FillRR(dst, Rect{card.X + 18, card.Y + 56, card.W - 36, 1}, 0, alpha(colBorder, 0.9))
	TextSpaced(dst, strings.ToUpper(p.title), Bold, 16, card.X+22, card.Y+30, 2.5, st.MetalHi, AlignLeft)
	p.close.R = p.close.R.Move(0, off)
	p.close.Draw(dst)
	body := p.body.Move(0, off)
	sub := dst.SubImage(rectToImage(body)).(*ebiten.Image)
	pad := 22.0
	p.content(sub, body.X+pad, body.Y-p.scroll, body.W-2*pad)
	if p.contentH > p.body.H+1 {
		frac := p.body.H / p.contentH
		th := p.body.H * frac
		ty := body.Y + (p.body.H-th)*(p.scroll/(p.contentH-p.body.H))
		FillRR(dst, Rect{body.Right() - 7, ty, 4, th}, 2, alpha(colTextDim, 0.5))
	}
}

func rectToImage(r Rect) imageRect {
	return imageRectFrom(int(r.X*S), int(r.Y*S), int(math.Ceil(r.Right()*S)), int(math.Ceil(r.Bottom()*S)))
}

// para draws wrapped text and returns its height.
func para(dst *ebiten.Image, s string, x, y, w float64, size float64, c colorRGBA) float64 {
	return TextWrapped(dst, s, Regular, size, x, y, w, size*1.5, c, AlignLeft)
}

func heading(dst *ebiten.Image, s string, x, y float64) float64 {
	if dst != nil {
		TextSpaced(dst, strings.ToUpper(s), Bold, 12, x, y+9, 2, style().Metal, AlignLeft)
	}
	return 24
}

// ---- How to play

func newHelpPanel() *panel {
	p := newPanel("How to play", 480)
	p.content = func(dst *ebiten.Image, x, y, w float64) float64 {
		y0 := y
		mp := marksPalette()
		y += para(dst, "Crack today's vault. It has three locks: a 4-letter word, then a 5-letter word, then a 6-letter word.", x, y, w, 15, colText)
		y += 8
		y += para(dst, "You get 15 guesses for the whole vault, shared by all three locks. Guesses you save on one lock carry over to the next.", x, y, w, 15, colText)
		y += 14
		y += heading(dst, "Every guess turns the tumblers", x, y)
		t := math.Min(46, (w-4*6)/5)
		word := "vault"
		ms := []Mark{Click, Wiggle, Silent, Silent, Wiggle}
		if dst != nil {
			for i := range word {
				drawTile(dst, Rect{x + float64(i)*(t+6), y + 4, t, t}, word[i], tMarked, ms[i], 0, mp)
			}
		}
		y += t + 14
		legend := []struct {
			m    Mark
			name string
			txt  string
		}{
			{Click, "Click", "right letter, right spot"},
			{Wiggle, "Wiggle", "in the word, wrong spot"},
			{Silent, "Silent", "not in the word"},
		}
		for _, l := range legend {
			if dst != nil {
				drawTile(dst, Rect{x, y + 2, 24, 24}, 0, tMarked, l.m, 0, mp)
				Text(dst, l.name, Bold, 15, x+34, y+14, colText, AlignLeft)
				Text(dst, l.txt, Regular, 14, x+34+TextWidth(l.name, Bold, 15)+8, y+14, colTextDim, AlignLeft)
			}
			y += 32
		}
		y += 10
		y += heading(dst, "Pins", x, y)
		y += para(dst, "When a lock opens, its letters become pins. Before your first guess on the next lock, tap a pin to test it for free. You'll learn whether that letter is in the new word and roughly where: front, middle or back.", x, y, w, 15, colText)
		y += 6
		y += para(dst, "Open a lock in 3 guesses or fewer and you get two free pin tests on the next one.", x, y, w, 15, colText)
		y += 14
		y += heading(dst, "Gold keys", x, y)
		y += para(dst, "Open the daily vault with 6 or more guesses to spare to earn a gold key. Gold keys unlock new lock styles, from a steel safe to a royal vault.", x, y, w, 15, colText)
		y += 14
		y += heading(dst, "A new vault every day", x, y)
		y += para(dst, "Everyone gets the same vault each day, and a new one opens at midnight. Want more? Practice vaults are unlimited.", x, y, w, 15, colText)
		y += 14
		y += heading(dst, "Zen Vault", x, y)
		y += para(dst, "No guess limit and no way to fail. Every lock opens eventually, and your score is how many guesses it took. Fewer is better.", x, y, w, 15, colText)
		y += 14
		y += heading(dst, "Theme Vaults", x, y)
		y += para(dst, "Pick a topic, like Animals, Food or Space. Every word in the vault fits it, which makes the locks easier to crack.", x, y, w, 15, colText)
		return y - y0 + 6
	}
	return p
}

// ---- Stats

func newStatsPanel() *panel {
	p := newPanel("Statistics", 460)
	p.content = func(dst *ebiten.Image, x, y, w float64) float64 {
		y0 := y
		st := style()
		s := save.Stats
		pct := 0
		if s.Played > 0 {
			pct = int(math.Round(float64(s.Opened) * 100 / float64(s.Played)))
		}
		nums := []struct{ n, l string }{{itoa(s.Played), "Played"}, {itoa(pct) + "%", "Opened"}, {itoa(currentStreak()), "Streak"}, {itoa(s.MaxStreak), "Best"}}
		cw := w / 4
		if dst != nil {
			for i, n := range nums {
				cx := x + cw*(float64(i)+0.5)
				Text(dst, n.n, Bold, 30, cx, y+20, colText, AlignCenter)
				Text(dst, n.l, Regular, 12, cx, y+46, colTextDim, AlignCenter)
			}
		}
		y += 66
		if dst != nil {
			r := Rect{x, y, w, 44}
			FillRR(dst, r, 12, alpha(colBlack, 0.28))
			IconKey(dst, r.X+24, r.CY(), 22, st.Metal)
			Text(dst, plural(s.GoldKeys, "gold key", "gold keys")+" earned", Bold, 15, r.X+46, r.CY()+1, colText, AlignLeft)
		}
		y += 58
		y += heading(dst, "Guesses to spare", x, y)
		groups := []struct {
			label  string
			lo, hi int
		}{{"0-2", 0, 2}, {"3-5", 3, 5}, {"6-8", 6, 8}, {"9+", 9, 15}}
		maxN := 1
		counts := make([]int, len(groups))
		for gi, g := range groups {
			for k := g.lo; k <= g.hi; k++ {
				counts[gi] += s.Spare[k]
			}
			if counts[gi] > maxN {
				maxN = counts[gi]
			}
		}
		last := -1
		if v := save.Daily; v != nil && v.Done && v.Opened {
			for gi, g := range groups {
				if v.Spare() >= g.lo && v.Spare() <= g.hi {
					last = gi
				}
			}
		}
		for gi, g := range groups {
			if dst != nil {
				Text(dst, g.label, Medium, 13, x, y+13, colTextDim, AlignLeft)
				bw := 28 + (w-70)*float64(counts[gi])/float64(maxN)
				c := colSurf2
				if gi == last {
					c = st.Metal
				}
				r := Rect{x + 40, y + 2, bw, 22}
				FillRR(dst, r, 5, c)
				fc := colText
				if gi == last {
					fc = colInk
				}
				Text(dst, itoa(counts[gi]), Bold, 13, r.Right()-8, r.CY()+1, fc, AlignRight)
			}
			y += 30
		}
		y += 10
		y += heading(dst, "Practice, Zen and Themes", x, y)
		if dst != nil {
			Text(dst, fmt.Sprintf("Practice: %d played, %d opened, best %d to spare", s.PracPlayed, s.PracOpened, s.PracBest), Regular, 14, x, y+10, colText, AlignLeft)
			zb := "no best yet"
			if s.ZenBest > 0 {
				zb = "best " + plural(s.ZenBest, "guess", "guesses")
			}
			Text(dst, fmt.Sprintf("Zen: %d opened, %s", s.ZenPlayed, zb), Regular, 14, x, y+34, colText, AlignLeft)
			Text(dst, fmt.Sprintf("Themes: %d played, %d opened", s.ThemePlay, s.ThemeOpen), Regular, 14, x, y+58, colText, AlignLeft)
		}
		y += 78
		if v := save.Daily; v != nil && v.Done && v.Number == TodayNumber() {
			y += 6
			if dst != nil {
				Text(dst, "Next vault in "+fmtDur(UntilNextVault()), Bold, 16, x+w/2, y+12, st.MetalHi, AlignCenter)
			}
			y += 30
		}
		return y - y0 + 4
	}
	return p
}

// ---- Options

func newOptionsPanel() *panel {
	p := newPanel("Options", 460)
	music, sfx := &Slider{}, &Slider{}
	check := &Button{ID: "opt-check", Kind: BtnDark, Label: "Check for updates", Size: 14}
	help := &Button{ID: "opt-help", Kind: BtnDark, Label: "How to play", Size: 14, Icon: IconHelp}
	p.content = func(dst *ebiten.Image, x, y, w float64) float64 {
		y0 := y
		row := func(label string) {
			if dst != nil {
				Text(dst, label, Medium, 15, x, y+14, colText, AlignLeft)
			}
		}
		// volumes
		for i, sl := range []*Slider{music, sfx} {
			lab := "Music"
			v, m := &save.Settings.MusicVolume, &save.Settings.MusicMuted
			if i == 1 {
				lab, v, m = "Sound effects", &save.Settings.SFXVolume, &save.Settings.SFXMuted
			}
			row(lab)
			sl.R = Rect{x + 130, y + 10, w - 130 - 70, 8}
			tr := Rect{x + w - 50, y + 3, 50, 26}
			if dst == nil {
				if sl.Update(v) {
					*m = false
					writeSave()
				}
				if Toggle(nil, lab, tr, !*m) {
					*m = !*m
					writeSave()
				}
			} else {
				sl.Draw(dst, *v, *m)
				Toggle(dst, lab, tr, !*m)
			}
			y += 46
		}
		toggles := []struct {
			label, desc string
			val         *bool
			show        bool
			apply       func()
		}{
			{"High contrast", "Orange and blue tiles", &save.Settings.HighContrast, true, nil},
			{"Fullscreen", "F11 also works", &save.Settings.Fullscreen, !cfg.Mobile, func() { ebiten.SetFullscreen(save.Settings.Fullscreen) }},
			{"Check for updates on start", "Looks for new versions on GitHub", &save.Settings.AutoCheckUpdates, cfg.CanUpdate, nil},
		}
		for _, t := range toggles {
			if !t.show {
				continue
			}
			tr := Rect{x + w - 50, y + 8, 50, 26}
			if dst == nil {
				if Toggle(nil, t.label, tr, *t.val) {
					*t.val = !*t.val
					if t.apply != nil {
						t.apply()
					}
					writeSave()
				}
			} else {
				Text(dst, t.label, Medium, 15, x, y+12, colText, AlignLeft)
				Text(dst, t.desc, Regular, 12, x, y+32, colTextDim, AlignLeft)
				Toggle(dst, t.label, tr, *t.val)
			}
			y += 52
		}
		y += 4
		help.R = Rect{x, y, w, 46}
		if dst == nil {
			if help.Update() {
				game.Push(newHelpPanel())
			}
		} else {
			help.Draw(dst)
		}
		y += 58
		if cfg.CanUpdate {
			check.R = Rect{x, y, w, 46}
			if dst == nil {
				if check.Update() {
					if updateChecker.available() {
						game.Push(newUpdatePanel())
					} else {
						updateChecker.check(true)
					}
				}
			} else {
				if updateChecker.available() {
					check.Label = "Update to " + updateChecker.latestVersion()
				}
				check.Draw(dst)
			}
			y += 52
			if dst != nil {
				TextWrapped(dst, updateChecker.statusText(), Regular, 12, x, y, w, 18, updateChecker.statusColor(), AlignCenter)
			}
			y += 24
		}
		if dst != nil {
			Text(dst, "Keyhole "+Version+"  ·  made by Ironite", Regular, 12, x+w/2, y+12, alpha(colTextDim, 0.8), AlignCenter)
		}
		y += 26
		return y - y0
	}
	return p
}

// ---- Lock collection

func newLocksPanel() *panel {
	p := newPanel("Lock styles", 480)
	p.content = func(dst *ebiten.Image, x, y, w float64) float64 {
		y0 := y
		keys := save.Stats.GoldKeys
		if dst != nil {
			IconKey(dst, x+12, y+12, 22, style().Metal)
			Text(dst, "You have "+plural(keys, "gold key", "gold keys"), Bold, 15, x+32, y+12, colText, AlignLeft)
		}
		y += 28
		y += para(dst, "Earn a key by opening the daily vault with 6 or more guesses to spare.", x, y, w, 13, colTextDim)
		y += 12
		for i := range lockStyles {
			s := &lockStyles[i]
			r := Rect{x, y, w, 76}
			unlocked := keys >= s.Keys
			sel := save.Settings.LockStyle == i
			if dst == nil {
				if unlocked && in.Click(r) {
					save.Settings.LockStyle = i
					writeSave()
					playSFX("unlock")
				}
			} else {
				h := animHover("lock"+s.Name, in.Hover(r) && unlocked)
				bg := lerpColor(alpha(colBlack, 0.3), alpha(colWhite, 0.06), h)
				FillRR(dst, r, 14, bg)
				if sel {
					StrokeRR(dst, r, 14, 2, s.Metal)
				} else {
					StrokeRR(dst, r, 14, 1, colBorder)
				}
				GradientRect(dst, Rect{r.X + 10, r.Y + 10, 56, 56}, s.BgTop, s.BgBot)
				drawDial(dst, r.X+38, r.CY(), 24, 0, 0.3, s)
				if !unlocked {
					FillRR(dst, Rect{r.X + 8, r.Y + 8, 60, 60}, 30, alpha(colBlack, 0.55))
					drawPadlock(dst, r.X+38, r.CY()+2, 22, hex("#55666A"), hex("#22292B"), 0, false)
				}
				nc := colText
				if !unlocked {
					nc = colTextDim
				}
				Text(dst, s.Name, Bold, 16, r.X+80, r.Y+27, nc, AlignLeft)
				sub := s.Blurb
				if !unlocked {
					sub = "Needs " + plural(s.Keys, "gold key", "gold keys")
				}
				Text(dst, sub, Regular, 13, r.X+80, r.Y+51, colTextDim, AlignLeft)
				if sel {
					Text(dst, "IN USE", Bold, 11, r.Right()-14, r.CY(), s.MetalHi, AlignRight)
				}
			}
			y += 86
		}
		return y - y0
	}
	return p
}

// ---- Result

func newResultPanel(ps *playScene) *panel {
	v := ps.v
	title := "Vault sealed"
	if v.Opened {
		title = "Vault opened"
	}
	p := newPanel(title, 460)
	share := &Button{ID: "r-share", Kind: BtnGold, Label: "Share result", Icon: IconShare, Size: 17}
	if !cfg.Mobile {
		share.Label = "Copy result"
	}
	again := &Button{ID: "r-prac", Kind: BtnDark, Label: "Practice vault", Icon: IconDice, Size: 15}
	switch v.Kind {
	case Practice:
		again.Label = "New practice vault"
	case ZenKind:
		again.Label, again.Icon = "New zen vault", IconZen
	case ThemeKind:
		again.Label, again.Icon = "Another "+v.Theme+" vault", IconTheme
	}
	menu := &Button{ID: "r-menu", Kind: BtnGhost, Label: "Back to menu", Size: 15}
	p.content = func(dst *ebiten.Image, x, y, w float64) float64 {
		y0 := y
		st := style()
		mp := marksPalette()
		cx := x + w/2
		sub := "Practice vault"
		switch v.Kind {
		case Daily:
			sub = "Daily vault #" + itoa(v.Number)
		case ZenKind:
			sub = "Zen vault"
		case ThemeKind:
			sub = "Theme: " + v.Theme
		}
		if dst != nil {
			Text(dst, sub, Medium, 13, cx, y+8, colTextDim, AlignCenter)
		}
		y += 22
		if v.Kind == ZenKind {
			if dst != nil {
				Text(dst, itoa(v.Used), Bold, 56, cx, y+36, st.MetalHi, AlignCenter)
				label := "guesses to open all three locks"
				if save.Stats.ZenBest == v.Used && save.Stats.ZenPlayed > 1 {
					label = "guesses. A new personal best!"
				}
				Text(dst, label, Medium, 14, cx, y+76, colText, AlignCenter)
			}
			y += 94
		} else if v.Opened {
			if dst != nil {
				Text(dst, itoa(v.Spare()), Bold, 56, cx, y+36, st.MetalHi, AlignCenter)
				Text(dst, "guesses to spare", Medium, 14, cx, y+76, colText, AlignCenter)
			}
			y += 94
		} else {
			if dst != nil {
				drawPadlock(dst, cx, y+38, 54, darken(colDanger, 0.2), darken(colDanger, 0.6), 0, true)
				Text(dst, []string{"The first lock held firm. Try again tomorrow!", "One lock down. Two to go next time.", "Stuck on the last lock. So close!"}[v.Cur], Medium, 14, cx, y+82, colText, AlignCenter)
			}
			y += 98
		}
		if ps.earnedKey {
			r := Rect{x, y, w, 50}
			if dst != nil {
				FillRRV(dst, r, 12, lighten(st.Metal, 0.2), darken(st.Metal, 0.15))
				IconKey(dst, r.X+28, r.CY(), 26, colInk)
				Text(dst, "Gold key earned!", Bold, 16, r.X+52, r.CY()-(map[bool]float64{true: 8, false: 0})[ps.newStyle != ""], colInk, AlignLeft)
				if ps.newStyle != "" {
					Text(dst, "New lock style: "+ps.newStyle, Regular, 12, r.X+52, r.CY()+11, alpha(colInk, 0.8), AlignLeft)
				}
			}
			y += 62
		}
		// the three answers
		for i := 0; i < 3; i++ {
			l := &v.Locks[i]
			t := 26.0
			if dst != nil {
				for k := 0; k < len(l.Answer); k++ {
					m := Click
					ts := tMarked
					if !l.Opened {
						ts = tAnswer
					}
					drawTile(dst, Rect{x + float64(k)*(t+4), y, t, t}, l.Answer[k], ts, m, 0, mp)
				}
				info := plural(len(l.Guesses), "guess", "guesses")
				if !l.Opened {
					if i > v.Cur {
						info = "never reached"
					} else {
						info = "not opened"
					}
				}
				Text(dst, info, Regular, 13, x+w, y+t/2, colTextDim, AlignRight)
			}
			y += t + 8
		}
		y += 8
		if v.Kind == Daily {
			s := save.Stats
			pct := 0
			if s.Played > 0 {
				pct = int(math.Round(float64(s.Opened) * 100 / float64(s.Played)))
			}
			nums := []struct{ n, l string }{{itoa(s.Played), "Played"}, {itoa(pct) + "%", "Opened"}, {itoa(currentStreak()), "Streak"}, {itoa(s.MaxStreak), "Best"}}
			cw := w / 4
			if dst != nil {
				for i, n := range nums {
					ncx := x + cw*(float64(i)+0.5)
					Text(dst, n.n, Bold, 22, ncx, y+14, colText, AlignCenter)
					Text(dst, n.l, Regular, 11, ncx, y+34, colTextDim, AlignCenter)
				}
			}
			y += 50
			if dst != nil {
				Text(dst, "Next vault in "+fmtDur(UntilNextVault()), Bold, 15, cx, y+10, st.MetalHi, AlignCenter)
			}
			y += 30
		}
		share.R = Rect{x, y, w, 52}
		again.R = Rect{x, y + 62, w, 48}
		menu.R = Rect{x, y + 118, w, 42}
		if dst == nil {
			if share.Update() {
				if cfg.Share != nil {
					if msg := cfg.Share(v.ShareText()); msg != "" {
						game.Toast(msg)
					}
				} else {
					game.Toast("Sharing isn't available here")
				}
			}
			if again.Update() {
				var nv *Vault
				switch v.Kind {
				case ZenKind:
					save.Zen = NewVault(ZenKind, 0, ZenAnswers())
					nv = save.Zen
				case ThemeKind:
					nv = newThemeVault(v.Theme)
				default:
					save.Practice = NewVault(Practice, 0, PracticeAnswers())
					nv = save.Practice
				}
				writeSave()
				p.Dismiss()
				game.SetScene(newPlayScene(nv))
			}
			if menu.Update() {
				p.Dismiss()
				game.SetScene(newMenuScene())
			}
		} else {
			share.Draw(dst)
			again.Draw(dst)
			menu.Draw(dst)
		}
		y += 166
		return y - y0
	}
	return p
}

// ---- Update available

func newUpdatePanel() *panel {
	p := newPanel("Update available", 440)
	now := &Button{ID: "u-now", Kind: BtnGold, Label: "Update now", Size: 16}
	later := &Button{ID: "u-later", Kind: BtnDark, Label: "Later", Size: 15}
	skip := &Button{ID: "u-skip", Kind: BtnGhost, Label: "Skip this version", Size: 14}
	p.content = func(dst *ebiten.Image, x, y, w float64) float64 {
		y0 := y
		ver := updateChecker.latestVersion()
		y += para(dst, "Keyhole "+ver+" is ready to download. You have "+Version+". Your stats, streak and today's vault are kept."+map[bool]string{true: " Android will ask you to confirm the install.", false: ""}[cfg.Mobile], x, y, w, 14, colText)
		y += 12
		if dst != nil {
			TextWrapped(dst, updateChecker.statusText(), Regular, 12, x, y, w, 18, updateChecker.statusColor(), AlignCenter)
		}
		y += 30
		now.R = Rect{x, y, w, 50}
		later.R = Rect{x, y + 60, (w - 10) / 2, 44}
		skip.R = Rect{x + (w+10)/2, y + 60, (w - 10) / 2, 44}
		if dst == nil {
			if now.Update() {
				updateChecker.download()
			}
			if later.Update() {
				p.Dismiss()
			}
			if skip.Update() {
				save.Settings.SkippedVersion = ver
				writeSave()
				p.Dismiss()
			}
		} else {
			now.Draw(dst)
			later.Draw(dst)
			skip.Draw(dst)
		}
		y += 110
		return y - y0
	}
	return p
}

// ---- Theme picker

func newThemeVault(name string) *Vault {
	t := ThemeByName(name)
	if t == nil {
		t = &themes[0]
	}
	v := NewVault(ThemeKind, 0, ThemeAnswers(t))
	v.Theme = t.Name
	save.Themed = v
	return v
}

func newThemesPanel() *panel {
	p := newPanel("Theme Vaults", 480)
	cont := &Button{ID: "th-cont", Kind: BtnGold, Size: 16, Icon: IconTheme}
	p.content = func(dst *ebiten.Image, x, y, w float64) float64 {
		y0 := y
		st := style()
		y += para(dst, "Every word in the vault fits the topic you pick. 15 guesses, three locks.", x, y, w, 14, colTextDim)
		y += 10
		if t := save.Themed; t != nil && !t.Done && t.Used > 0 {
			cont.Label = "Continue: " + t.Theme
			cont.R = Rect{x, y, w, 50}
			if dst == nil {
				if cont.Update() {
					p.Dismiss()
					game.SetScene(newPlayScene(t))
				}
			} else {
				cont.Draw(dst)
			}
			y += 62
		}
		gap := 10.0
		cw := (w - gap) / 2
		for i := range themes {
			t := &themes[i]
			r := Rect{x + float64(i%2)*(cw+gap), y + float64(i/2)*(78+gap), cw, 78}
			if dst == nil {
				if in.Click(r) {
					playSFX("button")
					v := newThemeVault(t.Name)
					writeSave()
					p.Dismiss()
					game.SetScene(newPlayScene(v))
				}
				continue
			}
			h := animHover("theme"+t.Name, in.Hover(r))
			FillRR(dst, r, 14, lerpColor(alpha(colBlack, 0.3), alpha(colWhite, 0.07), h))
			StrokeRR(dst, r, 14, 1.2, lerpColor(colBorder, st.Metal, h))
			Text(dst, t.Name, Bold, 16, r.X+14, r.Y+24, colText, AlignLeft)
			TextWrapped(dst, t.Blurb, Regular, 12, r.X+14, r.Y+38, r.W-24, 16, colTextDim, AlignLeft)
		}
		y += float64((len(themes)+1)/2) * (78 + gap)
		return y - y0
	}
	return p
}
