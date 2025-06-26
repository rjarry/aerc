package ui

import "testing"

func TestWordStarts(t *testing.T) {
	// Test both wordStart and nextWordStart
	type string_case struct {
		name           string
		cursor_pos     int
		this_start_pos int
		next_start_pos int
	}
	tests := []struct {
		text  string
		cases []string_case
	}{
		{
			text: "hello world",
			cases: []string_case{
				{
					name:           "hello-world",
					cursor_pos:     0,
					this_start_pos: 0,
					next_start_pos: 6,
				},
				{
					name:           "hello-world-middle",
					cursor_pos:     3,
					this_start_pos: 0,
					next_start_pos: 6,
				},
				{
					name:           "hello-world-second-word",
					cursor_pos:     7,
					this_start_pos: 6,
					next_start_pos: len("hello world"),
				},
				{
					name:           "hello-world-npos",
					cursor_pos:     len("hello world"),
					this_start_pos: 6,
					next_start_pos: len("hello world"),
				},
				{
					name:           "hello-world-npos-1",
					cursor_pos:     len("hello world") - 1,
					this_start_pos: 6,
					next_start_pos: len("hello world"),
				},
			},
		},
		{
			text: "    hello   ",
			cases: []string_case{
				{
					name:           "space-around-mid",
					cursor_pos:     6,
					this_start_pos: 4,
					next_start_pos: 9,
				},
				{
					name:           "space-around-start",
					cursor_pos:     4,
					this_start_pos: 0,
					next_start_pos: 9,
				},
				{
					name:           "space-around-end",
					cursor_pos:     9,
					this_start_pos: 4,
					next_start_pos: len("    hello   "),
				},
			},
		},
		{
			text: "",
			cases: []string_case{
				{
					name:           "empty",
					cursor_pos:     0,
					this_start_pos: 0,
					next_start_pos: 0,
				},
			},
		},
		{
			text: " 'hello '  world",
			cases: []string_case{
				{
					name:           "space-and-quote-midword",
					cursor_pos:     len(" 'he"),
					this_start_pos: len(" '"),
					next_start_pos: len(" 'hello "),
				},
				{
					name:           "space-and-quote-startword",
					cursor_pos:     len(" '"),
					this_start_pos: len(" "),
					next_start_pos: len(" 'hello "),
				},
				{
					name:           "space-and-quote-atquote",
					cursor_pos:     len(" "),
					this_start_pos: len(""),
					next_start_pos: len(" '"),
				},
				{
					name:           "space-and-quote-at-second-quote",
					cursor_pos:     len(" 'hello "),
					this_start_pos: len(" '"),
					next_start_pos: len(" 'hello '  "),
				},
				{
					name:           "space-and-quote-after-second-quote",
					cursor_pos:     len(" 'hello '"),
					this_start_pos: len(" 'hello "),
					next_start_pos: len(" 'hello '  "),
				},
				{
					name:           "space-and-quote-after-second-quote",
					cursor_pos:     len(" 'hello '  "),
					this_start_pos: len(" 'hello "),
					next_start_pos: len(" 'hello '  world"),
				},
				{
					name:           "space-and-quote-at-last-word",
					cursor_pos:     len(" 'hello '  w"),
					this_start_pos: len(" 'hello '  "),
					next_start_pos: len(" 'hello '  world"),
				},
				{
					name:           "space-and-quote-start",
					cursor_pos:     0,
					this_start_pos: 0,
					next_start_pos: len(" "),
				},
			},
		},
		{
			text: " /hello",
			cases: []string_case{
				{
					name:           "space-and-path-midword",
					cursor_pos:     4,
					this_start_pos: 2,
					next_start_pos: len(" 'hello"),
				},
				{
					name:           "space-and-path-startword",
					cursor_pos:     2,
					this_start_pos: 1,
					next_start_pos: len(" 'hello"),
				},
				{
					name:           "space-and-path-atpath",
					cursor_pos:     1,
					this_start_pos: 0,
					next_start_pos: 2,
				},
				{
					name:           "space-and-path-start",
					cursor_pos:     0,
					this_start_pos: 0,
					next_start_pos: 1,
				},
			},
		},
	}
	for _, strt := range tests {
		for _, test := range strt.cases {
			t.Run(test.name, func(t *testing.T) {
				textinput := NewTextInput(strt.text, nil)
				textinput.index = test.cursor_pos
				this_word_start := textinput.wordStart()
				if this_word_start != test.this_start_pos {
					t.Errorf("cursor was moved (this word start) incorrectly: got %d but expected %d (test string: \"%s\")", this_word_start, test.this_start_pos, charactersToString(textinput.text))
				}
				next_word_start := textinput.nextWordStart()
				if next_word_start != test.next_start_pos {
					t.Errorf("cursor was moved (next word start) incorrectly: got %d but expected %d (test string: \"%s\")", next_word_start, test.next_start_pos, charactersToString(textinput.text))
				}
			})
		}
	}
}

func TestDeleteWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "hello-world",
			text:     "hello world",
			expected: "hello ",
		},
		{
			name:     "empty",
			text:     "",
			expected: "",
		},
		{
			name:     "quoted",
			text:     `"hello"`,
			expected: `"hello`,
		},
		{
			name:     "hello-and-space",
			text:     "hello ",
			expected: "",
		},
		{
			name:     "space-and-hello",
			text:     " hello",
			expected: " ",
		},
		{
			name:     "only-quote",
			text:     `"`,
			expected: "",
		},
		{
			name:     "only-space",
			text:     " ",
			expected: "",
		},
		{
			name:     "space-and-quoted",
			text:     " 'hello",
			expected: " '",
		},
		{
			name:     "paths",
			text:     "foo/bar/baz",
			expected: "foo/bar/",
		},
		{
			name:     "space-and-paths",
			text:     " /foo",
			expected: " /",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			textinput := NewTextInput(test.text, nil)
			textinput.deleteWord()
			if charactersToString(textinput.text) != test.expected {
				t.Errorf("word was deleted incorrectly: got %s but expected %s", charactersToString(textinput.text), test.expected)
			}
		})
	}
}
