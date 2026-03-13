/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package scm

import (
	_ "time/tzdata" // embed IANA timezone database
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// tzAbbrevMap maps common timezone abbreviations to IANA zone names.
// Used as fallback when time.LoadLocation fails for an abbreviation.
var tzAbbrevMap = map[string]string{
	"UTC": "UTC", "GMT": "UTC",
	"CET": "Europe/Paris", "CEST": "Europe/Paris",
	"WET": "Europe/Lisbon", "WEST": "Europe/Lisbon",
	"EET": "Europe/Helsinki", "EEST": "Europe/Helsinki",
	"MSK": "Europe/Moscow",
	"EST": "America/New_York", "EDT": "America/New_York",
	"CST": "America/Chicago", "CDT": "America/Chicago",
	"MST": "America/Denver", "MDT": "America/Denver",
	"PST": "America/Los_Angeles", "PDT": "America/Los_Angeles",
	"AKST": "America/Anchorage", "AKDT": "America/Anchorage",
	"HST": "Pacific/Honolulu",
	"IST": "Asia/Kolkata",
	"JST": "Asia/Tokyo",
	"KST": "Asia/Seoul",
	"CST_CN": "Asia/Shanghai",
	"AEST": "Australia/Sydney", "AEDT": "Australia/Sydney",
	"NZST": "Pacific/Auckland", "NZDT": "Pacific/Auckland",
}

// tzLocationCache caches resolved *time.Location values by name to avoid repeated IANA parsing.
var tzLocationCache sync.Map // map[string]*time.Location

// ResolveLocation resolves a timezone name string to a *time.Location.
// Accepts: "UTC", "SYSTEM", "+HH:MM" / "-HH:MM" offsets, IANA names, abbreviations.
// Results are cached to avoid repeated parsing of the embedded IANA timezone database.
func ResolveLocation(name string) (*time.Location, error) {
	if v, ok := tzLocationCache.Load(name); ok {
		return v.(*time.Location), nil
	}
	loc, err := resolveLocationUncached(name)
	if err == nil {
		tzLocationCache.Store(name, loc)
	}
	return loc, err
}

func resolveLocationUncached(name string) (*time.Location, error) {
	switch strings.ToUpper(name) {
	case "UTC", "UTC+0", "UTC-0", "+00:00", "-00:00", "+0:00", "-0:00":
		return time.UTC, nil
	case "SYSTEM", "LOCAL":
		return time.Local, nil
	}
	// Fixed offset: +HH:MM or -HH:MM
	if len(name) >= 3 && (name[0] == '+' || name[0] == '-') {
		loc, err := parseFixedOffset(name)
		if err == nil {
			return loc, nil
		}
	}
	// IANA named zone
	if loc, err := time.LoadLocation(name); err == nil {
		return loc, nil
	}
	// Abbreviation fallback
	if iana, ok := tzAbbrevMap[strings.ToUpper(name)]; ok {
		return time.LoadLocation(iana)
	}
	return nil, fmt.Errorf("unknown timezone: %q", name)
}

// parseFixedOffset parses "+HH:MM", "+H:MM", or "+HH" into a fixed-offset location.
func parseFixedOffset(s string) (*time.Location, error) {
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	s = s[1:]
	var h, m int
	var err error
	switch {
	case len(s) == 5 && s[2] == ':':
		h, err = strconv.Atoi(s[0:2])
		if err == nil {
			m, err = strconv.Atoi(s[3:5])
		}
	case len(s) == 4 && s[2] == ':':
		h, err = strconv.Atoi(s[0:2])
		if err == nil {
			m, err = strconv.Atoi(s[3:4])
		}
	case len(s) == 2:
		h, err = strconv.Atoi(s)
	default:
		return nil, fmt.Errorf("cannot parse offset %q", s)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot parse offset %q: %w", s, err)
	}
	offset := sign * (h*3600 + m*60)
	name := fmt.Sprintf("%+03d:%02d", sign*h, m)
	return time.FixedZone(name, offset), nil
}

// GetCurrentSessionLocation returns the *time.Location for the current session's time_zone.
// Reads "time_zone" from the GLS session. Falls back to UTC if not set or invalid.
func GetCurrentSessionLocation() *time.Location {
	if mgr == nil {
		return time.UTC
	}
	val, ok := mgr.GetValue("session")
	if !ok {
		return time.UTC
	}
	sessionScmer := val.(Scmer)
	tz := Apply(sessionScmer, NewString("time_zone"))
	if tz.IsNil() {
		return time.UTC
	}
	loc, err := ResolveLocation(tz.String())
	if err != nil {
		return time.UTC
	}
	return loc
}

// DateToDisplay formats a tagDate Scmer value for display, respecting zone_id and session TZ.
// If the value's zone_id != 0, displays in that zone; otherwise uses sessionLoc.
func DateToDisplay(v Scmer, sessionLoc *time.Location) string {
	unix := TagDateDecodeUnix(auxVal(v.aux))
	zoneID := TagDateDecodeZone(auxVal(v.aux))
	loc := sessionLoc
	if loc == nil {
		loc = time.UTC
	}
	if zoneID != 0 {
		// zone_id is set — look up via GlobalZoneRegistry (set at startup from system.timezones).
		// For now: use UTC (zone registry is populated later in the implementation).
		// TODO: look up zone by ID from zone registry
		loc = time.UTC
	}
	return time.Unix(unix, 0).In(loc).Format("2006-01-02 15:04:05")
}

func init_timezone() {
	DeclareTitle("Timezone")

	// UNIX_TIMESTAMP(): returns current unix timestamp as integer
	// UNIX_TIMESTAMP(dt): converts datetime string to unix timestamp integer
	Declare(&Globalenv, &Declaration{
		"unix_timestamp", "returns a unix timestamp (integer seconds since epoch)",
		0, 1,
		[]DeclarationParameter{
			{"dt", "any", "optional datetime value to convert", nil},
		}, "int",
		func(a ...Scmer) Scmer {
			if len(a) == 0 {
				return NewInt(time.Now().Unix())
			}
			if a[0].IsNil() {
				return NewNil()
			}
			t, ok := toTime(a[0])
			if !ok {
				return NewNil()
			}
			return NewInt(t.Unix())
		},
		true, false, nil, nil,
	})

	// system_time_zone: returns the OS-level timezone name
	Declare(&Globalenv, &Declaration{
		"system_time_zone", "returns the operating system's local timezone name",
		0, 0,
		[]DeclarationParameter{}, "string",
		func(a ...Scmer) Scmer {
			return NewString(time.Local.String())
		},
		false, false, nil, nil,
	})

	// CONVERT_TZ(dt, from_tz, to_tz)
	Declare(&Globalenv, &Declaration{
		"convert_tz", "converts a datetime from one timezone to another",
		3, 3,
		[]DeclarationParameter{
			{"dt", "any", "datetime value", nil},
			{"from_tz", "string", "source timezone", nil},
			{"to_tz", "string", "target timezone", nil},
		}, "date",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() || a[2].IsNil() {
				return NewNil()
			}
			fromLoc, err := ResolveLocation(a[1].String())
			if err != nil {
				return NewNil()
			}
			toLoc, err := ResolveLocation(a[2].String())
			if err != nil {
				return NewNil()
			}
			// parse the input as a wall-clock time in fromLoc
			var t time.Time
			switch a[0].GetTag() {
			case tagDate:
				// tagDate stores a naive UTC unix (wall-clock as UTC); reinterpret as local in fromLoc
				wall := time.Unix(a[0].Int(), 0).UTC()
				t = time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), 0, fromLoc)
			default:
				unix, ok := parseDateStringInLoc(a[0].String(), fromLoc)
				if !ok {
					return NewNil()
				}
				t = time.Unix(unix, 0)
			}
			// convert to target zone; encode result as naive UTC (wall-clock in toLoc stored as UTC)
			tInTo := t.In(toLoc)
			naive := time.Date(tInTo.Year(), tInTo.Month(), tInTo.Day(), tInTo.Hour(), tInTo.Minute(), tInTo.Second(), 0, time.UTC)
			return NewDate(naive.Unix())
		},
		true, false, nil, nil,
	})

	// FROM_UNIXTIME(unix_ts [, format])
	Declare(&Globalenv, &Declaration{
		"from_unixtime", "converts a unix timestamp to a datetime in the session timezone",
		1, 2,
		[]DeclarationParameter{
			{"unix_ts", "number", "unix timestamp (seconds since epoch)", nil},
			{"format", "string", "optional MySQL format string", nil},
		}, "date",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() {
				return NewNil()
			}
			unix := a[0].Int()
			if len(a) == 2 && !a[1].IsNil() {
				// with format string: return string
				loc := GetCurrentSessionLocation()
				t := time.Unix(unix, 0).In(loc)
				return NewString(formatDateMySQL(t, a[1].String()))
			}
			return NewDate(unix)
		},
		true, false, nil, nil,
	})

	// UTC_TIMESTAMP()
	Declare(&Globalenv, &Declaration{
		"utc_timestamp", "returns the current UTC datetime",
		0, 0,
		[]DeclarationParameter{}, "date",
		func(a ...Scmer) Scmer {
			return NewDate(time.Now().UTC().Unix())
		},
		false, false, nil, nil,
	})

	// UTC_DATE()
	Declare(&Globalenv, &Declaration{
		"utc_date", "returns the current UTC date (midnight)",
		0, 0,
		[]DeclarationParameter{}, "date",
		func(a ...Scmer) Scmer {
			now := time.Now().UTC()
			midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			return NewDate(midnight.Unix())
		},
		false, false, nil, nil,
	})

	// UTC_TIME()
	Declare(&Globalenv, &Declaration{
		"utc_time", "returns the current UTC time (as a datetime at epoch date)",
		0, 0,
		[]DeclarationParameter{}, "date",
		func(a ...Scmer) Scmer {
			now := time.Now().UTC()
			// Return as seconds since midnight
			seconds := int64(now.Hour()*3600 + now.Minute()*60 + now.Second())
			return NewDate(seconds)
		},
		false, false, nil, nil,
	})

	// SYSDATE() — re-evaluated on every call (unlike NOW() which is constant per query)
	Declare(&Globalenv, &Declaration{
		"sysdate", "returns the current datetime (re-evaluated per call, unlike now())",
		0, 0,
		[]DeclarationParameter{}, "date",
		func(a ...Scmer) Scmer {
			return NewDate(time.Now().Unix())
		},
		false, false, nil, nil,
	})

	// AT_TIME_ZONE(dt, zone): PostgreSQL AT TIME ZONE operator implementation.
	// If dt has zone_id=0 (TIMESTAMP without TZ): interpret as local time in zone → return UTC.
	// If dt has zone_id!=0 (TIMESTAMPTZ): convert UTC moment to local time in zone → return as-is.
	Declare(&Globalenv, &Declaration{
		"at_time_zone", "PostgreSQL AT TIME ZONE operator: converts between timezones",
		2, 2,
		[]DeclarationParameter{
			{"dt", "any", "datetime value", nil},
			{"zone", "string", "target timezone", nil},
		}, "date",
		func(a ...Scmer) Scmer {
			if a[0].IsNil() || a[1].IsNil() {
				return NewNil()
			}
			toLoc, err := ResolveLocation(a[1].String())
			if err != nil {
				return NewNil()
			}
			var unix int64
			zoneID := 0
			if a[0].GetTag() == tagDate {
				unix = TagDateDecodeUnix(auxVal(a[0].aux))
				zoneID = TagDateDecodeZone(auxVal(a[0].aux))
			} else {
				unix = a[0].Int()
			}
			if zoneID == 0 {
				// TIMESTAMP without TZ: the stored unix is a wall-clock time (UTC-interpreted).
				// Reinterpret it as local time in toLoc and return UTC.
				wall := time.Unix(unix, 0).UTC()
				local := time.Date(wall.Year(), wall.Month(), wall.Day(), wall.Hour(), wall.Minute(), wall.Second(), 0, toLoc)
				return NewDate(local.UTC().Unix())
			}
			// TIMESTAMPTZ: convert the absolute UTC moment to the target zone's wall clock.
			utcTime := time.Unix(unix, 0).In(toLoc)
			// Return the local wall-clock reading as a "naive" UTC timestamp (zone_id=0)
			naive := time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), utcTime.Hour(), utcTime.Minute(), utcTime.Second(), 0, time.UTC)
			return NewDate(naive.Unix())
		},
		true, false, nil, nil,
	})

	// TIMESTAMPDIFF(unit, dt1, dt2)
	Declare(&Globalenv, &Declaration{
		"timestampdiff", "returns the difference between two datetimes in the given unit",
		3, 3,
		[]DeclarationParameter{
			{"unit", "string", "SECOND, MINUTE, HOUR, DAY, WEEK, MONTH, YEAR", nil},
			{"dt1", "any", "first datetime", nil},
			{"dt2", "any", "second datetime", nil},
		}, "int",
		func(a ...Scmer) Scmer {
			if a[1].IsNil() || a[2].IsNil() {
				return NewNil()
			}
			t1, ok1 := toTime(a[1])
			t2, ok2 := toTime(a[2])
			if !ok1 || !ok2 {
				return NewNil()
			}
			unit := strings.ToUpper(String(a[0]))
			diff := t2.Sub(t1)
			switch unit {
			case "SECOND":
				return NewInt(int64(diff.Seconds()))
			case "MINUTE":
				return NewInt(int64(diff.Minutes()))
			case "HOUR":
				return NewInt(int64(diff.Hours()))
			case "DAY":
				return NewInt(int64(diff.Hours() / 24))
			case "WEEK":
				return NewInt(int64(diff.Hours() / (24 * 7)))
			case "MONTH":
				y1, m1, _ := t1.Date()
				y2, m2, _ := t2.Date()
				return NewInt(int64((y2-y1)*12 + int(m2-m1)))
			case "YEAR":
				y1, _, _ := t1.Date()
				y2, _, _ := t2.Date()
				return NewInt(int64(y2 - y1))
			default:
				return NewNil() // unknown unit → NULL (MySQL compatible)
			}
		},
		true, false, nil, nil,
	})
}

