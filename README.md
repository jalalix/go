# JalaliX - Go

Jalali (Persian / Shamsi) calendar on top of a millisecond Unix timestamp. Pure Go - `time.Location` for IANA zones, no external dependencies.

[![Go Reference](https://pkg.go.dev/badge/github.com/jalalix/go.svg)](https://pkg.go.dev/github.com/jalalix/go)
[![license](https://img.shields.io/github/license/jalalix/go.svg)](./LICENSE)

```bash
go get github.com/jalalix/go
```

```go
import "github.com/jalalix/go"

date := jalalix.MustCreate("2025-02-25")

date.Format("YYYY/MM/DD") // 1403/12/07
date.Format("DD MMM YYYY", jalalix.FormatOptions{PersianDigits: true}) // ۰۷ اسفند ۱۴۰۳

jalalix.MustParseJalali("1403/12/07").ToISODate() // 2025-02-25
```

`ToISODate()` uses wall-clock getters in the active timezone. Prefer it over UTC-based formatting when you need a calendar date string.

## Install

```bash
go get github.com/jalalix/go
```

```go
import "github.com/jalalix/go"
```

## Usage

### Create / Format

```go
now := jalalix.MustCreate(time.Now(), jalalix.Options{Timezone: "Asia/Tehran"})

now.Format("YYYY/MM/DD")
now.Format("WN، DD MMM YYYY")
now.Format("YYYY/MM/DD", jalalix.FormatOptions{PersianDigits: true})
now.Format("DD MMM YYYY", jalalix.FormatOptions{Calendar: "gregorian"})
```

`Format` also accepts the older positional form: `Format(str, persianDigits?, gregorian?)`.

### ParseJalali

```go
jalalix.MustParseJalali("1403/12/07")
jalalix.MustParseJalali("۱۴۰۳-۱۲-۰۷ 14:30")
jalalix.MustParseJalali("1403.12.07")
```

Accepts `/`, `-`, or `.` as the date separator. Use `ParseJalali` when you need error handling; `MustParseJalali` panics on invalid input.

### NormalizeQueryDate

Soft-normalize loose API/query date values to Gregorian `Y-m-d H:i:s` (or `nil`). Never panics on junk input.

```go
jalalix.NormalizeQueryDate("1403/12/07", jalalix.QueryDateOptions{Timezone: "Asia/Tehran"})
// "2025-02-25 00:00:00"

jalalix.NormalizeQueryDate("undefined") // nil
jalalix.NormalizeQueryDate(int64(1740430800000), jalalix.QueryDateOptions{Timezone: "Asia/Tehran"})
// "2025-02-25 00:30:00"
```

Handles sentinels (`nil`, `""`, `"null"`, `"undefined"`, `"Invalid Date"`, `NaN`) → `nil`; unix timestamps (seconds or milliseconds; non-positive → `nil`); Persian/Arabic-Indic digits; `/` `-` `.` separators; fractional seconds; optional `Z` / offset suffixes.

Detects Jalali when the year is in `JalaliYearMin`…`JalaliYearMax` on `QueryDateOptions` (default `1`…`1899`) **and** the civil date is valid; otherwise Gregorian when valid, or returns the digit-cleaned string. Non-string/non-number values pass through unchanged.

### ToISODate

```go
d := jalalix.MustCreate("2025-02-25T00:30:00+03:30")

d.ToISODate()          // 2025-02-25
d.ToISODate("jalali")  // 1403-12-07
```

### Add / start / end

```go
jalalix.MustCreate("2025-02-25").AddMonths(1).GetStartOfMonth().Format("YYYY/MM/DD")
// e.g. 1404/01/01

d := jalalix.MustCreate("2025-02-25", jalalix.Options{Timezone: "Asia/Tehran"})
d.AddDays(7).Format("YYYY/MM/DD")
d.GetEndOfWeek().Format("YYYY/MM/DD")
```

### Persian / Arabic digits

```go
jalalix.ToPersian("1403/12/07")   // ۱۴۰۳/۱۲/۰۷
jalalix.FromPersian("۱۴۰۳/۱۲/۰۷") // 1403/12/07
jalalix.FromPersian("١٤٠٣/١٢/٠٧") // 1403/12/07 (Arabic-Indic)
```

## Timezone

Internal storage is always a **UTC Unix millisecond instant** plus an IANA timezone id. `SetTimezone` is **instant-preserving**: the timestamp stays the same; only the zone (and therefore wall clock / Jalali day) changes.

```go
d := jalalix.MustCreate(int64(1740430800000)) // 2025-02-25T00:30:00+03:30

tehran, _ := d.SetTimezone("Asia/Tehran")
tehran.Format("YYYY/MM/DD") // 1403/12/07
tehran.Format("HH:mm")      // 00:30

utc, _ := d.SetTimezone("UTC")
utc.Format("YYYY/MM/DD") // 1403/12/06
utc.Format("HH:mm")      // 21:00

tehran.GetTime() == utc.GetTime() // true
```

- **Aliases:** `utc` / `gmt` → `UTC`; `local` / `default` / `system` → host zone when detectable, else `UTC`.
- **Invalid zone** → error from `SetTimezone` / `Create` / `ParseJalali`.
- Offsets use Go's `time.LoadLocation` (system zoneinfo).

```go
jalalix.MustCreate(time.Now(), jalalix.Options{Timezone: "Asia/Tehran"})
```

## Months

Easy to get wrong:

- `GetMonth()` → **1…12** (Farvardin = 1)
- `SetMonth(m)` / `SetFullYear(y, m)` → **0…11**, same as JS `Date` (Farvardin = 0)

```go
d := jalalix.MustCreate(time.Now())
d.SetFullYear(1404, 0, 1) // 1 Farvardin 1404
d.GetMonth()              // 1
```

## Format tokens

| Token               | Meaning                         |
| ------------------- | ------------------------------- |
| `YYYY` `YY`         | year                            |
| `MMM` `MM` `M`      | month name / number             |
| `DD` `D`            | day                             |
| `HH` `hh` `mm` `ss` | time                            |
| `A` `a`             | meridiem (ق.ظ. / ب.ظ. or AM/PM) |
| `WN` `WL`           | weekday name / letter           |
| `w` `DDD`           | week of year / day of year      |
| `X` `x`             | unix seconds / ms               |

## API

Exported names are idiomatic Go (PascalCase). They map 1:1 to the TypeScript camelCase API.

### Package - types

| API                | Description                                                                                                   |
| ------------------ | ------------------------------------------------------------------------------------------------------------- |
| `Options`          | `Create` / `ParseJalali` options: `Timezone string`.                                                          |
| `QueryDateOptions` | `NormalizeQueryDate` options: `Timezone`, `JalaliYearMin`, `JalaliYearMax` (defaults `1` / `1899` when zero). |
| `FormatOptions`    | `Format` options: `Calendar` (`"jalali"` or `"gregorian"`), `PersianDigits bool`.                             |
| `JalaliX`          | Date value holding a UTC unix-ms instant and IANA timezone.                                                   |

### Static (package functions)

| API                                  | Description                                                                           |
| ------------------------------------ | ------------------------------------------------------------------------------------- |
| `Create(date, opts...)`              | Build from unix ms, ISO string, or `time.Time`. Returns `(*JalaliX, error)`.          |
| `MustCreate(date, opts...)`          | Same as `Create`; panics on error.                                                    |
| `ParseJalali(str, opts...)`          | Parse a Jalali date/time string (`/` `-` `.`, Persian/Arabic digits, optional time).  |
| `MustParseJalali(str, opts...)`      | Same as `ParseJalali`; panics on error.                                               |
| `NormalizeQueryDate(value, opts...)` | Soft-normalize loose query/form values → Gregorian `Y-m-d H:i:s` or `nil`.            |
| `IsValidDate(value)`                 | `true` for non-zero `time.Time`, non-nil `*JalaliX`, `JalaliX`, or numeric timestamp. |
| `ToPersian(str)`                     | Replace Western digits `0-9` with Persian `۰-۹`.                                      |
| `FromPersian(str)`                   | Replace Persian `۰-۹` and Arabic-Indic `٠-٩` with Western digits.                     |
| `Compare(a, b)`                      | `true` if `a` is strictly before `b` (by unix ms).                                    |

### Instance - getters (`*JalaliX`)

| API                                                                  | Description                                                           |
| -------------------------------------------------------------------- | --------------------------------------------------------------------- |
| `GetFullYear()`                                                      | Jalali year in the active timezone.                                   |
| `GetMonth()`                                                         | Jalali month **1…12** (Farvardin = 1).                                |
| `GetDate()`                                                          | Jalali day of month **1…31**.                                         |
| `GetDay()`                                                           | Weekday **1…7** (Saturday = 1 … Friday = 7).                          |
| `GetHours()` / `GetMinutes()` / `GetSeconds()` / `GetMilliseconds()` | Wall-clock time in the active timezone.                               |
| `GetTime()`                                                          | Unix timestamp in milliseconds (UTC instant).                         |
| `GetTimezone()`                                                      | Active IANA timezone id.                                              |
| `GetWeekOfYear()`                                                    | Jalali week-of-year (1-based).                                        |
| `GetDayOfYear()`                                                     | Day of Jalali year (1…365/366).                                       |
| `GetDayISO()`                                                        | ISO-style weekday index used internally for week math.                |
| `GetHoursISO()`                                                      | Hours in 0…23 for formatting helpers.                                 |
| `GetMeridiem()`                                                      | Localized meridiem string (ق.ظ./ب.ظ. or AM/PM depending on calendar). |

### Instance - setters

| API                          | Description                                                       |
| ---------------------------- | ----------------------------------------------------------------- |
| `SetTime(ms)`                | Set the UTC instant from unix milliseconds; returns previous ms.  |
| `SetMilliseconds(ms)`        | Set milliseconds (wall clock).                                    |
| `SetSeconds(sec, ms...)`     | Set seconds (and optional ms).                                    |
| `SetMinutes(min, rest...)`   | Set minutes (and optional lower units).                           |
| `SetHours(h, rest...)`       | Set hours (and optional lower units).                             |
| `SetDate(day)`               | Set Jalali day of month.                                          |
| `SetMonth(month, date...)`   | Set Jalali month **0…11** (Farvardin = 0); optional day.          |
| `SetFullYear(year, rest...)` | Set Jalali year; optional month **0…11** and day.                 |
| `SetTimezone(tz)`            | Instant-preserving zone change; returns new `*JalaliX` and error. |

### Instance - math, range, format

| API                                                                               | Description                                                       |
| --------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| `AddYears(n)` / `AddMonths(n)` / `AddWeeks(n)` / `AddDays(n)`                     | Add calendar units; mutate receiver and return it.                |
| `GetStartOfYear()` / `GetStartOfMonth()` / `GetStartOfWeek()` / `GetStartOfDay()` | Start of unit (new instance).                                     |
| `GetEndOfYear()` / `GetEndOfMonth()` / `GetEndOfWeek()` / `GetEndOfDay()`         | End of unit (new instance).                                       |
| `Format(str, args...)`                                                            | Format with tokens above; `FormatOptions` or positional booleans. |
| `ToISODate(calendar...)`                                                          | `YYYY-MM-DD` in `"gregorian"` (default) or `"jalali"`.            |
| `Clone()`                                                                         | Copy with the same instant and timezone.                          |
| `String()` / `ToDateString()` / `ToLocaleDateString()` / `ToLocaleTimeString()`   | String views of the date/time.                                    |

## Links

- https://pkg.go.dev/github.com/jalalix/go
- https://jalalix.com
- https://github.com/jalalix/go
- [CHANGELOG.md](./CHANGELOG.md)

## License

MIT
