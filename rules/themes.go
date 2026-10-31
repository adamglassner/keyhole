package rules

import "strings"

// ThemeDef is a topic for Theme Vaults: every answer in the vault fits it.
type ThemeDef struct {
	Name  string
	Blurb string
	words string // space-separated, any of the lengths 4, 5 and 6
}

// Themes lists every topic. Each needs plenty of 4-, 5- and 6-letter words.
var Themes = []ThemeDef{
	{"Animals", "Furry, feathered and scaly friends.", `
		bear deer wolf lion frog duck goat lamb seal swan crab mole hare toad mule moth wasp newt puma orca calf colt hawk crow dove fish bull lynx
		horse tiger zebra camel sheep otter panda koala eagle shark whale mouse moose llama snake skunk raven robin finch goose bison hyena lemur sloth squid gecko rhino
		rabbit monkey turtle donkey beaver badger parrot falcon jaguar walrus ferret gopher iguana lizard salmon spider weasel turkey kitten pigeon alpaca beetle cougar coyote oyster`},
	{"Food", "Snacks, suppers and sweet treats.", `
		cake rice soup bean corn pear plum lime meat milk beef pork tuna kiwi taco salt chip mint kale dill tofu stew
		bread apple grape lemon mango melon peach pizza pasta salad bacon toast cream honey sugar olive onion fudge candy donut curry sushi wafer crepe gravy chili berry cocoa
		banana cherry cookie muffin butter cheese pepper carrot potato tomato noodle waffle burger pickle garlic orange walnut peanut yogurt raisin celery radish turnip cereal sundae toffee`},
	{"Nature", "Forests, rivers and wide open skies.", `
		tree leaf rock lake hill sand moss fern pond wave wind rain snow star moon seed root bark cave reef vine mist dune
		river ocean beach cliff field grass stone cloud storm petal bloom creek marsh coral earth trail ridge shore woods grove frost brook maple cedar birch acorn
		forest meadow canyon valley island desert jungle flower garden branch breeze sunset pebble spruce willow autumn spring summer winter lagoon tundra pollen nectar`},
	{"Halloween", "Ghosts, goblins and things that go bump.", `
		fang moon mask cape wand howl tomb ogre hood bone mist toad wolf dark fear grim
		ghost witch broom candy scary spook ghoul crypt skull spell haunt eerie raven magic cloak trick treat night beast attic
		goblin candle spider potion shadow cackle fright zombie cobweb creepy spooky wizard scream hollow`},
	{"Space", "Rockets, planets and the stars beyond.", `
		star moon dust void beam ring halo lens atom glow nova rock dark
		comet orbit earth alien rover laser solar lunar space flare probe light world
		rocket galaxy nebula cosmos planet meteor launch quasar crater zenith module signal voyage astral photon oxygen`},
	{"Music", "Songs, sounds and instruments.", `
		song drum harp horn band beat tune note solo duet bass lute alto jazz rock folk hymn lyre gong tone
		piano flute cello banjo organ opera chord tempo choir music lyric blues disco radio album track remix viola tenor dance sound carol
		guitar violin rhythm melody singer chorus sonata ballad record stereo encore octave string anthem fiddle cymbal`},
	{"Sports & Games", "Goals, gold medals and game night.", `
		golf ball goal race swim kick pass team game dice card jump dart puck bowl lane polo judo surf hoop
		chess rugby score skate relay match medal coach field court serve throw poker bingo catch pitch
		soccer tennis hockey karate boxing trophy sprint racket skiing player puzzle rowing archer dugout umpire league tackle runner`},
}

// Words returns the theme's words of length n.
func (t *ThemeDef) Words(n int) []string {
	var out []string
	for _, w := range strings.Fields(t.words) {
		if len(w) == n {
			out = append(out, w)
		}
	}
	return out
}

// ThemeByName finds a theme (nil if there's no such theme).
func ThemeByName(name string) *ThemeDef {
	for i := range Themes {
		if Themes[i].Name == name {
			return &Themes[i]
		}
	}
	return nil
}

// ThemeAnswers picks three random answers (4, 5 and 6 letters) from a theme.
func ThemeAnswers(t *ThemeDef) [3]string {
	var out [3]string
	for i, n := range LockLengths {
		ws := t.Words(n)
		out[i] = ws[practiceRng.intn(len(ws))]
	}
	return out
}

// ZenAnswers picks a relaxed vault from the common answer lists.
func ZenAnswers() [3]string { return PracticeAnswers() }
