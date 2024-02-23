//go:build notmuch
// +build notmuch

package notmuch

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-maildir"
)

func TestFilterForStrategy(t *testing.T) {
	tests := []struct {
		filenames   []string
		strategy    types.MultiFileStrategy
		curDir      string
		expectedAct []string
		expectedDel []string
		expectedErr bool
	}{
		// if there's only one file, always act on it
		{
			filenames:   []string{"/h/j/m/A/cur/a.b.c:2,"},
			strategy:    types.Refuse,
			curDir:      "/h/j/m/B",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{},
		},
		{
			filenames:   []string{"/h/j/m/A/cur/a.b.c:2,"},
			strategy:    types.ActAll,
			curDir:      "/h/j/m/B",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{},
		},
		{
			filenames:   []string{"/h/j/m/A/cur/a.b.c:2,"},
			strategy:    types.ActOne,
			curDir:      "/h/j/m/B",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{},
		},
		{
			filenames:   []string{"/h/j/m/A/cur/a.b.c:2,"},
			strategy:    types.ActOneDelRest,
			curDir:      "/h/j/m/B",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{},
		},
		{
			filenames:   []string{"/h/j/m/A/cur/a.b.c:2,"},
			strategy:    types.ActDir,
			curDir:      "/h/j/m/B",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{},
		},
		{
			filenames:   []string{"/h/j/m/A/cur/a.b.c:2,"},
			strategy:    types.ActDirDelRest,
			curDir:      "/h/j/m/B",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{},
		},

		// follow strategy for multiple files
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy:    types.Refuse,
			curDir:      "/h/j/m/B",
			expectedErr: true,
		},
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy: types.ActAll,
			curDir:   "/h/j/m/B",
			expectedAct: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			expectedDel: []string{},
		},
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy:    types.ActOne,
			curDir:      "/h/j/m/B",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{},
		},
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy:    types.ActOneDelRest,
			curDir:      "/h/j/m/B",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
		},
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy: types.ActDir,
			curDir:   "/h/j/m/B",
			expectedAct: []string{
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
			},
			expectedDel: []string{},
		},
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy: types.ActDirDelRest,
			curDir:   "/h/j/m/B",
			expectedAct: []string{
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
			},
			expectedDel: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/C/new/d.e.f",
			},
		},

		// refuse to act on multiple files for ActDir and friends if
		// no current dir is provided
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy:    types.ActDir,
			curDir:      "",
			expectedErr: true,
		},
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy:    types.ActDirDelRest,
			curDir:      "",
			expectedErr: true,
		},

		// act on multiple files w/o current dir for other strategies
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy: types.ActAll,
			curDir:   "",
			expectedAct: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			expectedDel: []string{},
		},
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy:    types.ActOne,
			curDir:      "",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{},
		},
		{
			filenames: []string{
				"/h/j/m/A/cur/a.b.c:2,",
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
			strategy:    types.ActOneDelRest,
			curDir:      "",
			expectedAct: []string{"/h/j/m/A/cur/a.b.c:2,"},
			expectedDel: []string{
				"/h/j/m/B/new/b.c.d",
				"/h/j/m/B/cur/c.d.e:2,S",
				"/h/j/m/C/new/d.e.f",
			},
		},
	}

	for i, test := range tests {
		act, del, err := filterForStrategy(test.filenames, test.strategy,
			maildir.Dir(test.curDir))

		if test.expectedErr && err == nil {
			t.Errorf("[test %d] got nil, expected error", i)
		}

		if !test.expectedErr && err != nil {
			t.Errorf("[test %d] got %v, expected nil", i, err)
		}

		if !arrEq(act, test.expectedAct) {
			t.Errorf("[test %d] got %v, expected %v", i, act, test.expectedAct)
		}

		if !arrEq(del, test.expectedDel) {
			t.Errorf("[test %d] got %v, expected %v", i, del, test.expectedDel)
		}
	}
}

func arrEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
