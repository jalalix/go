// Package jalalix is a Jalali (Persian / Shamsi) calendar library.
//
// Algorithm matches @jalalix/typescript@0.8.0: base date 1348/10/11 (= 1970-01-01),
// leapYearDivision = 0.24219858156.
//
// Method names are idiomatic Go (Create, ParseJalali, Format, …). TypeScript
// equivalents use the same camelCase names (create, parseJalali, format, …).
package jalalix

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	leapYearDivision = 0.24219858156
	dateLength       = int64(86400000)
	firstDayOfWeek   = 6
	baseYear         = 1348
	baseMonth        = 10
	baseDay          = 11
)

var (
	weekDaysNames = []string{"شنبه", "یکشنبه", "دوشنبه", "سه شنبه", "چهار شنبه", "پنج شنبه", "جمعه"}
	weekDaysLetters = []string{"ش", "ی", "د", "س", "چ", "پ", "ج"}
	weekDaysGregorianNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	weekDaysGregorianLetters = []string{"S", "M", "T", "W", "T", "F", "S"}
	monthsNames = []string{"فروردین", "اردیبهشت", "خرداد", "تیر", "مرداد", "شهریور", "مهر", "آبان", "آذر", "دی", "بهمن", "اسفند"}
	monthsGregorianNames = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	persianNumbers = []rune{'۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'}
)

const (
	am          = "ق.ظ."
	pm          = "ب.ظ."
	amGregorian = "AM"
	pmGregorian = "PM"
)

// Options configures Create / ParseJalali (TS: { timezone }).
type Options struct {
	Timezone string
}

// FormatOptions configures Format (TS: { calendar, persianDigits }).
type FormatOptions struct {
	Calendar      string // "jalali" (default) or "gregorian"
	PersianDigits bool
}

type wallParts struct {
	year, month, day int
	hour, minute, second, ms int
}

type jalaliDate struct {
	year, month, day int
	hours, min, sec, ms int
	isLeapYear bool
}

// JalaliX holds a UTC Unix-millisecond instant plus an IANA timezone id.
type JalaliX struct {
	ms          int64
	timezone    string
	loc         *time.Location
	jalali      jalaliDate
	isConverted bool
}

func hostTimezone() string {
	if tz := os.Getenv("TZ"); tz != "" && tz != ":" {
		if loc, err := time.LoadLocation(tz); err == nil && loc != nil {
			return tz
		}
	}

	// Best-effort: local zone name when it looks like IANA.
	name, _ := time.Now().Zone()

	if strings.Contains(name, "/") {
		return name
	}

	return "UTC"
}

func loadLocation(id string) (*time.Location, error) {
	if id == "UTC" || id == "Etc/UTC" {
		return time.UTC, nil
	}

	loc, err := time.LoadLocation(id)

	if err != nil {
		return nil, fmt.Errorf("invalid IANA timezone: %s", id)
	}

	return loc, nil
}

func (j *JalaliX) normalizeTimezone(timezone string) (string, *time.Location, error) {
	if timezone == "" {
		return j.timezone, j.loc, nil
	}

	lowered := strings.ToLower(strings.TrimSpace(timezone))

	switch lowered {
		case "default", "local", "system":
			id := hostTimezone()
			loc, err := loadLocation(id)

			if err != nil {
				return "UTC", time.UTC, nil
			}

			return id, loc, nil
		case "utc", "gmt":
			return "UTC", time.UTC, nil
	}

	loc, err := loadLocation(timezone)

	if err != nil {
		return "", nil, err
	}

	// Prefer canonical casing when LoadLocation accepts the id.
	return timezone, loc, nil
}

func newWithMS(ms int64, timezone string) (*JalaliX, error) {
	id, loc, err := (&JalaliX{timezone: "UTC", loc: time.UTC}).normalizeTimezone(timezone)

	if err != nil {
		return nil, err
	}

	if timezone == "" {
		id, loc = "UTC", time.UTC
	}

	j := &JalaliX{
		ms:       ms,
		timezone: id,
		loc:      loc,
		jalali: jalaliDate{
			year: baseYear, month: baseMonth, day: baseDay,
		},
	}

	return j, nil
}

