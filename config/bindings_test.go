package config

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell"
	"github.com/stretchr/testify/assert"
)

func TestGetBinding(t *testing.T) {
	assert := assert.New(t)

	bindings := NewKeyBindings()
	add := func(binding, cmd string) {
		b, _ := ParseBinding(binding, cmd)
		bindings.Add(b)
	}

	add("abc", ":abc")
	add("cba", ":cba")
	add("foo", ":foo")
	add("bar", ":bar")

	test := func(input []KeyStroke, result int, output string) {
		_output, _ := ParseKeyStrokes(output)
		r, out := bindings.GetBinding(input)
		assert.Equal(result, int(r), fmt.Sprintf(
			"%s: Expected result %d, got %d", output, result, r))
		assert.Equal(_output, out, fmt.Sprintf(
			"%s: Expected output %v, got %v", output, _output, out))
	}

	test([]KeyStroke{
		{tcell.KeyRune, 'a'},
	}, BINDING_INCOMPLETE, "")
	test([]KeyStroke{
		{tcell.KeyRune, 'a'},
		{tcell.KeyRune, 'b'},
		{tcell.KeyRune, 'c'},
	}, BINDING_FOUND, ":abc")
	test([]KeyStroke{
		{tcell.KeyRune, 'c'},
		{tcell.KeyRune, 'b'},
		{tcell.KeyRune, 'a'},
	}, BINDING_FOUND, ":cba")
	test([]KeyStroke{
		{tcell.KeyRune, 'f'},
		{tcell.KeyRune, 'o'},
	}, BINDING_INCOMPLETE, "")
	test([]KeyStroke{
		{tcell.KeyRune, '4'},
		{tcell.KeyRune, '0'},
		{tcell.KeyRune, '4'},
	}, BINDING_NOT_FOUND, "")

	add("<C-a>", "c-a")
	test([]KeyStroke{
		{tcell.KeyCtrlA, 0},
	}, BINDING_FOUND, "c-a")
}
