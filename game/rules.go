package game

import "github.com/adamglassner/keyhole/rules"

// Short local names for the rules package, which holds the testable game logic.
type (
	Vault     = rules.Vault
	Lock      = rules.Lock
	Mark      = rules.Mark
	PinResult = rules.PinResult
	VaultKind = rules.VaultKind
)

const (
	Silent = rules.Silent
	Wiggle = rules.Wiggle
	Click  = rules.Click

	Daily    = rules.Daily
	Practice = rules.Practice

	GuessOK       = rules.GuessOK
	GuessTooShort = rules.GuessTooShort
	GuessNotWord  = rules.GuessNotWord
	GuessRepeat   = rules.GuessRepeat

	RegionFront  = rules.RegionFront
	RegionMiddle = rules.RegionMiddle
	RegionBack   = rules.RegionBack

	TotalGuesses = rules.TotalGuesses
	GoldKeySpare = rules.GoldKeySpare
)

var (
	NewVault        = rules.NewVault
	DailyAnswers    = rules.DailyAnswers
	PracticeAnswers = rules.PracticeAnswers
	TodayNumber     = rules.TodayNumber
	UntilNextVault  = rules.UntilNextVault
	isValid         = rules.IsValid
	loadWords       = rules.Load
	itoa            = rules.Itoa
)
