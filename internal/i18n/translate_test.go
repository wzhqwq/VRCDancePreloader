package i18n

import (
	"testing"
	"time"
)

// A2 — time.Month is 1..12 while the translated list is 0 based, so December
// used to index one past the end of the slice and panic.
func TestParseMonthCoversEveryMonth(t *testing.T) {
	originalLang, originalTranslations := lang, dateTranslations
	t.Cleanup(func() {
		lang, dateTranslations = originalLang, originalTranslations
	})

	abbr := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	lang = "en"
	dateTranslations = DateTranslations{MonthsAbbr: abbr}

	for i, want := range abbr {
		month := time.Month(i + 1)
		if got := ParseMonth(month); got != want {
			t.Fatalf("ParseMonth(%d) = %q, want %q", month, got, want)
		}
	}
}

// Without an English translation the month is printed as a number, and that path
// must not touch the slice at all (it may be empty).
func TestParseMonthWithoutEnglish(t *testing.T) {
	originalLang, originalTranslations := lang, dateTranslations
	t.Cleanup(func() {
		lang, dateTranslations = originalLang, originalTranslations
	})

	lang = "zh"
	dateTranslations = DateTranslations{}

	if got := ParseMonth(time.December); got != "12" {
		t.Fatalf(`ParseMonth(December) = %q, want "12"`, got)
	}
}