func (j *JalaliX) getWallParts() wallParts {
	t := time.UnixMilli(j.ms).In(j.loc)
	ms := int(((j.ms % 1000) + 1000) % 1000)

	return wallParts{
		year: t.Year(), month: int(t.Month()), day: t.Day(),
		hour: t.Hour(), minute: t.Minute(), second: t.Second(), ms: ms,
	}
}

// wallToUTC maps wall-clock datetime in the active zone to a UTC millisecond instant.
func (j *JalaliX) wallToUTC(y, m, d, h, mi, s, ms int) int64 {
	want := time.Date(y, time.Month(m), d, h, mi, s, 0, time.UTC).UnixMilli()
	utc := want

	for i := 0; i < 14; i++ {
		gotT := time.UnixMilli(utc).In(j.loc)
		got := time.Date(gotT.Year(), gotT.Month(), gotT.Day(), gotT.Hour(), gotT.Minute(), gotT.Second(), 0, time.UTC).UnixMilli()
		diff := want - got

		if diff == 0 {
			break
		}

		utc += diff
	}

	return (utc/1000)*1000 + int64(((ms%1000)+1000)%1000)
}

func checkLeapYear(year int) bool {
	if year > 0 {
		checkYear := float64(year+2346) * leapYearDivision
		return checkYear-math.Trunc(checkYear) < leapYearDivision
	}

	return false
}

func (j *JalaliX) toJalali() jalaliDate {
	if !j.isConverted {
		wall := j.getWallParts()
		civilDay := time.Date(wall.year, time.Month(wall.month), wall.day, 0, 0, 0, 0, time.UTC).UnixMilli() / dateLength
		baseDayNum := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli() / dateLength
		diffDays := civilDay - baseDayNum

		j.jalali.year = baseYear
		j.jalali.month = baseMonth
		j.jalali.day = baseDay

		if diffDays > 0 {
			for i := int64(0); i < diffDays; i++ {
				j.jalali.day++

				if j.jalali.month < 7 && j.jalali.day > 31 {
					j.jalali.month++
					j.jalali.day = 1
				} else if j.jalali.month > 6 && j.jalali.month < 12 && j.jalali.day > 30 {
					j.jalali.month++
					j.jalali.day = 1
				} else if j.jalali.month == 12 {
					isLeap := checkLeapYear(j.jalali.year)

					if isLeap && j.jalali.day > 30 {
						j.jalali.year++
						j.jalali.month = 1
						j.jalali.day = 1
					} else if !isLeap && j.jalali.day > 29 {
						j.jalali.year++
						j.jalali.month = 1
						j.jalali.day = 1
					}
				}
			}
		} else if diffDays < 0 {
			for i := int64(0); i < -diffDays; i++ {
				j.jalali.day--

				if j.jalali.day < 1 {
					if j.jalali.month == 1 {
						j.jalali.year--
						j.jalali.month = 12

						if checkLeapYear(j.jalali.year) {
							j.jalali.day = 30
						} else {
							j.jalali.day = 29
						}
					} else if j.jalali.month < 8 {
						j.jalali.month--
						j.jalali.day = 31
					} else if j.jalali.month < 13 {
						j.jalali.month--
						j.jalali.day = 30
					}
				}
			}
		}

		j.jalali.hours = wall.hour
		j.jalali.min = wall.minute
		j.jalali.sec = wall.second
		j.jalali.ms = wall.ms
		j.jalali.isLeapYear = checkLeapYear(j.jalali.year)
		j.isConverted = true
	}

	return j.jalali
}

