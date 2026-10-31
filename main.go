// Keyhole for Windows (and any desktop Ebiten supports). The game itself lives
// in package game so the Android build (package mobile) can share it.
package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/adamglassner/keyhole/game"
	"github.com/hajimehoshi/ebiten/v2"
)

// dataDir is %APPDATA%\Ironite\Keyhole on Windows (outside the install folder,
// so updates and reinstalls never touch saves).
func dataDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "Ironite", "Keyhole")
}

func main() {
	g := game.New(game.Config{
		DataDir:   dataDir(),
		CanUpdate: runtime.GOOS == "windows",
		Share: func(s string) string {
			if copyToClipboard(s) {
				return "Result copied. Paste it anywhere!"
			}
			return "Couldn't copy the result"
		},
	})
	game.DebugFPS = os.Getenv("KEYHOLE_FPS") != "" || runtime.GOOS == "js"
	game.ForceNoAA = runtime.GOOS == "js"
	ebiten.SetWindowTitle("Keyhole")
	ebiten.SetWindowSize(960, 860)
	ebiten.SetWindowSizeLimits(420, 680, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if icons := game.WindowIcons(); len(icons) > 0 {
		ebiten.SetWindowIcon(icons)
	}
	ebiten.SetTPS(60)
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetVsyncEnabled(true)
	if err := ebiten.RunGame(g); err != nil && err != ebiten.Termination {
		log.Fatal(err)
	}
}
