package parse_test

import (
	"os"
	"testing"

	"git.sr.ht/~rjarry/aerc/lib/parse"
	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	//nolint:errcheck // we'll fail the test if this fails
	_ = os.Setenv("COLORTERM", "truecolor")
	tests := []struct {
		name           string
		input          string
		expectedString string
		expectedLen    int
	}{
		{
			name:           "no style",
			input:          "hello, world",
			expectedString: "hello, world",
			expectedLen:    12,
		},
		{
			name:           "bold",
			input:          "\x1b[1mhello, world",
			expectedString: "\x1b[m\x1b[1mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "dim",
			input:          "\x1b[2mhello, world",
			expectedString: "\x1b[m\x1b[2mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "bold and dim",
			input:          "\x1b[1;2mhello, world",
			expectedString: "\x1b[m\x1b[1m\x1b[2mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "italic",
			input:          "\x1b[3mhello, world",
			expectedString: "\x1b[m\x1b[3mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "underline",
			input:          "\x1b[4mhello, world",
			expectedString: "\x1b[m\x1b[4mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "blink",
			input:          "\x1b[5mhello, world",
			expectedString: "\x1b[m\x1b[5mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "fast blink",
			input:          "\x1b[6mhello, world",
			expectedString: "\x1b[m\x1b[5mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "reverse",
			input:          "\x1b[7mhello, world",
			expectedString: "\x1b[m\x1b[7mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "hidden",
			input:          "\x1b[8mhello, world",
			expectedString: "hello, world",
			expectedLen:    12,
		},
		{
			name:           "strikethrough",
			input:          "\x1b[9mhello, world",
			expectedString: "\x1b[m\x1b[9mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "bold hello, normal world",
			input:          "\x1b[1mhello, \x1b[21mworld",
			expectedString: "\x1b[m\x1b[1mhello, \x1b[mworld\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "bold hello, normal world v2",
			input:          "\x1b[1mhello, \x1b[mworld",
			expectedString: "\x1b[m\x1b[1mhello, \x1b[mworld\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "8 bit color: foreground",
			input:          "\x1b[30mhello, world",
			expectedString: "\x1b[m\x1b[38;5;0mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "8 bit color: background",
			input:          "\x1b[41mhello, world",
			expectedString: "\x1b[m\x1b[48;5;1mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "8 bit color: foreground and background",
			input:          "\x1b[31;41mhello, world",
			expectedString: "\x1b[m\x1b[38;5;1;48;5;1mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "16 bit color: foreground",
			input:          "\x1b[90mhello, world",
			expectedString: "\x1b[m\x1b[38;5;8mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "16 bit color: background",
			input:          "\x1b[101mhello, world",
			expectedString: "\x1b[m\x1b[48;5;9mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "16 bit color: foreground and background",
			input:          "\x1b[91;101mhello, world",
			expectedString: "\x1b[m\x1b[38;5;9;48;5;9mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "256 color: foreground",
			input:          "\x1b[38;5;2mhello, world",
			expectedString: "\x1b[m\x1b[38;5;2mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "256 color: foreground",
			input:          "\x1b[38;5;132mhello, world",
			expectedString: "\x1b[m\x1b[38;5;132mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "256 color: background",
			input:          "\x1b[48;5;132mhello, world",
			expectedString: "\x1b[m\x1b[48;5;132mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "256 color: foreground and background",
			input:          "\x1b[38;5;20;48;5;20mhello, world",
			expectedString: "\x1b[m\x1b[38;5;20;48;5;20mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "256 color: background",
			input:          "\x1b[48;5;2mhello, world",
			expectedString: "\x1b[m\x1b[48;5;2mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "true color: foreground",
			input:          "\x1b[38;2;0;0;0mhello, world",
			expectedString: "\x1b[m\x1b[38;2;0;0;0mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "true color: foreground with color space",
			input:          "\x1b[38;2;;0;0;0mhello, world",
			expectedString: "\x1b[m\x1b[38;2;0;0;0mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "true color: foreground with color space and colons",
			input:          "\x1b[38:2::0:0:0mhello, world",
			expectedString: "\x1b[m\x1b[38;2;0;0;0mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "true color: background",
			input:          "\x1b[48;2;0;0;0mhello, world",
			expectedString: "\x1b[m\x1b[48;2;0;0;0mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "true color: background with color space",
			input:          "\x1b[48;2;;0;0;0mhello, world",
			expectedString: "\x1b[m\x1b[48;2;0;0;0mhello, world\x1b[m",
			expectedLen:    12,
		},
		{
			name:           "true color: foreground and background",
			input:          "\x1b[38;2;200;200;200;48;2;0;0;0mhello, world",
			expectedString: "\x1b[m\x1b[38;2;200;200;200;48;2;0;0;0mhello, world\x1b[m",
			expectedLen:    12,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := parse.ParseANSI(test.input)
			assert.Equal(t, test.expectedString, buf.String())
			assert.Equal(t, test.expectedLen, buf.Len())
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedString string
	}{
		{
			name:           "no style, truncate at 5",
			input:          "hello, world",
			expectedString: "hello",
		},
		{
			name:           "bold, truncate at 5",
			input:          "\x1b[1mhello, world",
			expectedString: "\x1b[m\x1b[1mhello\x1b[m",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := parse.ParseANSI(test.input)
			assert.Equal(t, test.expectedString, buf.Truncate(5, 0))
		})
	}
}

func TestTruncateHead(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedString string
		expectedLen    int
	}{
		{
			name:           "no style, truncate head at 5",
			input:          "hello, world",
			expectedString: "world",
		},
		{
			name:           "bold, truncate head at 5",
			input:          "\x1b[1mhello, world",
			expectedString: "\x1b[m\x1b[1mworld\x1b[m",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buf := parse.ParseANSI(test.input)
			assert.Equal(t, test.expectedString, buf.TruncateHead(5, 0))
		})
	}
}
