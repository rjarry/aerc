package pama_test

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/pama"
)

func TestFromSubject(t *testing.T) {
	tests := []struct {
		s    string
		want string
	}{
		{
			s:    "[PATCH aerc] pama: new patch",
			want: "aerc",
		},
		{
			s:    "[PATCH aerc v2] pama: new patch",
			want: "aerc",
		},
		{
			s:    "[PATCH aerc 1/2] pama: new patch",
			want: "aerc",
		},
		{
			s:    "[Patch aerc] pama: new patch",
			want: "aerc",
		},
		{
			s:    "[patch aerc] pama: new patch",
			want: "aerc",
		},
		{
			s:    "[RFC PATCH aerc] pama: new patch",
			want: "aerc",
		},
		{
			s:    "[DRAFT PATCH aerc] pama: new patch",
			want: "aerc",
		},
		{
			s:    "RE: [PATCH aerc v1] pama: new patch",
			want: "aerc",
		},
		{
			s:    "[PATCH] pama: new patch",
			want: "",
		},
		{
			s:    "just a subject line",
			want: "",
		},
		{
			s:    "just a subject line with unrelated [asdf aerc v1]",
			want: "",
		},
	}

	for _, test := range tests {
		got := pama.FromSubject(test.s)
		if got != test.want {
			t.Errorf("failed to get name from '%s': "+
				"got '%s', want '%s'", test.s, got, test.want)
		}
	}
}
