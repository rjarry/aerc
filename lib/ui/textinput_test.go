package ui

import "testing"

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
			if string(textinput.text) != test.expected {
				t.Errorf("word was deleted incorrectly: got %s but expected %s", string(textinput.text), test.expected)
			}
		})
	}
}
