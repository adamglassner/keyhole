package rules

// The rules of Keyhole, kept free of drawing code so they can be unit tested.
//
// A vault has three locks: a 4-letter word, then 5, then 6. All three share one
// budget of guesses. Each guess marks every letter:
//   Click  - right letter, right spot
//   Wiggle - in the word, wrong spot
//   Silent - not in the word (or no copies left)
// When a lock opens, its letters become pins. Before the first guess on the
// next lock the player may test pins for free: the game says whether that
// letter is in the new word and roughly where (front, middle, back). Opening a
// lock in 3 guesses or fewer earns a second pin test.

import "strings"

const (
	TotalGuesses = 15
	CleanSolve   = 3 // open a lock in this many guesses or fewer for a bonus pin test
	GoldKeySpare = 6 // open the daily vault with this many guesses left for a gold key
)

var LockLengths = [3]int{4, 5, 6}

type Mark uint8

const (
	Silent Mark = iota
	Wiggle
	Click
)

// Score marks a guess against the answer, Wordle-style: exact matches first,
// then wrong-spot matches limited by how many copies of the letter are left.
func Score(guess, answer string) []Mark {
	n := len(answer)
	marks := make([]Mark, n)
	left := map[byte]int{}
	for i := 0; i < n; i++ {
		if guess[i] == answer[i] {
			marks[i] = Click
		} else {
			left[answer[i]]++
		}
	}
	for i := 0; i < n; i++ {
		if marks[i] == Click {
			continue
		}
		if left[guess[i]] > 0 {
			marks[i] = Wiggle
			left[guess[i]]--
		}
	}
	return marks
}

// Regions of a word for pin results.
const (
	RegionFront  = 1 << 0
	RegionMiddle = 1 << 1
	RegionBack   = 1 << 2
)

// regionOf says which part of a word of length n position i is in.
// 5 letters: front 1-2, middle 3, back 4-5. 6 letters: front 1-2, middle 3-4, back 5-6.
// 4 letters: front 1-2, back 3-4.
func regionOf(i, n int) int {
	switch {
	case i < 2:
		return RegionFront
	case i >= n-2:
		return RegionBack
	}
	return RegionMiddle
}

type PinResult struct {
	Letter  byte `json:"l"`
	Regions int  `json:"r"` // 0 = not in the word
	Count   int  `json:"c"`
}

func TestPinAgainst(letter byte, answer string) PinResult {
	r := PinResult{Letter: letter}
	for i := 0; i < len(answer); i++ {
		if answer[i] == letter {
			r.Regions |= regionOf(i, len(answer))
			r.Count++
		}
	}
	return r
}

// Describe gives a short human description of a pin result.
func (p PinResult) Describe() string {
	if p.Regions == 0 {
		return "not in this lock"
	}
	var parts []string
	if p.Regions&RegionFront != 0 {
		parts = append(parts, "front")
	}
	if p.Regions&RegionMiddle != 0 {
		parts = append(parts, "middle")
	}
	if p.Regions&RegionBack != 0 {
		parts = append(parts, "back")
	}
	s := "in the " + strings.Join(parts, " & ")
	if p.Count > 1 {
		s += " (twice)"
		if p.Count > 2 {
			s = "in the " + strings.Join(parts, " & ") + " (3 times)"
		}
	}
	return s
}

type Lock struct {
	Answer   string      `json:"a"`
	Guesses  []string    `json:"g"`
	Opened   bool        `json:"o"`
	PinTests int         `json:"pt"` // free pin tests available on this lock
	Pins     []PinResult `json:"p"`  // pins tested on this lock
}

func (l *Lock) Marks(i int) []Mark { return Score(l.Guesses[i], l.Answer) }

type VaultKind int

const (
	Daily VaultKind = iota
	Practice
)

type Vault struct {
	Kind   VaultKind `json:"kind"`
	Number int       `json:"num"` // daily vault number (1 = first day)
	Locks  [3]Lock   `json:"locks"`
	Cur    int       `json:"cur"`
	Used   int       `json:"used"`
	Done   bool      `json:"done"`
	Opened bool      `json:"opened"`
}

func NewVault(kind VaultKind, number int, answers [3]string) *Vault {
	v := &Vault{Kind: kind, Number: number}
	for i := range v.Locks {
		v.Locks[i].Answer = answers[i]
	}
	return v
}

func (v *Vault) Lock() *Lock           { return &v.Locks[v.Cur] }
func (v *Vault) Left() int             { return TotalGuesses - v.Used }
func (v *Vault) WordLen() int          { return len(v.Locks[v.Cur].Answer) }
func (v *Vault) GuessesUsed(i int) int { return len(v.Locks[i].Guesses) }

type GuessResult int

const (
	GuessOK GuessResult = iota
	GuessTooShort
	GuessNotWord
	GuessRepeat
	GuessDone
)

