package lib_test

import (
	"testing"
	"time"

	"git.sr.ht/~rjarry/aerc/worker/lib"
)

func TestParseDateRange(t *testing.T) {
	dateFmt := "2006-01-02"
	parse := func(s string) time.Time { d, _ := time.Parse(dateFmt, s); return d }
	tests := []struct {
		s     string
		start time.Time
		end   time.Time
	}{
		{
			s:     "2022-11-01",
			start: parse("2022-11-01"),
			end:   parse("2022-11-02"),
		},
		{
			s:     "2022-11-01..",
			start: parse("2022-11-01"),
		},
		{
			s:   "..2022-11-05",
			end: parse("2022-11-05"),
		},
		{
			s:     "2022-11-01..2022-11-05",
			start: parse("2022-11-01"),
			end:   parse("2022-11-05"),
		},
	}

	for _, test := range tests {
		start, end, err := lib.ParseDateRange(test.s)
		if err != nil {
			t.Errorf("ParseDateRange return error for %s: %v",
				test.s, err)
		}

		if !start.Equal(test.start) {
			t.Errorf("wrong start date; expected %v, got %v",
				test.start, start)
		}

		if !end.Equal(test.end) {
			t.Errorf("wrong end date; expected %v, got %v",
				test.end, end)
		}
	}
}
