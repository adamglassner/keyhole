package game

import (
	"bytes"
	"io"
	"io/fs"
	"math/rand"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const sampleRate = 44100

var (
	audioCtx   *audio.Context
	sfxData    = map[string][]byte{}
	musicFiles []string
	musicIdx   int
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
	mus, _ := fs.ReadDir(assetFS, "assets/music")
	for _, e := range mus {
		if strings.HasSuffix(e.Name(), ".ogg") {
			musicFiles = append(musicFiles, "assets/music/"+e.Name())
		}
	}
	sort.Strings(musicFiles)
	if len(musicFiles) > 0 {
		musicIdx = rand.Intn(len(musicFiles))
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

func startTrack(i int) {
	if len(musicFiles) == 0 {
		return
	}
	if musicPl != nil {
		musicPl.Close()
		musicPl = nil
	}
	musicIdx = (i%len(musicFiles) + len(musicFiles)) % len(musicFiles)
	b, err := assetFS.ReadFile(musicFiles[musicIdx])
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

// updateMusic keeps the playlist going and applies the volume every tick.
func updateMusic() {
	if audioCtx == nil || len(musicFiles) == 0 {
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
		startTrack(musicIdx)
		return
	}
	if !musicPl.IsPlaying() {
		if !musicPause {
			startTrack(musicIdx + 1)
			return
		}
		musicPause = false
		musicPl.Play()
	}
	musicFade = approach(musicFade, 1, 0.02)
	musicPl.SetVolume(musicVolume() * musicFade)
}
