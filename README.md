# JalaliX (Go)

Jalali (Persian / Shamsi) calendar library. Pure Go — `time.Location` for IANA zones, no external dependencies.

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

Algorithm matches `@jalalix/typescript@0.8.0`: base date **1348/10/11** (= 1970-01-01), `leapYearDivision = 0.24219858156`.

Exported methods use idiomatic Go names (`Create`, `ParseJalali`, `Format`, …). They map 1:1 to the TypeScript camelCase API (`create`, `parseJalali`, `format`, …).

`ToISODate()` uses wall-clock getters in the active timezone. Prefer it when you need a calendar date string without UTC shift surprises.

## Timezone

Internal storage is a **UTC Unix millisecond instant** plus an IANA timezone id. `SetTimezone` is **instant-preserving**: the timestamp stays the same; wall clock / Jalali day may change.

```go
d := jalalix.MustCreate(int64(1740430800000))

tehran, _ := d.SetTimezone("Asia/Tehran")
tehran.Format("YYYY/MM/DD") // 1403/12/07
tehran.Format("HH:mm:ss")   // 00:30:00

utc, _ := d.SetTimezone("UTC")
utc.Format("YYYY/MM/DD") // 1403/12/06
utc.Format("HH:mm:ss")   // 21:00:00

tehran.GetTime() == utc.GetTime() // true
```

- **Aliases:** `utc` / `gmt` -> `UTC`; `local` / `default` / `system` -> host zone when detectable, else `UTC`.
- **Invalid zone** -> error.
- Offsets use Go’s `time.LoadLocation` (system zoneinfo).

```go
jalalix.MustCreate(time.Now(), jalalix.Options{Timezone: "Asia/Tehran"})
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
```

### ToISODate

```go
d := jalalix.MustCreate("2025-02-25")

d.ToISODate()          // 2025-02-25
d.ToISODate("jalali")  // 1403-12-07
```

### Add / start / end

```go
jalalix.MustCreate("2025-02-25").AddMonths(1).GetStartOfMonth().Format("YYYY/MM/DD")
```

### Persian digits

```go
jalalix.ToPersian("1403/12/07")   // ۱۴۰۳/۱۲/۰۷
jalalix.FromPersian("۱۴۰۳/۱۲/۰۷") // 1403/12/07
```

## Months

Easy to get wrong:

- `GetMonth()` -> 1..12 (Farvardin = 1)
- `SetMonth(m)` / `SetFullYear(y, m)` -> 0..11, same as JS `Date` (Farvardin = 0)

```go
d := jalalix.MustCreate(time.Now())
d.SetFullYear(1404, 0, 1) // 1 Farvardin 1404
d.GetMonth()              // 1
```

## Format tokens

| Token               |                                 |
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

Static:

- `Create(date, options?)` — ms, ISO string, or `time.Time`
- `ParseJalali(str, options?)` / `MustParseJalali`
- `IsValidDate(value)`
- `ToPersian(str)` / `FromPersian(str)`
- `Compare(a, b)` — true if `a` is before `b`

Instance (main ones):

- `Format(...)`, `ToISODate(calendar?)`
- `GetFullYear`, `GetMonth`, `GetDate`, `GetDay`, …
- `SetFullYear`, `SetMonth`, `SetDate`, `SetHours`, …
- `AddYears`, `AddMonths`, `AddWeeks`, `AddDays`
- `GetStartOfYear` / `Month` / `Week` / `Day` and matching `GetEndOf*`
- `Clone()`, `SetTimezone(tz)`, `GetTimezone()`

## Links

- https://jalalix.com
- https://github.com/jalalix/go
- [CHANGELOG.md](./CHANGELOG.md)

## License

MIT
