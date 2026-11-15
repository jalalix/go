# Release Notes
All notable changes to this package will be documented in this file.

## v0.2.0
- Added `NormalizeQueryDate` for loose query/form inputs (sentinels, unix s/ms, Jalali vs Gregorian auto-detect; default year band `1…1899`, overridable via `QueryDateOptions.JalaliYearMin` / `JalaliYearMax`)
- `FromPersian` also converts Arabic-Indic digits (`٠-٩`)
- `ParseJalali` accepts `.` as a date separator

## v0.1.0
- Initial commit