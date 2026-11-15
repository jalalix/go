package jalalix_test

import (
	"testing"

	jalalix "github.com/jalalix/go"
)

func TestFromPersianArabicIndicDigits(t *testing.T) {
	if got := jalalix.FromPersian("۱۴۰۳/۱۲/۰۷"); got != "1403/12/07" {
		t.Errorf("persian digits: got %q want %q", got, "1403/12/07")
	}

	if got := jalalix.FromPersian("١٤٠٣/١٢/٠٧"); got != "1403/12/07" {
		t.Errorf("arabic-indic digits: got %q want %q", got, "1403/12/07")
	}
}

func TestParseJalaliDotSeparator(t *testing.T) {
	d, err := jalalix.ParseJalali("1403.12.07")

	if err != nil {
		t.Fatalf("parseJalali: %v", err)
	}

	if got := d.ToISODate(); got != "2025-02-25" {
		t.Errorf("got %q want %q", got, "2025-02-25")
	}
}

func TestNormalizeQueryDateSentinelsAndTimestamps(t *testing.T) {
	cases := []any{nil, "null", "", "undefined", "Invalid Date", "NaN", 0, -1}

	for _, c := range cases {
		if got := jalalix.NormalizeQueryDate(c); got != nil {
			t.Errorf("NormalizeQueryDate(%#v): got %#v want nil", c, got)
		}
	}

	const seconds = 1740430800 // 2025-02-24 21:00:00 UTC / 2025-02-25 00:30:00 Tehran

	if got := jalalix.NormalizeQueryDate(seconds, jalalix.QueryDateOptions{Timezone: "UTC"}); got != "2025-02-24 21:00:00" {
		t.Errorf("seconds UTC: got %#v", got)
	}

	if got := jalalix.NormalizeQueryDate(int64(seconds)*1000, jalalix.QueryDateOptions{Timezone: "Asia/Tehran"}); got != "2025-02-25 00:30:00" {
		t.Errorf("ms Tehran: got %#v", got)
	}
}

func TestNormalizeQueryDateJalaliAndGregorian(t *testing.T) {
	if got := jalalix.NormalizeQueryDate("1403/12/07", jalalix.QueryDateOptions{Timezone: "UTC"}); got != "2025-02-25 00:00:00" {
		t.Errorf("jalali slash: got %#v", got)
	}

	if got := jalalix.NormalizeQueryDate("۱۴۰۳-۱۲-۰۷ 14:30:00", jalalix.QueryDateOptions{Timezone: "UTC"}); got != "2025-02-25 14:30:00" {
		t.Errorf("jalali persian digits + time: got %#v", got)
	}

	if got := jalalix.NormalizeQueryDate("2025.02.25"); got != "2025-02-25 00:00:00" {
		t.Errorf("gregorian dot separator: got %#v", got)
	}

	if got := jalalix.NormalizeQueryDate("2025-02-25 12:30:45.123Z"); got != "2025-02-25 12:30:45" {
		t.Errorf("gregorian frac seconds + Z: got %#v", got)
	}

	if got := jalalix.NormalizeQueryDate("not-a-date"); got != "not-a-date" {
		t.Errorf("cleaned fallback: got %#v", got)
	}

	if got := jalalix.NormalizeQueryDate("1200/01/01", jalalix.QueryDateOptions{Timezone: "UTC"}); got == nil {
		t.Errorf("wide jalali band 1200: got nil")
	}

	if got := jalalix.NormalizeQueryDate("1800/01/01", jalalix.QueryDateOptions{Timezone: "UTC"}); got == nil {
		t.Errorf("wide jalali band 1800: got nil")
	}

	if got := jalalix.NormalizeQueryDate("2025-01-01"); got != "2025-01-01 00:00:00" {
		t.Errorf("gregorian year stays gregorian: got %#v", got)
	}

	if got := jalalix.NormalizeQueryDate("2025-01-01", jalalix.QueryDateOptions{
		Timezone:      "UTC",
		JalaliYearMin: 1,
		JalaliYearMax: 3000,
	}); got == "2025-01-01 00:00:00" {
		t.Errorf("overridden jalaliYearMax should treat 2025 as jalali: got %#v", got)
	}
}
