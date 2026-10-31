package rules

import (
	"embed"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed words
var wordFS embed.FS

var (
	answers    [3][]string // 4, 5, 6 letters
	validWords []string    // sorted
)

var loadOnce sync.Once

// Load reads the word lists (safe to call more than once).
func Load() { loadOnce.Do(loadWords) }

func loadWords() {
	for i, n := range LockLengths {
		b, _ := wordFS.ReadFile("words/answers-" + Itoa(n) + ".txt")
		answers[i] = strings.Fields(string(b))
	}
	b, _ := wordFS.ReadFile("words/valid.txt")
	validWords = strings.Fields(string(b))
	sort.Strings(validWords)
}

// IsValid reports whether w is an accepted guess.
func IsValid(w string) bool {
	i := sort.SearchStrings(validWords, w)
	return i < len(validWords) && validWords[i] == w
}

// ---- daily vaults

// The first daily vault (#1) is on this date. Everyone gets the same vault on the
// same local calendar day.
var firstDay = time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)

// nowFunc can be replaced in tests.
var NowFunc = time.Now

// TodayNumber is today's daily vault number in the player's local time zone.
func TodayNumber() int {
	t := NowFunc()
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return int(d.Sub(firstDay).Hours()/24) + 1
}

// UntilNextVault is the time left until local midnight.
func UntilNextVault() time.Duration {
	t := NowFunc()
	next := time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
	return next.Sub(t)
}

// splitmix64: a small PRNG whose output never changes between Go versions,
// so daily answers are the same for every player and every build.
type rng struct{ s uint64 }

func (r *rng) next() uint64 {
	r.s += 0x9E3779B97F4A7C15
	z := r.s
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}
func (r *rng) intn(n int) int { return int(r.next() % uint64(n)) }

func shuffled(words []string, seed uint64) []string {
	out := append([]string(nil), words...)
	r := &rng{seed}
	for i := len(out) - 1; i > 0; i-- {
		j := r.intn(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

var dailyOrder [3][]string

// DailyAnswers returns the three answers for daily vault n.
func DailyAnswers(n int) [3]string {
	var out [3]string
	for i := range out {
		if dailyOrder[i] == nil {
			dailyOrder[i] = shuffled(answers[i], 0x6B657968+uint64(LockLengths[i])*7919)
		}
		list := dailyOrder[i]
		k := (n - 1) % len(list)
		if k < 0 {
			k += len(list)
		}
		out[i] = list[k]
	}
	return out
}

var practiceRng = &rng{uint64(time.Now().UnixNano())}

func PracticeAnswers() [3]string {
	var out [3]string
	for i := range out {
		out[i] = answers[i][practiceRng.intn(len(answers[i]))]
	}
	return out
}
