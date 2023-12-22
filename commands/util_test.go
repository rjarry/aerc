package commands_test

import (
	"os"
	"testing"

	"git.sr.ht/~rjarry/aerc/commands"
	"github.com/stretchr/testify/assert"
)

func TestCompletePath(t *testing.T) {
	os.Chdir("testdata")
	defer os.Chdir("..")

	vectors := []struct {
		arg      string
		onlyDirs bool
		expected []string
	}{
		{
			arg:      "",
			expected: []string{"baz/", "foo.ini", "foo/"},
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
			expected: []string{"foo.ini", "foo/"},
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
	}
	for _, vec := range vectors {
		t.Run(vec.arg, func(t *testing.T) {
			res := commands.CompletePath(vec.arg, vec.onlyDirs)
			assert.Equal(t, vec.expected, res)
		})
	}
}
