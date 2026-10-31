package game

import (
	"bytes"
	"io"
	"io/fs"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const sampleRate = 44100

var (
	audioCtx   *audio.Context
	sfxData    = map[string][]byte{}
	musicPl    *audio.Player
	lastSFX    = map[string]int64{}
	musicFade  = 0.0
	musicPause bool
	musicHold  bool // music stays quiet until the first tap (needed in browsers)
)

func initAudio() {
	audioCtx = audio.NewContext(sampleRate)
	entries, _ := fs.ReadDir(assetFS, "assets/sfx")
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".ogg") {
			continue
		}
		b, err := assetFS.ReadFile("assets/sfx/" + e.Name())
		if err != nil {
			continue
		}
		s, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
		if err != nil {
			continue
		}
		pcm, err := io.ReadAll(s)
		if err != nil {
			continue
		}
		sfxData[strings.TrimSuffix(e.Name(), ".ogg")] = pcm
	}
}

func playSFX(name string) { playSFXv(name, 1) }

func playSFXv(name string, vol float64) {
	if audioCtx == nil || save.Settings.SFXMuted || save.Settings.SFXVolume <= 0 {
		return
	}
	b, ok := sfxData[name]
	if !ok {
		return
	}
	if t, ok := lastSFX[name]; ok && in.Tick-t < 2 {
		return
	}
	lastSFX[name] = in.Tick
	p := audioCtx.NewPlayerFromBytes(b)
	v := save.Settings.SFXVolume
	p.SetVolume(v * v * vol)
	p.Play()
}

func musicVolume() float64 {
	if save.Settings.MusicMuted || musicHold {
		return 0
	}
	v := save.Settings.MusicVolume
	return v * v * 0.7
}

// Each part of the game has its own song.
var musicFor = map[string]string{
	"menu":     "lobby-lights",
	"daily":    "after-hours",
	"practice": "velvet-tumblers",
	"zen":      "still-water",
	"theme":    "open-house",
}

var (
	wantTrack string // the song for the current screen
	curTrack  string // the song playing now
)

// setMusic switches to the song for a part of the game ("menu", "daily", ...).
// The current song fades out first.
func setMusic(part string) {
	if t, ok := musicFor[part]; ok {
		wantTrack = t
	}
}

func musicForVault(v *Vault) string {
	switch v.Kind {
	case Daily:
		return "daily"
	case ZenKind:
		return "zen"
	case ThemeKind:
		return "theme"
	}
	return "practice"
}

func startTrack(name string) {
	if musicPl != nil {
		musicPl.Close()
		musicPl = nil
	}
	curTrack = name
	b, err := assetFS.ReadFile("assets/music/" + name + ".ogg")
	if err != nil {
		return
	}
	s, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(b))
	if err != nil {
		return
	}
	p, err := audioCtx.NewPlayer(s)
	if err != nil {
		return
	}
	musicPl = p
	musicFade = 0
	p.SetVolume(0)
	p.Play()
}

// updateMusic fades between songs, loops the current one and applies the volume every tick.
func updateMusic() {
	if audioCtx == nil || wantTrack == "" {
		return
	}
	if musicVolume() <= 0 {
		if musicPl != nil && musicPl.IsPlaying() {
			musicPl.Pause()
			musicPause = true
		}
		return
	}
	if musicPl == nil {
		startTrack(wantTrack)
		return
	}
	if curTrack != wantTrack {
		// fade the old song out, then start the new one
		musicFade = approach(musicFade, 0, 0.06)
		musicPl.SetVolume(musicVolume() * musicFade)
		if musicFade < 0.02 {
			startTrack(wantTrack)
		}
		return
	}
	if !musicPl.IsPlaying() {
		if !musicPause {
			startTrack(curTrack) // loop
			return
		}
		musicPause = false
		musicPl.Play()
	}
	musicFade = approach(musicFade, 1, 0.02)
	musicPl.SetVolume(musicVolume() * musicFade)
}
