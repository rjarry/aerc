package parse

import (
	"fmt"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/log"
)

const dateFmt = "2006-01-02"

// ParseDateRange parses a date range into a start and end date. Dates are
// expected to be in the YYYY-MM-DD format.
//
// Start and end dates are connected by the range operator ".." where end date
// is not included in the date range.
//
// ParseDateRange can also parse open-ended ranges, i.e. start.. or ..end are
// allowed.
//
// Relative date terms (such as "1 week 1 day" or "1w 1d") can be used, too.
func DateRange(s string) (start, end time.Time, err error) {
	s = cleanInput(s)
	s = ensureRangeOp(s)
	i := strings.Index(s, "..")
	switch {
	case i < 0:
		// single date
		start, err = translate(s)
		if err != nil {
			err = fmt.Errorf("failed to parse date: %w", err)
			return
		}
		end = start.AddDate(0, 0, 1)

	case i == 0:
		// end date only
		if len(s) < 2 {
			err = fmt.Errorf("no date found")
			return
		}
		end, err = translate(s[2:])
		if err != nil {
			err = fmt.Errorf("failed to parse date: %w", err)
			return
		}

	case i > 0:
		// start date first
		start, err = translate(s[:i])
		if err != nil {
			err = fmt.Errorf("failed to parse date: %w", err)
			return
		}
		if len(s[i:]) <= 2 {
			return
		}
		// and end dates if available
		end, err = translate(s[(i + 2):])
		if err != nil {
			err = fmt.Errorf("failed to parse date: %w", err)
			return
		}
	}

	return
}

type dictFunc = func(bool) time.Time

// dict is a dictionary to translate words to dates. Map key must be at least 3
// characters for matching purposes.
var dict map[string]dictFunc = map[string]dictFunc{
	"today": func(_ bool) time.Time {
		return time.Now()
	},
	"yesterday": func(_ bool) time.Time {
		return time.Now().AddDate(0, 0, -1)
	},
	"week": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -7
		}
		return time.Now().AddDate(0, 0,
			daydiff(time.Monday)+diff)
	},
	"month": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(0, diff, -t.Day()+1)
	},
	"year": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff, 0, -t.YearDay()+1)
	},
	"monday": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -7
		}
		return time.Now().AddDate(0, 0,
			daydiff(time.Monday)+diff)
	},
	"tuesday": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -7
		}
		return time.Now().AddDate(0, 0,
			daydiff(time.Tuesday)+diff)
	},
	"wednesday": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -7
		}
		return time.Now().AddDate(0, 0,
			daydiff(time.Wednesday)+diff)
	},
	"thursday": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -7
		}
		return time.Now().AddDate(0, 0,
			daydiff(time.Thursday)+diff)
	},
	"friday": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -7
		}
		return time.Now().AddDate(0, 0,
			daydiff(time.Friday)+diff)
	},
	"saturday": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -7
		}
		return time.Now().AddDate(0, 0,
			daydiff(time.Saturday)+diff)
	},
	"sunday": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -7
		}
		return time.Now().AddDate(0, 0,
			daydiff(time.Sunday)+diff)
	},
	"january": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.January), -t.Day()+1)
	},
	"february": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.February), -t.Day()+1)
	},
	"march": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.March), -t.Day()+1)
	},
	"april": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.April), -t.Day()+1)
	},
	"may": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.May), -t.Day()+1)
	},
	"june": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.June), -t.Day()+1)
	},
	"july": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.July), -t.Day()+1)
	},
	"august": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.August), -t.Day()+1)
	},
	"september": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.September), -t.Day()+1)
	},
	"october": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.October), -t.Day()+1)
	},
	"november": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.November), -t.Day()+1)
	},
	"december": func(this bool) time.Time {
		diff := 0
		if !this {
			diff = -1
		}
		t := time.Now()
		return t.AddDate(diff,
			monthdiff(time.December), -t.Day()+1)
	},
}

func daydiff(d time.Weekday) int {
	daydiff := d - time.Now().Weekday()
	if daydiff > 0 {
		return int(daydiff) - 7
	}
	return int(daydiff)
}

