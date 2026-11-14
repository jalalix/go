package jalalix_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	jalalix "github.com/jalalix/go"
)

type goldenRoot struct {
	Calendar  []goldenCalendar  `json:"calendar"`
	Parse     []goldenParse     `json:"parse"`
	Format    []goldenFormat    `json:"format"`
	Timezone  []goldenTimezone  `json:"timezone"`
	Mutations []goldenMutation  `json:"mutations"`
}

type createSpec struct {
	ISODate  string `json:"isoDate"`
	UnixMs   *int64 `json:"unixMs"`
	Timezone string `json:"timezone"`
}

type goldenCalendar struct {
	ID     string     `json:"id"`
	Create createSpec `json:"create"`
	Expect struct {
		FormatYYYYMMDD     string `json:"formatYYYYMMDD"`
		ToISODateGregorian string `json:"toISODateGregorian"`
		ToISODateJalali    string `json:"toISODateJalali"`
	} `json:"expect"`
}

type goldenParse struct {
	ID     string `json:"id"`
	Input  string `json:"input"`
	Expect struct {
		ToISODateGregorian string `json:"toISODateGregorian"`
		FormatYYYYMMDD     string `json:"formatYYYYMMDD"`
	} `json:"expect"`
}

type goldenFormat struct {
	ID      string         `json:"id"`
	Create  createSpec     `json:"create"`
	Pattern string         `json:"pattern"`
	Options map[string]any `json:"options"`
	Expect  string         `json:"expect"`
	ToPersian *struct {
		Input  string `json:"input"`
		Expect string `json:"expect"`
	} `json:"toPersian"`
	FromPersian *struct {
		Input  string `json:"input"`
		Expect string `json:"expect"`
	} `json:"fromPersian"`
}

type goldenTimezone struct {
	ID               string `json:"id"`
	UnixMs           int64  `json:"unixMs"`
	WithTimezone     string `json:"withTimezone"`
	SetTimezoneAlias string `json:"setTimezoneAlias"`
	ExpectTimezone   string `json:"expectTimezone"`
	Expect           *struct {
		JalaliSlash    string `json:"jalaliSlash"`
		JalaliIso      string `json:"jalaliIso"`
		WallGregorian  string `json:"wallGregorian"`
		WallTime       string `json:"wallTime"`
		Hours          int    `json:"hours"`
		Minutes        int    `json:"minutes"`
		GetTime        *int64 `json:"getTime"`
	} `json:"expect"`
}

type goldenMutation struct {
	ID          string     `json:"id"`
	Create      createSpec `json:"create"`
	Ops         []string   `json:"ops"`
	SetFullYear *struct {
		Year   int `json:"year"`
		Month0 int `json:"month0"`
		Day    int `json:"day"`
	} `json:"setFullYear"`
	Expect struct {
		FormatYYYYMMDD string `json:"formatYYYYMMDD"`
		GetMonth       *int   `json:"getMonth"`
	} `json:"expect"`
}

func loadGolden(t *testing.T) goldenRoot {
	t.Helper()
	path := filepath.Join("testdata", "golden.json")
	data, err := os.ReadFile(path)

	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var g goldenRoot

	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("parse golden: %v", err)
	}

	return g
}

func createFrom(t *testing.T, spec createSpec) *jalalix.JalaliX {
	t.Helper()

	if spec.UnixMs != nil {
		d, err := jalalix.Create(*spec.UnixMs)
		if err != nil {
			t.Fatalf("create ms: %v", err)
		}

		if spec.Timezone != "" {
			d, err = d.SetTimezone(spec.Timezone)
			if err != nil {
				t.Fatalf("setTimezone: %v", err)
			}
		}

		return d
	}

	d, err := jalalix.Create(spec.ISODate)

	if err != nil {
		t.Fatalf("create iso: %v", err)
	}

	return d
}

func TestGoldenCalendar(t *testing.T) {
	g := loadGolden(t)

	for _, c := range g.Calendar {
		d := createFrom(t, c.Create)

		if got := d.Format("YYYY/MM/DD"); got != c.Expect.FormatYYYYMMDD {
			t.Errorf("%s format: got %q want %q", c.ID, got, c.Expect.FormatYYYYMMDD)
		}

		if got := d.ToISODate(); got != c.Expect.ToISODateGregorian {
			t.Errorf("%s iso g: got %q want %q", c.ID, got, c.Expect.ToISODateGregorian)
		}

		if got := d.ToISODate("jalali"); got != c.Expect.ToISODateJalali {
			t.Errorf("%s iso j: got %q want %q", c.ID, got, c.Expect.ToISODateJalali)
		}
	}
}

