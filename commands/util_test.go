package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletePath(t *testing.T) {
	os.Chdir("testdata")
	defer os.Chdir("..")

	vectors := []struct {
		arg           string
		onlyDirs      bool
		fuzzyComplete bool
		expected      []string
	}{
		{
			arg:      "",
			expected: []string{"Foobar", "baz/", "foo.ini", "foo/"},
		},
		{
			arg:      "",
			onlyDirs: true,
			expected: []string{"baz/", "foo/"},
		},
		{
			arg:      ".",
			expected: []string{".hidden/", ".keep-me"},
		},
		{
			arg:      "fo",
			expected: []string{"Foobar", "foo.ini", "foo/"},
		},
		{
			arg:      "Fo",
			expected: []string{"Foobar"},
		},
		{
			arg:      "..",
			expected: []string{"../"},
		},
		{
			arg:      "../..",
			expected: []string{"../../"},
		},
		{
			arg: "../testdata/",
			expected: []string{
				"../testdata/Foobar",
				"../testdata/baz/",
				"../testdata/foo.ini",
				"../testdata/foo/",
			},
		},
		{
			arg:      "../testdata/f",
			onlyDirs: true,
			expected: []string{"../testdata/foo/"},
		},
		{
			arg:      "oo",
			expected: []string{},
		},
		{
			arg:           "oo",
			fuzzyComplete: true,
			expected:      []string{"Foobar", "foo.ini", "foo/"},
		},
		{
			arg:      "../testdata/oo",
			expected: []string{},
		},
		{
			arg:           "../testdata/oo",
			fuzzyComplete: true,
			expected:      []string{"../testdata/Foobar", "../testdata/foo.ini", "../testdata/foo/"},
		},
	}
	for _, vec := range vectors {
		t.Run(vec.arg, func(t *testing.T) {
			res := completePath(vec.arg, vec.onlyDirs, vec.fuzzyComplete)
			assert.Equal(t, vec.expected, res)
		})
	}
}