func monthdiff(d time.Month) int {
	monthdiff := d - time.Now().Month()
	if monthdiff > 0 {
		return int(monthdiff) - 12
	}
	return int(monthdiff)
}

// translate translates regular time words into date strings
func translate(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), fmt.Errorf("empty string")
	}
	log.Tracef("input: %s", s)
	s0 := s

	// if next characters is integer, then parse a relative date
	if '0' <= s[0] && s[0] <= '9' && hasUnit(s) {
		relDate, err := RelativeDate(s)
		if err != nil {
			log.Errorf("could not parse relative date from '%s': %v",
				s0, err)
		} else {
			log.Tracef("relative date: translated to %v from %s",
				relDate, s0)
			return bod(relDate.Apply(time.Now())), nil
		}
	}

	// consult dictionary for terms translation
	s, this, hasPrefix := handlePrefix(s)
	for term, dateFn := range dict {
		if term == "month" && !hasPrefix {
			continue
		}
		if strings.Contains(term, s) {
			log.Tracef("dictionary: translated to %s from %s",
				term, s0)
			return bod(dateFn(this)), nil
		}
	}

	// this is a regular date, parse it in the normal format
	log.Infof("parse: translates %s to regular format", s0)
	return time.Parse(dateFmt, s)
}

// bod returns the begin of the day
func bod(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func handlePrefix(s string) (string, bool, bool) {
	var hasPrefix bool
	this := true
	if strings.HasPrefix(s, "this") {
		hasPrefix = true
		s = strings.TrimPrefix(s, "this")
	}
	if strings.HasPrefix(s, "last") {
		hasPrefix = true
		this = false
		s = strings.TrimPrefix(s, "last")
	}
	return s, this, hasPrefix
}

func cleanInput(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// RelDate is the relative date in the past, e.g. yesterday would be
// represented as RelDate{0,0,1}.
type RelDate struct {
	Year  uint
	Month uint
	Day   uint
}

func (d RelDate) Apply(t time.Time) time.Time {
	return t.AddDate(-int(d.Year), -int(d.Month), -int(d.Day))
}

// ParseRelativeDate parses a string of relative terms into a DateAdd.
//
// Syntax: N (year|month|week|day) ..
//
// The following are valid inputs:
// 5weeks1day
// 5w1d
//
// Adapted from the Go stdlib in src/time/format.go
func RelativeDate(s string) (RelDate, error) {
	s0 := s
	s = cleanInput(s)
	var da RelDate
	for s != "" {
		var n uint

		var err error

		// expect an integer
		if !('0' <= s[0] && s[0] <= '9') {
			return da, fmt.Errorf("not a valid relative term: %s",
				s0)
		}

		// consume integer
		n, s, err = leadingInt(s)
		if err != nil {
			return da, fmt.Errorf("cannot read integer in %s",
				s0)
		}

		// consume the units
		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			return da, fmt.Errorf("missing unit in %s", s0)
		}

		u := s[:i]
		s = s[i:]
		switch u[0] {
		case 'y':
			da.Year += n
		case 'm':
			da.Month += n
		case 'w':
			da.Day += 7 * n
		case 'd':
			da.Day += n
		default:
			return da, fmt.Errorf("unknown unit %s in %s", u, s0)
		}

	}

	return da, nil
}

func hasUnit(s string) (has bool) {
	for _, u := range "ymwd" {
		if strings.Contains(s, string(u)) {
			return true
		}
	}
	return false
}

// leadingInt parses and returns the leading integer in s.
//
// Adapted from the Go stdlib in src/time/format.go
func leadingInt(s string) (x uint, rem string, err error) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		x = x*10 + uint(c) - '0'
	}
	return x, s[i:], nil
}

func ensureRangeOp(s string) string {
	if strings.Contains(s, "..") {
		return s
	}
	s0 := s
	for _, m := range []string{"this", "last"} {
		for _, u := range []string{"year", "month", "week"} {
			term := m + u
			if strings.Contains(s, term) {
				if m == "last" {
					return s0 + "..this" + u
				} else {
					return s0 + ".."
				}
			}
		}
	}
	return s0
}