// Guess plays a word on the current lock. It returns what happened and, when
// accepted, whether that guess opened the lock.
func (v *Vault) Guess(word string, valid func(string) bool) (GuessResult, bool) {
	if v.Done {
		return GuessDone, false
	}
	l := v.Lock()
	word = strings.ToLower(word)
	if len(word) != len(l.Answer) {
		return GuessTooShort, false
	}
	for _, g := range l.Guesses {
		if g == word {
			return GuessRepeat, false
		}
	}
	if word != l.Answer && !valid(word) {
		return GuessNotWord, false
	}
	l.Guesses = append(l.Guesses, word)
	v.Used++
	if word == l.Answer {
		l.Opened = true
		if v.Cur == len(v.Locks)-1 {
			v.Done, v.Opened = true, true
		}
		return GuessOK, true
	}
	if v.Used >= TotalGuesses {
		v.Done = true
	}
	return GuessOK, false
}

// Advance moves to the next lock after one opens and grants its pin tests.
func (v *Vault) Advance() {
	if v.Done || !v.Lock().Opened || v.Cur >= len(v.Locks)-1 {
		return
	}
	prev := v.Lock()
	v.Cur++
	n := v.Lock()
	n.PinTests = 1
	if len(prev.Guesses) <= CleanSolve {
		n.PinTests = 2
	}
}

// AvailablePins lists the letters carried over from the last lock that can
// still be tested on this one (only before the first guess).
func (v *Vault) AvailablePins() []byte {
	if v.Cur == 0 || v.Done {
		return nil
	}
	l := v.Lock()
	if len(l.Guesses) > 0 || len(l.Pins) >= l.PinTests {
		return nil
	}
	return v.CarriedPins()
}

// CarriedPins are the distinct letters of the previous lock's answer, minus any already tested.
func (v *Vault) CarriedPins() []byte {
	if v.Cur == 0 {
		return nil
	}
	prev := v.Locks[v.Cur-1].Answer
	l := v.Lock()
	var out []byte
	for i := 0; i < len(prev); i++ {
		c := prev[i]
		dup := false
		for _, o := range out {
			if o == c {
				dup = true
			}
		}
		for _, p := range l.Pins {
			if p.Letter == c {
				dup = true
			}
		}
		if !dup {
			out = append(out, c)
		}
	}
	return out
}

func (v *Vault) TestPin(c byte) (PinResult, bool) {
	ok := false
	for _, p := range v.AvailablePins() {
		if p == c {
			ok = true
		}
	}
	if !ok {
		return PinResult{}, false
	}
	l := v.Lock()
	r := TestPinAgainst(c, l.Answer)
	l.Pins = append(l.Pins, r)
	return r, true
}

func (v *Vault) PinTestsLeft() int {
	if len(v.AvailablePins()) == 0 {
		return 0
	}
	l := v.Lock()
	return l.PinTests - len(l.Pins)
}

// Spare is how many guesses were left when the vault opened.
func (v *Vault) Spare() int {
	if !v.Opened {
		return 0
	}
	return TotalGuesses - v.Used
}

// KeyState is the best knowledge about each letter on the current lock, for the keyboard.
func (v *Vault) KeyState() map[byte]int {
	st := map[byte]int{} // 0 unknown, 1 silent, 2 known-in (pin), 3 wiggle, 4 click
	l := v.Lock()
	for _, p := range l.Pins {
		if p.Regions == 0 {
			st[p.Letter] = 1
		} else {
			st[p.Letter] = 2
		}
	}
	for i, g := range l.Guesses {
		m := l.Marks(i)
		for j := 0; j < len(g); j++ {
			c := g[j]
			s := 1
			switch m[j] {
			case Click:
				s = 4
			case Wiggle:
				s = 3
			}
			if s == 1 && st[c] >= 2 {
				continue
			}
			if s > st[c] || st[c] == 0 {
				st[c] = s
			}
		}
	}
	return st
}

// ShareText builds the spoiler-free result to paste into a chat.
func (v *Vault) ShareText() string {
	var b strings.Builder
	title := "Keyhole Practice"
	if v.Kind == Daily {
		title = "Keyhole #" + Itoa(v.Number)
	}
	if v.Opened {
		b.WriteString("🔓 " + title + " · opened, " + Itoa(v.Spare()) + " to spare")
		if v.Kind == Daily && v.Spare() >= GoldKeySpare {
			b.WriteString(" 🔑")
		}
	} else {
		b.WriteString("🔒 " + title + " · stuck at lock " + Itoa(v.Cur+1))
	}
	nums := []string{"①", "②", "③"}
	for i := 0; i <= v.Cur && i < 3; i++ {
		l := &v.Locks[i]
		b.WriteString("\n" + nums[i])
		for gi := range l.Guesses {
			b.WriteString(" ")
			for _, m := range l.Marks(gi) {
				switch m {
				case Click:
					b.WriteString("🟨")
				case Wiggle:
					b.WriteString("🟧")
				default:
					b.WriteString("⬛")
				}
			}
		}
	}
	return b.String()
}

// Itoa formats an int (kept here to avoid strconv in hot draw code).
func Itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	if neg {
		return "-" + string(d)
	}
	return string(d)
}
