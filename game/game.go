// Package game is Keyhole: a daily word vault. The same code runs on Windows
// (package main at the repo root) and Android (package mobile).
package game

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Version is the app version. Bump it (and add a CHANGELOG.md entry) for every release.
// The GitHub release tag must be "v" + Version.
const Version = "1.3.0"

const (
	AppName     = "Keyhole"
	GitHubOwner = "adamglassner"
	GitHubRepo  = "keyhole"
)

// Config tells the game about the platform it runs on.
type Config struct {
	DataDir   string // where keyhole.json is saved ("" = don't save)
	Mobile    bool   // phone layout: no hover, no quit button, no fullscreen
	CanUpdate bool   // the built-in updater is available (Windows and Android)
	// InstallUpdate hands a downloaded update to the platform (Android's installer).
	// When nil, Windows runs the downloaded setup program itself.
	InstallUpdate func(path string)
	Share         func(string) string // shares or copies the result; returns a message for a toast
	Quit          func()
}

var cfg Config

// SW, SH are the logical screen size.
var SW, SH = 1000.0, 820.0

type Scene interface {
	Update()
	Draw(dst *ebiten.Image)
}

// Modal is a panel drawn over the current scene.
type Modal interface {
	Update() bool // returns false when it should close
	Draw(dst *ebiten.Image)
}

type Game struct {
	scene     Scene
	next      Scene
	fade      float64
	fadingOut bool
	modals    []Modal
	started   bool
	toast     string
	toastT    float64
}

var game *Game

// QuitRequested is set when the player quits from the menu (desktop).
var QuitRequested bool

func New(c Config) *Game {
	cfg = c
	game = &Game{}
	return game
}

func (g *Game) start() {
	g.started = true
	loadFonts()
	loadWords()
	loadSave()
	initAudio()
	if cfg.Mobile {
		musicHold = false
	}
	if cfg.CanUpdate && save.Settings.AutoCheckUpdates {
		updateChecker.check(false)
	}
	g.scene = newMenuScene()
	if !save.Settings.SeenHelp {
		g.Push(newHelpPanel())
		save.Settings.SeenHelp = true
		writeSave()
	}
}

func (g *Game) SetScene(s Scene) {
	g.next = s
	g.fadingOut = true
}

func (g *Game) Push(m Modal) { g.modals = append(g.modals, m) }

func (g *Game) Toast(msg string) {
	g.toast = msg
	g.toastT = 1
}

// Back handles the Android back button. It returns false when the app should close.
func (g *Game) Back() bool {
	if !g.started {
		return false
	}
	if _, ok := g.scene.(*menuScene); ok && len(g.modals) == 0 {
		return false
	}
	pendingBack = true
	return true
}

func (g *Game) Update() error {
	if !g.started {
		g.start()
	}
	if QuitRequested {
		writeSave()
		return ebiten.Termination
	}
	in.poll()
	if DebugFPS && in.Tick%120 == 0 {
		println("tps", int(ebiten.ActualTPS()), "fps", int(ebiten.ActualFPS()))
	}
	if in.Pressed && musicHold {
		musicHold = false
	}
	if !cfg.Mobile && fullscreenKeyJustPressed() {
		save.Settings.Fullscreen = !save.Settings.Fullscreen
		ebiten.SetFullscreen(save.Settings.Fullscreen)
		writeSave()
	}
	updateMusic()
	updateChecker.poll()
	if g.toastT > 0 {
		g.toastT -= 1.0 / 150
	}
	if g.fadingOut {
		g.fade = math.Min(1, g.fade+0.14)
		if g.fade >= 1 {
			g.scene, g.next, g.fadingOut = g.next, nil, false
		}
		return nil
	}
	if g.fade > 0 {
		g.fade = math.Max(0, g.fade-0.1)
	}
	if n := len(g.modals); n > 0 {
		top := g.modals[n-1]
		if !top.Update() {
			g.modals = g.modals[:n-1]
		}
		in.Consume()
		in.Chars = in.Chars[:0]
		in.Enter, in.Backspace, in.Back = false, false, false
	}
	if g.scene != nil {
		g.scene.Update()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.started {
		return
	}
	if g.scene != nil {
		g.scene.Draw(screen)
	}
	for _, m := range g.modals {
		m.Draw(screen)
	}
	if g.toastT > 0 && g.toast != "" {
		drawToast(screen, g.toast, g.toastT)
	}
	if g.fade > 0 {
		FillRect(screen, Rect{0, 0, SW, SH}, alpha(style().BgBot, g.fade))
	}
}

func drawToast(dst *ebiten.Image, msg string, t float64) {
	a := clamp(t*6, 0, 1)
	w := TextWidth(msg, Medium, 15) + 36
	y := math.Max(70, SH*0.25) + (1-easeOut(clamp((1-t)*8, 0, 1)))*-12
	r := Rect{SW/2 - w/2, y, w, 40}
	Shadow(dst, r, 20, a)
	FillRR(dst, r, 20, alpha(colText, a))
	Text(dst, msg, Medium, 15, r.CX(), r.CY()+1, alpha(colInk, a), AlignCenter)
}

func (g *Game) Layout(ow, oh int) (int, int) { panic("use LayoutF") }

func (g *Game) LayoutF(ow, oh float64) (float64, float64) {
	s := ebiten.Monitor().DeviceScaleFactor()
	if s <= 0 {
		s = 1
	}
	if s != S {
		faceCache = map[faceKey]*text.GoTextFace{}
	}
	S = s
	useAA = s < 1.75 && !ForceNoAA
	SW, SH = ow, oh
	return math.Ceil(ow * S), math.Ceil(oh * S)
}

var f11Was bool

func fullscreenKeyJustPressed() bool {
	down := ebiten.IsKeyPressed(ebiten.KeyF11)
	j := down && !f11Was
	f11Was = down
	return j
}

func loadFonts() {
	for i, n := range []string{"Poppins-Regular.ttf", "Poppins-Medium.ttf", "Poppins-Bold.ttf"} {
		b, err := assetFS.ReadFile("assets/fonts/" + n)
		if err != nil {
			continue
		}
		src, err := text.NewGoTextFaceSource(bytesReader(b))
		if err == nil {
			fontSources[i] = src
		}
	}
	for i := range fontSources {
		if fontSources[i] == nil {
			for j := range fontSources {
				if fontSources[j] != nil {
					fontSources[i] = fontSources[j]
				}
			}
		}
	}
}

// WindowIcons returns the app icon in a few sizes for the desktop window.
func WindowIcons() []image.Image {
	var out []image.Image
	for _, n := range []string{"icon_16.png", "icon_32.png", "icon_64.png", "icon_256.png"} {
		if im := loadPNG("assets/branding/" + n); im != nil {
			out = append(out, im)
		}
	}
	return out
}

// SetDataDir changes where the game saves (Android passes its files folder).
func SetDataDir(dir string) { cfg.DataDir = dir }

// DebugFPS logs the frame rate once a second (used by browser tests).
var DebugFPS bool

// ForceNoAA turns anti-aliasing off everywhere (browser tests use a software GPU).
var ForceNoAA bool
