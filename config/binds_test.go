package config

import (
	"fmt"
	"testing"

	"git.sr.ht/~rockorager/vaxis"
	"github.com/stretchr/testify/assert"
)

func TestGetBinding(t *testing.T) {
	assert := assert.New(t)

	bindings := NewKeyBindings()
	add := func(binding, cmd string) {
		b, _ := ParseBinding(binding, cmd, "")
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
		{vaxis.ModifierMask(0), 'a'},
	}, BINDING_INCOMPLETE, "")
	test([]KeyStroke{
		{vaxis.ModifierMask(0), 'a'},
		{vaxis.ModifierMask(0), 'b'},
		{vaxis.ModifierMask(0), 'c'},
	}, BINDING_FOUND, ":abc")
	test([]KeyStroke{
		{vaxis.ModifierMask(0), 'c'},
		{vaxis.ModifierMask(0), 'b'},
		{vaxis.ModifierMask(0), 'a'},
	}, BINDING_FOUND, ":cba")
	test([]KeyStroke{
		{vaxis.ModifierMask(0), 'f'},
		{vaxis.ModifierMask(0), 'o'},
	}, BINDING_INCOMPLETE, "")
	test([]KeyStroke{
		{vaxis.ModifierMask(0), '4'},
		{vaxis.ModifierMask(0), '0'},
		{vaxis.ModifierMask(0), '4'},
	}, BINDING_NOT_FOUND, "")

	add("<C-a>", "c-a")
	add("<C-Down>", ":next")
	add("<C-PgUp>", ":prev")
	add("<C-Enter>", ":open")
	test([]KeyStroke{
		{vaxis.ModCtrl, 'a'},
	}, BINDING_FOUND, "c-a")
	test([]KeyStroke{
		{vaxis.ModCtrl, vaxis.KeyDown},
	}, BINDING_FOUND, ":next")
	test([]KeyStroke{
		{vaxis.ModCtrl, vaxis.KeyPgUp},
	}, BINDING_FOUND, ":prev")
	test([]KeyStroke{
		{vaxis.ModCtrl, vaxis.KeyPgDown},
	}, BINDING_NOT_FOUND, "")
	test([]KeyStroke{
		{vaxis.ModCtrl, vaxis.KeyEnter},
	}, BINDING_FOUND, ":open")
}