func (j *JalaliX) toGregorian(jDate jalaliDate) int64 {
	wall := j.getWallParts()
	cursor := jalaliDate{year: baseYear, month: baseMonth, day: baseDay}
	diffDays := int64(0)

	// Match TypeScript condition exactly (including redundant clauses).
	afterOrEqual := jDate.year > 1348 ||
		(jDate.year == 1348 && jDate.month >= 10) ||
		(jDate.year == 1348 && jDate.month == 10 && jDate.day >= 11)

	before := jDate.year < 1348 ||
		(jDate.year == 1348 && jDate.month < 10) ||
		(jDate.year == 1348 && jDate.month == 10 && jDate.day < 11)

	if afterOrEqual {
		for {
			if cursor.year == jDate.year && cursor.month == jDate.month && cursor.day == jDate.day {
				break
			}

			diffDays++
			cursor.day++

			if cursor.month < 7 && cursor.day > 31 {
				cursor.month++
				cursor.day = 1
			} else if cursor.month > 6 && cursor.month < 12 && cursor.day > 30 {
				cursor.month++
				cursor.day = 1
			} else if cursor.month == 12 {
				isLeap := checkLeapYear(cursor.year)

				if isLeap && cursor.day > 30 {
					cursor.year++
					cursor.month = 1
					cursor.day = 1
				} else if !isLeap && cursor.day > 29 {
					cursor.year++
					cursor.month = 1
					cursor.day = 1
				}
			}
		}
	} else if before {
		for {
			if cursor.year == jDate.year && cursor.month == jDate.month && cursor.day == jDate.day {
				break
			}

			diffDays--
			cursor.day--

			if cursor.day < 1 {
				if cursor.month == 1 {
					cursor.year--
					cursor.month = 12

					if checkLeapYear(cursor.year) {
						cursor.day = 30
					} else {
						cursor.day = 29
					}
				} else if cursor.month < 8 {
					cursor.month--
					cursor.day = 31
				} else if cursor.month < 13 {
					cursor.month--
					cursor.day = 30
				}
			}
		}
	}

	// TS: jDate.hours ?? wall.hour — after toJalali, jalali HMS matches wall.
	hours, min, sec, ms := wall.hour, wall.minute, wall.second, wall.ms

	wallAsUTC := diffDays*dateLength + int64(hours*3600+min*60+sec)*1000 + int64(ms)
	civil := time.UnixMilli(wallAsUTC).UTC()

	return j.wallToUTC(
		civil.Year(), int(civil.Month()), civil.Day(),
		civil.Hour(), civil.Minute(), civil.Second(),
		int(civil.Nanosecond()/1e6),
	)
}

func (j *JalaliX) reset() jalaliDate {
	j.isConverted = false
	j.jalali.year = baseYear
	j.jalali.month = baseMonth
	j.jalali.day = baseDay
	return j.toJalali()
}

func addZero(n int) string {
	if n > -1 && n < 10 {
		return fmt.Sprintf("0%d", n)
	}

	return strconv.Itoa(n)
}

func (j *JalaliX) getWallWeekday() int {
	wall := j.getWallParts()

	return int(time.Date(wall.year, time.Month(wall.month), wall.day, 0, 0, 0, 0, time.UTC).Weekday())
}

func (j *JalaliX) getGregorianDayOfYear() int {
	wall := j.getWallParts()
	start := time.Date(wall.year, 1, 0, 0, 0, 0, 0, time.UTC) // Dec 31 previous year
	civil := time.Date(wall.year, time.Month(wall.month), wall.day, 0, 0, 0, 0, time.UTC)

	return int(math.Round(float64(civil.UnixMilli()-start.UnixMilli()) / float64(dateLength)))
}

