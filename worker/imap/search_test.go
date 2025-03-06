package imap

import (
	"testing"
	"time"

	"git.sr.ht/~rjarry/aerc/worker/types"
)

func Test_translateSearch_ByDate(t *testing.T) {
	tests := []struct {
		name      string
		StartDate time.Time
		EndDate   time.Time
	}{
		{
			name:      "Only StartDate",
			StartDate: time.Now(),
		},
		{
			name:    "Only EndDate",
			EndDate: time.Now(),
		},
		{
			name:      "Both dates",
			StartDate: time.Now(),
			EndDate:   time.Now(),
		},
	}
	for _, test := range tests {
		crit := &types.SearchCriteria{
			StartDate: test.StartDate,
			EndDate:   test.EndDate,
		}
		sc := translateSearch(crit)
		if sc.SentSince != test.StartDate {
			t.Errorf("test '%s' failed: got: '%s', but wanted: '%s'",
				test.name, sc.SentSince, test.StartDate)
		}
		if sc.SentBefore != test.EndDate {
			t.Errorf("test '%s' failed: got: '%s', but wanted: '%s'",
				test.name, sc.SentBefore, test.EndDate)
		}
	}
}
