package game

import (
	"fmt"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type menuScene struct {
	daily, practice, help, stats, locks, options, quit Button
	t                                                  float64
	dialY, dialR, titleY, chipsY                       float64
}

func newMenuScene() *menuScene {
	m := &menuScene{}
	m.daily = Button{ID: "m-daily", Kind: BtnGold, Size: 19}
	m.practice = Button{ID: "m-prac", Kind: BtnDark, Size: 17, Icon: IconDice}
	m.help = Button{ID: "m-help", Kind: BtnDark, Size: 14, Icon: IconHelp, Label: "How to play"}
	m.stats = Button{ID: "m-stats", Kind: BtnDark, Size: 14, Icon: IconStats, Label: "Stats"}
	m.locks = Button{ID: "m-locks", Kind: BtnDark, Size: 14, Icon: IconKey, Label: "Locks"}
	m.options = Button{ID: "m-opt", Kind: BtnGhost, Icon: IconGear, Size: 22}
	m.quit = Button{ID: "m-quit", Kind: BtnGhost, Label: "Quit", Size: 15}
	return m
}

func fmtDur(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, s/60%60, s%60)
}

func (m *menuScene) layout() {
	bw := math.Min(390, SW-40)
	m.dialR = clamp(math.Min(SW*0.24, SH*0.13), 54, 118)
	titleSize := clamp(SW*0.12, 36, 60)
	contentH := m.dialR*2 + 24 + titleSize + 36 + 44 + 72 + 12 + 62 + 12 + 52
	if !cfg.Mobile {
		contentH += 52
	}
	top := math.Max(54, (SH-contentH)/2)
	m.dialY = top + m.dialR
	m.titleY = m.dialY + m.dialR + 24 + titleSize/2
	m.chipsY = m.titleY + titleSize/2 + 36 + 6
	y := m.chipsY + 34
	x := SW/2 - bw/2
	m.daily.R = Rect{x, y, bw, 72}
	y += 72 + 12
	m.practice.R = Rect{x, y, bw, 62}
	y += 62 + 12
	gap := 10.0
	w3 := (bw - 2*gap) / 3
	m.help.R = Rect{x, y, w3, 52}
	m.stats.R = Rect{x + w3 + gap, y, w3, 52}
	m.locks.R = Rect{x + 2*(w3+gap), y, w3, 52}
	y += 52 + 10
	m.quit.R = Rect{SW/2 - 60, y, 120, 42}
	m.options.R = Rect{SW - 58, 10, 48, 48}
}

func (m *menuScene) labels() {
	v := save.Daily
	n := TodayNumber()
	m.daily.Label = "Today's Vault  #" + itoa(n)
	switch {
	case v == nil || v.Number != n || v.Used == 0:
		m.daily.Sub = "3 locks  ·  15 guesses  ·  one shot a day"
	case !v.Done:
		m.daily.Label = "Continue Vault  #" + itoa(n)
		m.daily.Sub = fmt.Sprintf("Lock %d of 3  ·  %d guesses left", v.Cur+1, v.Left())
	case v.Opened:
		m.daily.Sub = fmt.Sprintf("Opened, %d to spare  ·  next in %s", v.Spare(), fmtDur(UntilNextVault()))
	default:
		m.daily.Sub = "Sealed  ·  next vault in " + fmtDur(UntilNextVault())
	}
	m.practice.Label = "Practice Vault"
	if p := save.Practice; p != nil && !p.Done && p.Used > 0 {
		m.practice.Sub = fmt.Sprintf("Continue  ·  lock %d of 3, %d left", p.Cur+1, p.Left())
	} else {
		m.practice.Sub = "Unlimited random vaults"
	}
}

func (m *menuScene) Update() {
	m.t++
	m.layout()
	m.labels()
	if m.daily.Update() {
		v := todaysVault()
		game.SetScene(newPlayScene(v))
	}
	if m.practice.Update() {
		if save.Practice == nil || save.Practice.Done {
			save.Practice = NewVault(Practice, 0, PracticeAnswers())
			writeSave()
		}
		game.SetScene(newPlayScene(save.Practice))
	}
	if m.help.Update() {
		game.Push(newHelpPanel())
	}
	if m.stats.Update() {
		game.Push(newStatsPanel())
	}
	if m.locks.Update() {
		game.Push(newLocksPanel())
	}
	if m.options.Update() {
		game.Push(newOptionsPanel())
	}
	if !cfg.Mobile && m.quit.Update() {
		QuitRequested = true
	}
	if updateChecker.available() && save.Settings.SkippedVersion != updateChecker.latestVersion() && !shownUpdatePrompt {
		shownUpdatePrompt = true
		game.Push(newUpdatePanel())
	}
}

var shownUpdatePrompt bool

func (m *menuScene) Draw(dst *ebiten.Image) {
	m.layout()
	drawBackground(dst)
	st := style()
	glow := 0.5 + 0.5*math.Sin(m.t/50)
	drawDial(dst, SW/2, m.dialY, m.dialR, m.t/900, glow*0.6, st)
	titleSize := clamp(SW*0.12, 36, 60)
	sp := titleSize * 0.14
	TextSpaced(dst, "KEYHOLE", Bold, titleSize, SW/2+2, m.titleY+3, sp, alpha(colBlack, 0.6), AlignCenter)
	TextSpaced(dst, "KEYHOLE", Bold, titleSize, SW/2, m.titleY, sp, st.MetalHi, AlignCenter)
	TextSpaced(dst, "THE DAILY WORD VAULT", Medium, 13, SW/2, m.titleY+titleSize*0.62+6, 4, colTextDim, AlignCenter)

	// streak and key chips
	chips := []struct {
		icon func(*ebiten.Image, float64, float64, float64, colorRGBA)
		txt  string
	}{
		{IconFlame, plural(currentStreak(), "day streak", "day streak")},
		{IconKey, plural(save.Stats.GoldKeys, "gold key", "gold keys")},
	}
	total := 0.0
	ws := make([]float64, len(chips))
	for i, c := range chips {
		ws[i] = TextWidth(c.txt, Medium, 13) + 44
		total += ws[i]
	}
	total += 10
	x := SW/2 - total/2
	for i, c := range chips {
		r := Rect{x, m.chipsY - 15, ws[i], 30}
		FillRR(dst, r, 15, alpha(colBlack, 0.3))
		StrokeRR(dst, r, 15, 1, colBorder)
		c.icon(dst, r.X+18, r.CY(), 16, st.Metal)
		Text(dst, c.txt, Medium, 13, r.X+32, r.CY()+1, colText, AlignLeft)
		x += ws[i] + 10
	}

	m.daily.Draw(dst)
	m.practice.Draw(dst)
	m.help.Draw(dst)
	m.stats.Draw(dst)
	m.locks.Draw(dst)
	m.options.Draw(dst)
	if !cfg.Mobile {
		m.quit.Draw(dst)
	}
	Text(dst, "v"+Version, Regular, 11, SW-12, SH-14, alpha(colTextDim, 0.6), AlignRight)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return itoa(n) + " " + many
}