func (j *JalaliX) getGregorianWeekOfYear() int {
	wall := j.getWallParts()
	d := time.Date(wall.year, time.Month(wall.month), wall.day, 0, 0, 0, 0, time.UTC)
	dow := int(d.Weekday())

	if dow == 0 {
		dow = 7
	}

	d = d.AddDate(0, 0, 4-dow)
	yearStart := time.Date(d.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	return int(math.Ceil((float64(d.UnixMilli()-yearStart.UnixMilli())/float64(dateLength) + 1) / 7))
}

func (j *JalaliX) getWeekDayName() string {
	return weekDaysNames[(j.GetDay()+firstDayOfWeek)%7]
}

func (j *JalaliX) getWeekDayLetter() string {
	return weekDaysLetters[(j.GetDay()+firstDayOfWeek)%7]
}

func (j *JalaliX) getUTCTimezone() string {
	wall := j.getWallParts()
	asUTC := time.Date(wall.year, time.Month(wall.month), wall.day, wall.hour, wall.minute, wall.second, wall.ms*1e6, time.UTC).UnixMilli()
	offsetMin := int(math.Round(float64(asUTC-j.ms) / 60000))
	sign := "+"

	if offsetMin < 0 {
		sign = "-"
	}

	abs := offsetMin

	if abs < 0 {
		abs = -abs
	}

	return fmt.Sprintf(" GMT%s%s%s", sign, addZero(abs/60), addZero(abs%60))
}

func (j *JalaliX) normalizeDate(days int) int64 {
	j.ms += int64(days) * dateLength

	return j.ms
}

func (j *JalaliX) normalizeJalali(jDate jalaliDate, year, month, day int) int64 {
	if month < 1 {
		yearDiff := int(math.Ceil(float64(1-month) / 12))
		year -= yearDiff
		month = 12*yearDiff + month
	} else if month > 12 {
		yearDiff := int(math.Floor(float64(month-1) / 12))
		year += yearDiff
		month = month - 12*yearDiff
	}

	if month > 6 && month < 12 && day > 30 {
		day = 30
	} else if month == 12 {
		isLeap := checkLeapYear(year)

		if isLeap && day > 30 {
			day = 30
		} else if !isLeap && day > 29 {
			day = 29
		}
	}

	target := jDate
	target.year, target.month, target.day = year, month, day
	j.ms = j.toGregorian(target)

	return j.ms
}

var isoDateOnly = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var jalaliParseRE = regexp.MustCompile(`^(\d{4})[/-](\d{1,2})[/-](\d{1,2})(?:[ T](\d{1,2}):(\d{1,2})(?::(\d{1,2}))?)?$`)

func parseInputMS(value any) (int64, error) {
	switch v := value.(type) {
		case int:
			return int64(v), nil
		case int64:
			return v, nil
		case float64:
			return int64(v), nil
		case time.Time:
			return v.UnixMilli(), nil
		case *JalaliX:
			return v.ms, nil
		case JalaliX:
			return v.ms, nil
		case string:
			return parseISOString(v)
		default:
			return 0, fmt.Errorf("JalaliX.Create: unsupported date type %T", value)
	}
}

func parseISOString(value string) (int64, error) {
	s := strings.TrimSpace(value)

	if isoDateOnly.MatchString(s) {
		t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
		if err != nil {
			return 0, fmt.Errorf("JalaliX.Create: invalid date string %q", value)
		}

		return t.UnixMilli(), nil
	}

	// Try common layouts
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UnixMilli(), nil
		}

		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil && !strings.ContainsAny(s, "Zz+-") {
			// Naive datetime -> local, matching JS for non-Z ISO without offset is fuzzy;
			// prefer UTC for portability when no zone marker.
			_ = t
		}
	}

	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.UTC); err == nil {
		return t.UnixMilli(), nil
	}

	return 0, fmt.Errorf("JalaliX.Create: invalid date string %q", value)
}

// Create builds a JalaliX from a Unix-ms int, ISO string, time.Time, or JalaliX.
// TS: create(date, options?).
func Create(value any, opts ...Options) (*JalaliX, error) {
	ms, err := parseInputMS(value)

	if err != nil {
		return nil, err
	}

	tz := "UTC"

	if len(opts) > 0 && opts[0].Timezone != "" {
		tz = opts[0].Timezone
	}

	out, err := newWithMS(ms, tz)

	if err != nil {
		return nil, err
	}

	return out, nil
}

// MustCreate is Create that panics on error.
func MustCreate(value any, opts ...Options) *JalaliX {
	j, err := Create(value, opts...)

	if err != nil {
		panic(err)
	}

	return j
}

