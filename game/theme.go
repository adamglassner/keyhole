package game

import "image/color"

func hex(s string) color.RGBA {
	var v [3]uint8
	for i := 0; i < 3; i++ {
		v[i] = hexByte(s[1+i*2], s[2+i*2])
	}
	return color.RGBA{v[0], v[1], v[2], 255}
}

func hexByte(a, b byte) uint8 {
	h := func(c byte) uint8 {
		switch {
		case c >= '0' && c <= '9':
			return c - '0'
		case c >= 'a' && c <= 'f':
			return c - 'a' + 10
		case c >= 'A' && c <= 'F':
			return c - 'A' + 10
		}
		return 0
	}
	return h(a)<<4 | h(b)
}

// alpha returns c at the given opacity (premultiplied, as Ebiten expects).
func alpha(c color.RGBA, a float64) color.RGBA {
	f := clamp(a, 0, 1) * float64(c.A) / 255
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), uint8(255 * f)}
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	t = clamp(t, 0, 1)
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), l(a.A, b.A)}
}

func lighten(c color.RGBA, t float64) color.RGBA { return lerpColor(c, colWhite, t) }
func darken(c color.RGBA, t float64) color.RGBA  { return lerpColor(c, colBlack, t) }

var (
	colWhite = color.RGBA{255, 255, 255, 255}
	colBlack = color.RGBA{0, 0, 0, 255}

	colText    = hex("#F2E8D5")
	colTextDim = hex("#9AA8A6")
	colInk     = hex("#1A1206") // dark text on gold
	colSurface = hex("#15232A")
	colSurf2   = hex("#1D3038")
	colBorder  = hex("#2E434A")
	colDanger  = hex("#E0564B")
	colGood    = hex("#6FCF8F")

	colGold     = hex("#E2B13C")
	colGoldHi   = hex("#F6DC8E")
	colGoldDark = hex("#8A6420")
)

// Tile colors for the three marks. High contrast swaps to orange/blue.
type markColors struct{ Click, Wiggle, Silent, ClickText, WiggleText, SilentText color.RGBA }

func marksPalette() markColors {
	if save.Settings.HighContrast {
		return markColors{hex("#F5793A"), hex("#85C0F9"), hex("#3A4648"), hex("#1A1206"), hex("#0D1B2A"), hex("#C9D2D0")}
	}
	return markColors{hex("#E2B13C"), hex("#C8653A"), hex("#33403F"), hex("#1A1206"), hex("#FFF4E6"), hex("#A9B6B4")}
}

// LockStyle is a cosmetic look for the vault, unlocked with gold keys.
type LockStyle struct {
	Name, Blurb             string
	Keys                    int
	Metal, MetalHi, MetalLo color.RGBA
	BgTop, BgBot, Plate     color.RGBA
}

var lockStyles = []LockStyle{
	{"Brass Padlock", "The trusty classic.", 0, hex("#C9922E"), hex("#F6DC8E"), hex("#6E4A14"), hex("#0F1C1F"), hex("#081012"), hex("#2A3A3C")},
	{"Steel Safe", "Cold, quiet and heavy.", 2, hex("#9FAAB4"), hex("#E6ECF0"), hex("#4A545C"), hex("#121821"), hex("#090C11"), hex("#2B333D")},
	{"Copper Strongbox", "Warm, worn and stubborn.", 5, hex("#C46A3A"), hex("#F2B48A"), hex("#5E2C14"), hex("#1C1210"), hex("#0D0807"), hex("#3A2621")},
	{"Bank Vault", "Behind the teller's counter.", 10, hex("#8FA79A"), hex("#DDEDE3"), hex("#3C5246"), hex("#0D1A14"), hex("#06100B"), hex("#22352B")},
	{"Treasure Chest", "Pirate gold, sea-salted.", 16, hex("#E8B931"), hex("#FFE9A0"), hex("#7A5410"), hex("#21160A"), hex("#110B04"), hex("#4A3418")},
	{"Royal Vault", "Fit for the crown jewels.", 25, hex("#D9C27A"), hex("#FFF3C4"), hex("#6A5520"), hex("#1A1030"), hex("#0B0618"), hex("#33264F")},
}

func style() *LockStyle {
	i := save.Settings.LockStyle
	if i < 0 || i >= len(lockStyles) {
		i = 0
	}
	return &lockStyles[i]
}

type colorRGBA = color.RGBA
