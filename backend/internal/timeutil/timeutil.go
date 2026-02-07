package timeutil

import "time"

// Default location is UTC
var loc *time.Location = time.UTC

// Init sets the location at application startup.
// アプリケーション起動時にロケーションを設定する。
func Init(l *time.Location) {
	loc = l
}

// Now returns the current time in the configured location.
// 設定されたロケーションでの現在時刻を返す。
func Now() time.Time {
	return time.Now().In(loc)
}

// ParseDate parses a date string in "2006-01-02" format.
// "2006-01-02" 形式の日付文字列をパースする。
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, loc)
}

// Location returns the currently configured location.
// 現在設定されているロケーションを返す。
func Location() *time.Location {
	return loc
}
