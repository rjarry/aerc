package lib

import (
	"fmt"
	"strings"
	"time"
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
func ParseDateRange(s string) (start, end time.Time, err error) {
	s = strings.ReplaceAll(s, " ", "")
	i := strings.Index(s, "..")
	switch {
	case i < 0:
		// single date
		start, err = time.Parse(dateFmt, s)
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
		end, err = time.Parse(dateFmt, s[2:])
		if err != nil {
			err = fmt.Errorf("failed to parse date: %w", err)
			return
		}

	case i > 0:
		// start date first
		start, err = time.Parse(dateFmt, s[:i])
		if err != nil {
			err = fmt.Errorf("failed to parse date: %w", err)
			return
		}
		if len(s[i:]) <= 2 {
			return
		}
		// and end dates if available
		end, err = time.Parse(dateFmt, s[(i+2):])
		if err != nil {
			err = fmt.Errorf("failed to parse date: %w", err)
			return
		}
	}

	return
}