func TestGoldenParse(t *testing.T) {
	g := loadGolden(t)

	for _, c := range g.Parse {
		d, err := jalalix.ParseJalali(c.Input)

		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}

		if got := d.ToISODate(); got != c.Expect.ToISODateGregorian {
			t.Errorf("%s iso: got %q want %q", c.ID, got, c.Expect.ToISODateGregorian)
		}

		if c.Expect.FormatYYYYMMDD != "" {
			if got := d.Format("YYYY/MM/DD"); got != c.Expect.FormatYYYYMMDD {
				t.Errorf("%s format: got %q want %q", c.ID, got, c.Expect.FormatYYYYMMDD)
			}
		}
	}
}

func TestGoldenFormat(t *testing.T) {
	g := loadGolden(t)

	for _, c := range g.Format {
		if c.ToPersian != nil {
			if got := jalalix.ToPersian(c.ToPersian.Input); got != c.ToPersian.Expect {
				t.Errorf("toPersian: got %q want %q", got, c.ToPersian.Expect)
			}

			if got := jalalix.FromPersian(c.FromPersian.Input); got != c.FromPersian.Expect {
				t.Errorf("fromPersian: got %q want %q", got, c.FromPersian.Expect)
			}

			continue
		}

		d := createFrom(t, c.Create)
		got := d.Format(c.Pattern, c.Options)

		if got != c.Expect {
			t.Errorf("%s: got %q want %q", c.ID, got, c.Expect)
		}
	}
}

func TestGoldenTimezone(t *testing.T) {
	g := loadGolden(t)

	for _, c := range g.Timezone {
		if c.SetTimezoneAlias != "" {
			d, err := jalalix.Create(c.UnixMs)

			if err != nil {
				t.Fatal(err)
			}

			d, err = d.SetTimezone(c.SetTimezoneAlias)

			if err != nil {
				t.Fatal(err)
			}

			if d.GetTimezone() != c.ExpectTimezone {
				t.Errorf("%s: got tz %q want %q", c.ID, d.GetTimezone(), c.ExpectTimezone)
			}

			continue
		}

		d, err := jalalix.Create(c.UnixMs)

		if err != nil {
			t.Fatal(err)
		}

		d, err = d.SetTimezone(c.WithTimezone)

		if err != nil {
			t.Fatal(err)
		}

		ex := c.Expect

		if got := d.Format("YYYY/MM/DD"); got != ex.JalaliSlash {
			t.Errorf("%s jalali: got %q want %q", c.ID, got, ex.JalaliSlash)
		}

		if ex.JalaliIso != "" {
			if got := d.ToISODate("jalali"); got != ex.JalaliIso {
				t.Errorf("%s jalaliIso: got %q want %q", c.ID, got, ex.JalaliIso)
			}
		}

		if ex.WallGregorian != "" {
			if got := d.ToISODate(); got != ex.WallGregorian {
				t.Errorf("%s wallG: got %q want %q", c.ID, got, ex.WallGregorian)
			}
		}

		if d.GetHours() != ex.Hours {
			t.Errorf("%s h: got %d want %d", c.ID, d.GetHours(), ex.Hours)
		}

		if d.GetMinutes() != ex.Minutes {
			t.Errorf("%s m: got %d want %d", c.ID, d.GetMinutes(), ex.Minutes)
		}

		if ex.GetTime != nil && d.GetTime() != *ex.GetTime {
			t.Errorf("%s instant: got %d want %d", c.ID, d.GetTime(), *ex.GetTime)
		}
	}
}

func TestGoldenMutations(t *testing.T) {
	g := loadGolden(t)

	for _, c := range g.Mutations {
		d := createFrom(t, c.Create)

		if c.SetFullYear != nil {
			d.SetFullYear(c.SetFullYear.Year, c.SetFullYear.Month0, c.SetFullYear.Day)

			if c.Expect.GetMonth != nil && d.GetMonth() != *c.Expect.GetMonth {
				t.Errorf("%s getMonth: got %d want %d", c.ID, d.GetMonth(), *c.Expect.GetMonth)
			}

			if got := d.Format("YYYY/MM/DD"); got != c.Expect.FormatYYYYMMDD {
				t.Errorf("%s format: got %q want %q", c.ID, got, c.Expect.FormatYYYYMMDD)
			}

			continue
		}

		for _, op := range c.Ops {
			if strings.HasPrefix(op, "addMonths:") {
				n, _ := strconv.Atoi(strings.Split(op, ":")[1])
				d.AddMonths(n)
			} else if op == "getStartOfMonth" {
				d.GetStartOfMonth()
			}
		}

		if got := d.Format("YYYY/MM/DD"); got != c.Expect.FormatYYYYMMDD {
			t.Errorf("%s: got %q want %q", c.ID, got, c.Expect.FormatYYYYMMDD)
		}
	}
}