// parseDateStringInLoc parses a date string as a local time in loc.
func parseDateStringInLoc(s string, loc *time.Location) (int64, bool) {
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, fmt := range formats {
		if t, err := time.ParseInLocation(fmt, s, loc); err == nil {
			return t.Unix(), true
		}
	}
	return 0, false
}

// formatDateMySQL formats a time.Time using MySQL format specifiers.
func formatDateMySQL(t time.Time, format string) string {
	var buf strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 'Y':
				buf.WriteString(fmt.Sprintf("%04d", t.Year()))
			case 'y':
				buf.WriteString(fmt.Sprintf("%02d", t.Year()%100))
			case 'm':
				buf.WriteString(fmt.Sprintf("%02d", t.Month()))
			case 'd':
				buf.WriteString(fmt.Sprintf("%02d", t.Day()))
			case 'H':
				buf.WriteString(fmt.Sprintf("%02d", t.Hour()))
			case 'i':
				buf.WriteString(fmt.Sprintf("%02d", t.Minute()))
			case 's':
				buf.WriteString(fmt.Sprintf("%02d", t.Second()))
			case 'T':
				buf.WriteString(fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second()))
			case '%':
				buf.WriteByte('%')
			default:
				buf.WriteByte('%')
				buf.WriteByte(format[i+1])
			}
			i++
		} else {
			buf.WriteByte(format[i])
		}
	}
	return buf.String()
}
