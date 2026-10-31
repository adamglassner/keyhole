package rules

import (
	"strings"
	"testing"
	"time"
)

func marks(s string) []Mark {
	var m []Mark
	for _, c := range s {
		switch c {
		case 'C':
			m = append(m, Click)
		case 'W':
			m = append(m, Wiggle)
		default:
			m = append(m, Silent)
		}
	}
	return m
}

func TestScore(t *testing.T) {
	cases := []struct{ g, a, want string }{
		{"crane", "crane", "CCCCC"},
		{"spool", "loops", "WWCWW"},
		{"eerie", "there", "W.W.C"},
		{"llama", "hello", "WW..."},
		{"speed", "abide", "..W.W"},
		{"geese", "eagle", "WW..C"},
		{"aaaa", "bark", ".C.."},
	}
	for _, c := range cases {
		got := Score(c.g, c.a)
		want := marks(c.want)
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("Score(%s,%s) = %v want %s", c.g, c.a, got, c.want)
				break
			}
		}
	}
}

func TestPins(t *testing.T) {
	r := TestPinAgainst('e', "letter")
	if r.Regions != RegionFront|RegionBack || r.Count != 2 {
		t.Fatalf("pin e in letter = %+v", r)
	}
	if TestPinAgainst('z', "apple").Regions != 0 {
		t.Fatal("z should not be in apple")
	}
	if regionOf(2, 5) != RegionMiddle || regionOf(3, 6) != RegionMiddle || regionOf(2, 4) != RegionBack {
		t.Fatal("regions")
	}
}

func TestVaultFlow(t *testing.T) {
	Load()
	v := NewVault(Daily, 1, [3]string{"bolt", "crane", "silver"})
	if r, _ := v.Guess("zzzz", IsValid); r != GuessNotWord {
		t.Fatal("zzzz accepted")
	}
	if r, _ := v.Guess("abc", IsValid); r != GuessTooShort {
		t.Fatal("short accepted")
	}
	v.Guess("cold", IsValid)
	if r, _ := v.Guess("cold", IsValid); r != GuessRepeat {
		t.Fatal("repeat accepted")
	}
	if _, opened := v.Guess("bolt", IsValid); !opened {
		t.Fatal("bolt should open")
	}
	v.Advance()
	if v.Cur != 1 || v.Lock().PinTests != 2 {
		t.Fatalf("cur=%d pintests=%d", v.Cur, v.Lock().PinTests)
	}
	pins := string(v.AvailablePins())
	if pins != "bolt" {
		t.Fatalf("pins %q", pins)
	}
	if _, ok := v.TestPin('z'); ok {
		t.Fatal("z isn't a pin")
	}
	r, _ := v.TestPin('o')
	if r.Regions != 0 {
		t.Fatal("o not in crane")
	}
	v.TestPin('t')
	if v.PinTestsLeft() != 0 || len(v.AvailablePins()) != 0 {
		t.Fatal("should be out of pin tests")
	}
	v.Guess("crane", IsValid)
	v.Advance()
	if v.Lock().PinTests != 2 {
		t.Fatal("clean solve should give 2 tests")
	}
	for _, g := range []string{"potato", "banana", "carpet", "dragon", "violet", "number", "pencil", "garden", "window", "rocket"} {
		v.Guess(g, IsValid)
	}
	if v.Left() != 2 || v.Done {
		t.Fatalf("left %d done %v", v.Left(), v.Done)
	}
	v.Guess("silver", IsValid)
	if !v.Opened || v.Spare() != 1 {
		t.Fatalf("opened %v spare %d", v.Opened, v.Spare())
	}
	s := v.ShareText()
	if !strings.HasPrefix(s, "🔓 Keyhole #1") || strings.Count(s, "\n") != 3 {
		t.Fatalf("share text:\n%s", s)
	}
}

func TestOutOfGuesses(t *testing.T) {
	Load()
	v := NewVault(Practice, 0, [3]string{"bolt", "crane", "silver"})
	words := []string{"able", "acid", "aged", "also", "area", "army", "away", "baby", "back", "ball", "band", "bank", "base", "bath", "bear"}
	for _, w := range words {
		v.Guess(w, IsValid)
	}
	if !v.Done || v.Opened {
		t.Fatal("should be sealed")
	}
}

func TestDailyStable(t *testing.T) {
	Load()
	a := DailyAnswers(1)
	for i, n := range LockLengths {
		if len(a[i]) != n {
			t.Fatalf("lock %d len %d", i, len(a[i]))
		}
		if !IsValid(a[i]) {
			t.Fatalf("%s not valid", a[i])
		}
	}
	if DailyAnswers(1) != a || DailyAnswers(2) == a {
		t.Fatal("daily answers not stable")
	}
	NowFunc = func() time.Time { return time.Date(2026, 11, 2, 23, 59, 0, 0, time.Local) }
	defer func() { NowFunc = time.Now }()
	if TodayNumber() != 3 {
		t.Fatalf("today %d", TodayNumber())
	}
}

func TestThemes(t *testing.T) {
	Load()
	for i := range Themes {
		th := &Themes[i]
		seen := map[string]bool{}
		for _, n := range LockLengths {
			ws := th.Words(n)
			if len(ws) < 10 {
				t.Errorf("%s has only %d words of %d letters", th.Name, len(ws), n)
			}
			for _, w := range ws {
				if !IsValid(w) {
					t.Errorf("%s: %q isn't in the word list", th.Name, w)
				}
				if seen[w] {
					t.Errorf("%s: %q listed twice", th.Name, w)
				}
				seen[w] = true
			}
		}
		a := ThemeAnswers(th)
		for i, n := range LockLengths {
			if len(a[i]) != n {
				t.Fatalf("theme answer %q wrong length", a[i])
			}
		}
	}
}

func TestZenNeverRunsOut(t *testing.T) {
	Load()
	v := NewVault(Zen, 0, [3]string{"bolt", "crane", "silver"})
	words := []string{"able", "acid", "aged", "also", "area", "army", "away", "baby", "back", "ball", "band", "bank", "base", "bath", "bear", "beat", "been", "beer"}
	for _, w := range words {
		v.Guess(w, IsValid)
	}
	if v.Done || v.Left() <= 0 {
		t.Fatal("zen vault should never run out")
	}
	v.Guess("bolt", IsValid)
	v.Advance()
	v.Guess("crane", IsValid)
	v.Advance()
	v.Guess("silver", IsValid)
	if !v.Opened || v.Used != 21 {
		t.Fatalf("opened %v used %d", v.Opened, v.Used)
	}
	if !strings.Contains(v.ShareText(), "opened in 21 guesses") {
		t.Fatal(v.ShareText())
	}
}