// ParseJalali parses a Jalali date string (YYYY/MM/DD or YYYY-MM-DD, optional time, Persian digits).
// TS: parseJalali.
func ParseJalali(value string, opts ...Options) (*JalaliX, error) {
	normalized := strings.TrimSpace(FromPersian(value))
	m := jalaliParseRE.FindStringSubmatch(normalized)

	if m == nil {
		return nil, fmt.Errorf("JalaliX.parseJalali: invalid Jalali date %q", value)
	}

	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	hours, minutes, seconds := 0, 0, 0

	if m[4] != "" {
		hours, _ = strconv.Atoi(m[4])
		minutes, _ = strconv.Atoi(m[5])
		if m[6] != "" {
			seconds, _ = strconv.Atoi(m[6])
		}
	}

	if month < 1 || month > 12 || day < 1 || day > 31 {
		return nil, fmt.Errorf("JalaliX.parseJalali: invalid Jalali date %q", value)
	}

	out, err := Create(time.Now(), opts...)

	if err != nil {
		return nil, err
	}

	out.SetFullYear(year, month-1, day)
	out.SetHours(hours, minutes, seconds, 0)
	return out, nil
}

// MustParseJalali is ParseJalali that panics on error.
func MustParseJalali(value string, opts ...Options) *JalaliX {
	j, err := ParseJalali(value, opts...)
	if err != nil {
		panic(err)
	}
	return j
}

// IsValidDate reports whether value is a usable date input.
func IsValidDate(value any) bool {
	switch v := value.(type) {
		case time.Time:
			return !v.IsZero()
		case *JalaliX:
			return v != nil
		case JalaliX:
			return true
		case int, int64, float64:
			return true
		default:
			return false
	}
}

