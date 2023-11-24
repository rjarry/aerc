package patch

import (
	"reflect"
	"testing"
)

func TestPatchApply_ProposeName(t *testing.T) {
	tests := []struct {
		name     string
		exist    []string
		subjects []string
		want     []string
	}{
		{
			name:  "base case",
			exist: nil,
			subjects: []string{
				"[PATCH aerc v3 3/3] notmuch: remove unused code",
				"[PATCH aerc v3 2/3] notmuch: replace notmuch library with internal bindings",
				"[PATCH aerc v3 1/3] notmuch: add notmuch bindings",
			},
			want: []string{"notmuch_v3"},
		},
		{
			name:  "distorted case",
			exist: nil,
			subjects: []string{
				"[PATCH vaerc v3 3/3] notmuch: remove unused code",
				"[PATCH aerc 3v 2/3] notmuch: replace notmuch library with internal bindings",
			},
			want: []string{"notmuch_v1", "notmuch_v3"},
		},
		{
			name:  "invalid patches",
			exist: nil,
			subjects: []string{
				"notmuch: remove unused code",
				": replace notmuch library with internal bindings",
			},
			want: nil,
		},
	}

	for _, test := range tests {
		got := proposePatchName(test.exist, test.subjects)
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("test '%s' failed to propose the correct "+
				"name: got '%v', but want '%v'", test.name,
				got, test.want)
		}
	}
}
