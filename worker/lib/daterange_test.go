package lib_test

import (
	"reflect"
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

func TestParseRelativeDate(t *testing.T) {
	tests := []struct {
		s    string
		want lib.RelDate
	}{
		{
			s:    "5 weeks 1 day",
			want: lib.RelDate{Year: 0, Month: 0, Day: 5*7 + 1},
		},
		{
			s:    "5_weeks 1_day",
			want: lib.RelDate{Year: 0, Month: 0, Day: 5*7 + 1},
		},
		{
			s:    "5weeks1day",
			want: lib.RelDate{Year: 0, Month: 0, Day: 5*7 + 1},
		},
		{
			s:    "5w1d",
			want: lib.RelDate{Year: 0, Month: 0, Day: 5*7 + 1},
		},
		{
			s:    "5y4m3w1d",
			want: lib.RelDate{Year: 5, Month: 4, Day: 3*7 + 1},
		},
	}

	for _, test := range tests {
		da, err := lib.ParseRelativeDate(test.s)
		if err != nil {
			t.Errorf("ParseRelativeDate return error for %s: %v",
				test.s, err)
		}

		if !reflect.DeepEqual(da, test.want) {
			t.Errorf("results don't match. expected %v, got %v",
				test.want, da)
		}
	}
}