// ToPersian converts Western digits to Persian numerals.
func ToPersian(s string) string {
	var b strings.Builder

	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(persianNumbers[r-'0'])
		} else {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// FromPersian converts Persian numerals to Western digits.
func FromPersian(s string) string {
	var b strings.Builder

	for _, r := range s {
		found := false

		for i, p := range persianNumbers {
			if r == p {
				b.WriteByte(byte('0' + i))
				found = true
				break
			}
		}

		if !found {
			b.WriteRune(r)
		}
	}

	return b.String()
}

// Compare returns true if a is before b (TS: compare).
func Compare(a, b *JalaliX) bool {
	return a.GetTime() < b.GetTime()
}

func (j *JalaliX) GetTime() int64 { return j.ms }

func (j *JalaliX) GetTimezone() string { return j.timezone }

func (j *JalaliX) Clone() *JalaliX {
	out := *j
	out.jalali = j.jalali
	return &out
}

// SetTimezone returns a new instance with the same UTC instant in another zone (instant-preserving).
func (j *JalaliX) SetTimezone(timezone string) (*JalaliX, error) {
	id, loc, err := j.normalizeTimezone(timezone)

	if err != nil {
		return nil, err
	}

	out := &JalaliX{
		ms: j.ms, timezone: id, loc: loc,
		jalali: jalaliDate{year: baseYear, month: baseMonth, day: baseDay},
	}

	return out, nil
}

func (j *JalaliX) GetFullYear() int { return j.toJalali().year }

// GetMonth returns Jalali month 1–12 (Farvardin = 1). TS quirk vs setMonth 0–11.
func (j *JalaliX) GetMonth() int { return j.toJalali().month }

func (j *JalaliX) GetDate() int { return j.toJalali().day }

func (j *JalaliX) GetDay() int {
	return (j.getWallWeekday() - firstDayOfWeek + 7 + 1) % 7
}

func (j *JalaliX) GetHours() int   { return j.getWallParts().hour }
func (j *JalaliX) GetMinutes() int { return j.getWallParts().minute }
func (j *JalaliX) GetSeconds() int { return j.getWallParts().second }

func (j *JalaliX) GetMilliseconds() int {
	return int(((j.ms % 1000) + 1000) % 1000)
}

func (j *JalaliX) GetDayOfYear() int {
	jd := j.toJalali()

	if jd.month < 7 {
		return jd.day + (jd.month-1)*31
	}

	if jd.month < 12 {
		return jd.day + 186 + (jd.month-7)*30
	}

	return jd.day + 336
}

func (j *JalaliX) GetDayISO() int {
	return ((j.GetDay() + 6) % 7) + 1
}

func (j *JalaliX) GetWeekOfYear() int {
	dayOfYear := j.GetDayOfYear()
	start := j.Clone().GetStartOfYear()
	dayOfWeek := start.GetDayISO()

	return int(math.Ceil(float64(dayOfYear+dayOfWeek-1) / 7))
}

func (j *JalaliX) GetHoursISO() int {
	h := j.GetHours()

	if h > 12 {
		return h - 12
	}

	return h
}

func (j *JalaliX) GetMeridiem() string {
	if j.toJalali().hours > 12 {
		return pm
	}

	return am
}

func (j *JalaliX) getGregorianMeridiem() string {
	if j.GetHours() >= 12 {
		return pmGregorian
	}

	return amGregorian
}

func (j *JalaliX) SetTime(ms int64) int64 {
	j.ms = ms
	j.reset()

	return j.ms
}

func (j *JalaliX) SetMilliseconds(ms int) int64 {
	w := j.getWallParts()
	j.ms = j.wallToUTC(w.year, w.month, w.day, w.hour, w.minute, w.second, ms)
	j.reset()

	return j.ms
}

func (j *JalaliX) SetSeconds(sec int, ms ...int) int64 {
	w := j.getWallParts()
	mss := w.ms

	if len(ms) > 0 {
		mss = ms[0]
	}

	j.ms = j.wallToUTC(w.year, w.month, w.day, w.hour, w.minute, sec, mss)
	j.reset()

	return j.ms
}

func (j *JalaliX) SetMinutes(min int, rest ...int) int64 {
	w := j.getWallParts()
	sec, mss := w.second, w.ms

	if len(rest) > 0 {
		sec = rest[0]
	}

	if len(rest) > 1 {
		mss = rest[1]
	}

	j.ms = j.wallToUTC(w.year, w.month, w.day, w.hour, min, sec, mss)
	j.reset()

	return j.ms
}

func (j *JalaliX) SetHours(hours int, rest ...int) int64 {
	w := j.getWallParts()
	min, sec, mss := w.minute, w.second, w.ms

	if len(rest) > 0 {
		min = rest[0]
	}

	if len(rest) > 1 {
		sec = rest[1]
	}

	if len(rest) > 2 {
		mss = rest[2]
	}

	j.ms = j.wallToUTC(w.year, w.month, w.day, hours, min, sec, mss)
	j.reset()

	return j.ms
}

func (j *JalaliX) SetDate(date int) int64 {
	jd := j.toJalali()
	j.normalizeDate(date - jd.day)

	return j.ms
}

// SetMonth sets Jalali month using 0–11 (Farvardin = 0), matching JS Date / TS.
func (j *JalaliX) SetMonth(month int, date ...int) int64 {
	jd := j.toJalali()
	j.normalizeJalali(jd, jd.year, month+1, jd.day)
	j.reset()

	if len(date) > 0 {
		jd = j.toJalali()
		j.normalizeDate(date[0] - jd.day)
		j.reset()
	}

	return j.ms
}

// SetFullYear sets Jalali year; optional month is 0–11 (Farvardin = 0).
func (j *JalaliX) SetFullYear(year int, rest ...int) int64 {
	jd := j.toJalali()
	month := jd.month

	if len(rest) > 0 {
		month = rest[0] + 1
	}

	j.normalizeJalali(jd, year, month, jd.day)
	j.reset()

	if len(rest) > 1 {
		jd = j.toJalali()
		j.normalizeDate(rest[1] - jd.day)
		j.reset()
	}

	return j.ms
}

func (j *JalaliX) AddYears(n int) *JalaliX {
	jd := j.toJalali()
	j.normalizeJalali(jd, jd.year+n, jd.month, jd.day)
	j.reset()

	return j
}

func (j *JalaliX) AddMonths(n int) *JalaliX {
	jd := j.toJalali()
	j.normalizeJalali(jd, jd.year, jd.month+n, jd.day)
	j.reset()

	return j
}

func (j *JalaliX) AddWeeks(n int) *JalaliX {
	j.normalizeDate(n * 7)
	j.reset()

	return j
}

func (j *JalaliX) AddDays(n int) *JalaliX {
	j.normalizeDate(n)
	j.reset()

	return j
}

func (j *JalaliX) GetStartOfYear() *JalaliX {
	return j.AddDays(-j.GetDayOfYear() + 1).GetStartOfDay()
}

func (j *JalaliX) GetStartOfMonth() *JalaliX {
	return j.AddDays(-j.GetDate() + 1).GetStartOfDay()
}

func (j *JalaliX) GetStartOfWeek() *JalaliX {
	return j.AddDays(-j.GetDayISO() + 1).GetStartOfDay()
}

func (j *JalaliX) GetStartOfDay() *JalaliX {
	j.SetHours(0, 0, 0, 0)

	return j
}

func (j *JalaliX) GetEndOfYear() *JalaliX {
	jd := j.toJalali()
	last := 365

	if jd.isLeapYear {
		last = 366
	}

	return j.AddDays(last - j.GetDayOfYear()).GetEndOfDay()
}

func (j *JalaliX) GetEndOfMonth() *JalaliX {
	jd := j.toJalali()
	last := 31

	if jd.month > 6 && jd.month < 12 {
		last = 30
	} else if jd.month == 12 {
		if jd.isLeapYear {
			last = 30
		} else {
			last = 29
		}
	}

	return j.AddDays(last - j.GetDate()).GetEndOfDay()
}

func (j *JalaliX) GetEndOfWeek() *JalaliX {
	return j.AddDays(7 - j.GetDayISO()).GetEndOfDay()
}

func (j *JalaliX) GetEndOfDay() *JalaliX {
	j.SetHours(23, 59, 59, 999)

	return j
}

// ToISODate returns wall calendar YYYY-MM-DD (gregorian default, or "jalali").
func (j *JalaliX) ToISODate(calendar ...string) string {
	cal := "gregorian"

	if len(calendar) > 0 && calendar[0] != "" {
		cal = calendar[0]
	}

	if cal == "jalali" {
		return j.Format("YYYY-MM-DD")
	}

	w := j.getWallParts()
	return fmt.Sprintf("%d-%s-%s", w.year, addZero(w.month), addZero(w.day))
}

// Format formats according to pattern. Optional FormatOptions or (persianDigits, gregorian) bools.
// TS: format(str, persianOrOptions?, gregorian?).
func (j *JalaliX) Format(pattern string, args ...any) string {
	persian := false
	useGregorian := false

	if len(args) > 0 {
		switch v := args[0].(type) {
			case FormatOptions:
				persian = v.PersianDigits
				useGregorian = v.Calendar == "gregorian"
			case *FormatOptions:
				if v != nil {
					persian = v.PersianDigits
					useGregorian = v.Calendar == "gregorian"
				}
			case map[string]any:
				if pd, ok := v["persianDigits"].(bool); ok {
					persian = pd
				}
				if c, ok := v["calendar"].(string); ok {
					useGregorian = c == "gregorian"
				}
			case bool:
				persian = v
				if len(args) > 1 {
					if g, ok := args[1].(bool); ok {
						useGregorian = g
					}
				}
		}
	}

	placeholders := []string{}

	ph := func(val string) string {
		i := len(placeholders)
		placeholders = append(placeholders, val)
		return fmt.Sprintf("\uE000%d\uE001", i)
	}

	output := pattern
	wall := j.getWallParts()
	year := j.GetFullYear()
	month := j.GetMonth()
	weekOfYear := j.GetWeekOfYear()
	dayOfYear := j.GetDayOfYear()
	date := j.GetDate()
	monthName := monthsNames[month-1]
	weekDayName := j.getWeekDayName()
	weekDayLetter := j.getWeekDayLetter()
	meridiem := j.GetMeridiem()

	if useGregorian {
		year = wall.year
		month = wall.month
		weekOfYear = j.getGregorianWeekOfYear()
		dayOfYear = j.getGregorianDayOfYear()
		date = wall.day
		monthName = monthsGregorianNames[month-1]
		weekDayName = weekDaysGregorianNames[j.getWallWeekday()]
		weekDayLetter = weekDaysGregorianLetters[j.getWallWeekday()]
		meridiem = j.getGregorianMeridiem()
	}

	yearStr := strconv.Itoa(year)

	type tok struct {
		t string
		v string
	}

	tokens := []tok{
		{"YYYY", ph(yearStr)},
		{"YYY", ph(sliceEnd(yearStr, 3))},
		{"YY", ph(sliceEnd(yearStr, 2))},
		{"Y", ph(sliceEnd(yearStr, 1))},
		{"MMM", ph(monthName)},
		{"MM", ph(addZero(month))},
		{"M", ph(strconv.Itoa(month))},
		{"W", ph(addZero(weekOfYear))},
		{"w", ph(strconv.Itoa(weekOfYear))},
		{"DW", ph(strconv.Itoa(j.GetDayISO()))},
		{"dw", ph(strconv.Itoa(func() int {
			if useGregorian {
				d := j.getWallWeekday()
				if d == 0 {
					return 7
				}
				return d
			}
			return j.GetDay()
		}()))},
		{"WN", ph(weekDayName)},
		{"wn", ph(weekDayName)},
		{"WL", ph(weekDayLetter)},
		{"wl", ph(weekDayLetter)},
		{"DDDD", ph(addZero(dayOfYear))},
		{"DDD", ph(strconv.Itoa(dayOfYear))},
		{"DD", ph(addZero(date))},
		{"D", ph(strconv.Itoa(date))},
	}

	hours := j.GetHours()
	hoursISO := hours

	if hours > 12 {
		hoursISO = hours - 12
	} else if hours == 0 {
		hoursISO = 12
	}

	tokens = append(tokens,
		tok{"HH", ph(addZero(hours))},
		tok{"H", ph(strconv.Itoa(hours))},
		tok{"hh", ph(addZero(hoursISO))},
		tok{"h", ph(strconv.Itoa(hoursISO))},
		tok{"mm", ph(addZero(j.GetMinutes()))},
		tok{"m", ph(strconv.Itoa(j.GetMinutes()))},
		tok{"ss", ph(addZero(j.GetSeconds()))},
		tok{"s", ph(strconv.Itoa(j.GetSeconds()))},
		tok{"sss", ph(strconv.Itoa(j.GetMilliseconds()))},
		tok{"SSS", ph(strconv.Itoa(j.GetMilliseconds()))},
		tok{"A", ph(meridiem)},
		tok{"a", ph(meridiem)},
		tok{"X", ph(strconv.FormatInt(j.GetTime()/1000, 10))},
		tok{"x", ph(strconv.FormatInt(j.GetTime(), 10))},
	)

	for _, t := range tokens {
		if strings.Contains(output, t.t) {
			output = strings.ReplaceAll(output, t.t, t.v)
		}
	}

	for i, val := range placeholders {
		output = strings.ReplaceAll(output, fmt.Sprintf("\uE000%d\uE001", i), val)
	}

	if persian {
		return ToPersian(output)
	}

	return output
}

func sliceEnd(s string, n int) string {
	r := []rune(s)

	if len(r) <= n {
		return s
	}

	return string(r[len(r)-n:])
}

func (j *JalaliX) String() string {
	jd := j.toJalali()

	return fmt.Sprintf("%s %s %s %s %d:%s:%s%s",
		j.getWeekDayName(), addZero(jd.day), monthsNames[jd.month-1], addZero(jd.year),
		jd.hours, addZero(jd.min), addZero(jd.sec), j.getUTCTimezone())
}

func (j *JalaliX) ToDateString() string {
	jd := j.toJalali()

	return fmt.Sprintf("%s %s %s %s",
		j.getWeekDayName(), addZero(jd.day), monthsNames[jd.month-1], addZero(jd.year))
}

func (j *JalaliX) ToLocaleDateString() string {
	jd := j.toJalali()

	return fmt.Sprintf("%s/%s/%s", addZero(jd.year), addZero(jd.month), addZero(jd.day))
}

func (j *JalaliX) ToLocaleTimeString() string {
	jd := j.toJalali()

	return fmt.Sprintf("%d:%s:%s %s", jd.hours%12, addZero(jd.min), addZero(jd.sec), j.GetMeridiem())
}
