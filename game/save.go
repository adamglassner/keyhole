package game

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	MusicVolume      float64 `json:"musicVolume"`
	SFXVolume        float64 `json:"sfxVolume"`
	MusicMuted       bool    `json:"musicMuted"`
	SFXMuted         bool    `json:"sfxMuted"`
	Fullscreen       bool    `json:"fullscreen"`
	HighContrast     bool    `json:"highContrast"`
	LockStyle        int     `json:"lockStyle"`
	SeenHelp         bool    `json:"seenHelp"`
	AutoCheckUpdates bool    `json:"autoCheckUpdates"`
	SkippedVersion   string  `json:"skippedVersion"`
}

type Stats struct {
	Played     int     `json:"played"`
	Opened     int     `json:"opened"`
	CurStreak  int     `json:"curStreak"`
	MaxStreak  int     `json:"maxStreak"`
	LastPlayed int     `json:"lastPlayed"` // number of the last finished daily vault
	LastOpened int     `json:"lastOpened"`
	Spare      [16]int `json:"spare"` // daily vaults opened with n guesses to spare
	GoldKeys   int     `json:"goldKeys"`
	PracPlayed int     `json:"practicePlayed"`
	PracOpened int     `json:"practiceOpened"`
	PracBest   int     `json:"practiceBest"`
	ZenPlayed  int     `json:"zenPlayed"`
	ZenBest    int     `json:"zenBest"` // fewest guesses to open a Zen vault
	ThemePlay  int     `json:"themePlayed"`
	ThemeOpen  int     `json:"themeOpened"`
}

type SaveData struct {
	Version  int      `json:"version"`
	Settings Settings `json:"settings"`
	Stats    Stats    `json:"stats"`
	Daily    *Vault   `json:"daily,omitempty"`
	Practice *Vault   `json:"practice,omitempty"`
	Zen      *Vault   `json:"zen,omitempty"`
	Themed   *Vault   `json:"themed,omitempty"`
}

var save = SaveData{Version: 1, Settings: defaultSettings()}

func defaultSettings() Settings {
	return Settings{MusicVolume: 0.5, SFXVolume: 0.8, AutoCheckUpdates: true}
}

func savePath() string {
	if cfg.DataDir == "" {
		return ""
	}
	return filepath.Join(cfg.DataDir, "keyhole.json")
}

func loadSave() {
	p := savePath()
	if p == "" {
		return
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return
	}
	s := SaveData{Settings: defaultSettings()}
	if json.Unmarshal(b, &s) != nil {
		_ = os.Rename(p, p+".bad")
		return
	}
	save = s
	for _, v := range []*Vault{save.Daily, save.Practice, save.Zen, save.Themed} {
		if v != nil && !v.Done && v.Lock().Opened {
			v.Advance() // the app closed during the unlock animation
		}
	}
}

func writeSave() {
	p := savePath()
	if p == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	b, err := json.MarshalIndent(save, "", " ")
	if err != nil {
		return
	}
	tmp := p + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, p)
	}
}

// Streak shown to the player: it lapses once a whole day is missed.
func currentStreak() int {
	if save.Stats.LastOpened < TodayNumber()-1 {
		return 0
	}
	return save.Stats.CurStreak
}

// recordFinish updates stats when a vault is finished. It returns true if a gold key was earned.
func recordFinish(v *Vault) bool {
	st := &save.Stats
	switch v.Kind {
	case ZenKind:
		st.ZenPlayed++
		if st.ZenBest == 0 || v.Used < st.ZenBest {
			st.ZenBest = v.Used
		}
		writeSave()
		return false
	case ThemeKind:
		st.ThemePlay++
		if v.Opened {
			st.ThemeOpen++
		}
		writeSave()
		return false
	}
	if v.Kind == Practice {
		st.PracPlayed++
		if v.Opened {
			st.PracOpened++
			if v.Spare() > st.PracBest {
				st.PracBest = v.Spare()
			}
		}
		writeSave()
		return false
	}
	if v.Number <= st.LastPlayed {
		return false // already counted
	}
	st.Played++
	st.LastPlayed = v.Number
	key := false
	if v.Opened {
		st.Opened++
		if st.LastOpened == v.Number-1 {
			st.CurStreak++
		} else {
			st.CurStreak = 1
		}
		st.LastOpened = v.Number
		if st.CurStreak > st.MaxStreak {
			st.MaxStreak = st.CurStreak
		}
		sp := v.Spare()
		if sp >= len(st.Spare) {
			sp = len(st.Spare) - 1
		}
		st.Spare[sp]++
		if v.Spare() >= GoldKeySpare {
			st.GoldKeys++
			key = true
		}
	} else {
		st.CurStreak = 0
	}
	writeSave()
	return key
}

// todaysVault returns today's daily vault, creating it if needed.
func todaysVault() *Vault {
	n := TodayNumber()
	if save.Daily == nil || save.Daily.Number != n {
		save.Daily = NewVault(Daily, n, DailyAnswers(n))
	}
	return save.Daily
}
