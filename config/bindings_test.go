package config

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
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
		{tcell.ModNone, tcell.KeyRune, 'a'},
	}, BINDING_INCOMPLETE, "")
	test([]KeyStroke{
		{tcell.ModNone, tcell.KeyRune, 'a'},
		{tcell.ModNone, tcell.KeyRune, 'b'},
		{tcell.ModNone, tcell.KeyRune, 'c'},
	}, BINDING_FOUND, ":abc")
	test([]KeyStroke{
		{tcell.ModNone, tcell.KeyRune, 'c'},
		{tcell.ModNone, tcell.KeyRune, 'b'},
		{tcell.ModNone, tcell.KeyRune, 'a'},
	}, BINDING_FOUND, ":cba")
	test([]KeyStroke{
		{tcell.ModNone, tcell.KeyRune, 'f'},
		{tcell.ModNone, tcell.KeyRune, 'o'},
	}, BINDING_INCOMPLETE, "")
	test([]KeyStroke{
		{tcell.ModNone, tcell.KeyRune, '4'},
		{tcell.ModNone, tcell.KeyRune, '0'},
		{tcell.ModNone, tcell.KeyRune, '4'},
	}, BINDING_NOT_FOUND, "")

	add("<C-a>", "c-a")
	test([]KeyStroke{
		{tcell.ModCtrl, tcell.KeyCtrlA, 0},
	}, BINDING_FOUND, "c-a")
}
