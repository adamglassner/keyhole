// Package mobile is the Android build of Keyhole. It is compiled into an .aar
// with `ebitenmobile bind` and hosted by the small Java app in android/.
package mobile

import (
	"sync"

	"github.com/adamglassner/keyhole/game"
	"github.com/hajimehoshi/ebiten/v2/mobile"
)

var (
	mu       sync.Mutex
	share    string
	install  string
	dataDir  string
	instance *game.Game
)

func init() {
	instance = game.New(game.Config{
		Mobile:    true,
		CanUpdate: true,
		InstallUpdate: func(path string) {
			mu.Lock()
			install = path
			mu.Unlock()
		},
		Share: func(s string) string {
			mu.Lock()
			share = s
			mu.Unlock()
			return ""
		},
	})
	mobile.SetGame(instance)
}

// SetDataDir must be called by the activity before the game view starts.
func SetDataDir(dir string) {
	mu.Lock()
	dataDir = dir
	mu.Unlock()
	game.SetDataDir(dir)
}

// TakeShareText returns text the game wants to share (and clears it).
// The activity polls this and opens Android's share sheet.
func TakeShareText() string {
	mu.Lock()
	defer mu.Unlock()
	s := share
	share = ""
	return s
}

// TakeInstallPath returns a downloaded update APK the activity should install (and clears it).
func TakeInstallPath() string {
	mu.Lock()
	defer mu.Unlock()
	p := install
	install = ""
	return p
}

// InstallBlocked tells the game the player didn't allow installing apps.
func InstallBlocked() { game.UpdateBlocked() }

// OnBack handles the Android back button. It returns false when the activity should close.
func OnBack() bool { return instance.Back() }

// Dummy is required so gomobile always generates the Mobile class.
func Dummy() {}
