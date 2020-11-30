package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gdamore/tcell/v2"
)

type KeyStroke struct {
	Key  tcell.Key
	Rune rune
}

type Binding struct {
	Output []KeyStroke
	Input  []KeyStroke
}

type KeyBindings struct {
	bindings []*Binding

	// If false, disable global keybindings in this context
	Globals bool
	// Which key opens the ex line (default is :)
	ExKey KeyStroke
}

const (
	BINDING_FOUND = iota
	BINDING_INCOMPLETE
	BINDING_NOT_FOUND
)

type BindingSearchResult int

func NewKeyBindings() *KeyBindings {
	return &KeyBindings{
		ExKey:   KeyStroke{tcell.KeyRune, ':'},
		Globals: true,
	}
}

func MergeBindings(bindings ...*KeyBindings) *KeyBindings {
	merged := NewKeyBindings()
	for _, b := range bindings {
		merged.bindings = append(merged.bindings, b.bindings...)
	}
	merged.ExKey = bindings[0].ExKey
	merged.Globals = bindings[0].Globals
	return merged
}

func (bindings *KeyBindings) Add(binding *Binding) {
	// TODO: Search for conflicts?
	bindings.bindings = append(bindings.bindings, binding)
}

func (bindings *KeyBindings) GetBinding(
	input []KeyStroke) (BindingSearchResult, []KeyStroke) {

	incomplete := false
	// TODO: This could probably be a sorted list to speed things up
	// TODO: Deal with bindings that share a prefix
	for _, binding := range bindings.bindings {
		if len(binding.Input) < len(input) {
			continue
		}
		for i, stroke := range input {
			if stroke.Key != binding.Input[i].Key {
				goto next
			}
			if stroke.Key == tcell.KeyRune &&
				stroke.Rune != binding.Input[i].Rune {

				goto next
			}
		}
		if len(binding.Input) != len(input) {
			incomplete = true
		} else {
			return BINDING_FOUND, binding.Output
		}
	next:
	}
	if incomplete {
		return BINDING_INCOMPLETE, nil
	}
	return BINDING_NOT_FOUND, nil
}

var (
	keyNames map[string]KeyStroke
)

func ParseKeyStrokes(keystrokes string) ([]KeyStroke, error) {
	var strokes []KeyStroke
	buf := bytes.NewBufferString(keystrokes)
	for {
		tok, _, err := buf.ReadRune()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		// TODO: make it possible to bind to < or > themselves (and default to
		// switching accounts)
		switch tok {
		case '<':
			name, err := buf.ReadString(byte('>'))
			if err == io.EOF {
				return nil, errors.New("Expecting '>'")
			} else if err != nil {
				return nil, err
			} else if name == ">" {
				return nil, errors.New("Expected a key name")
			}
			name = name[:len(name)-1]
			if key, ok := keyNames[strings.ToLower(name)]; ok {
				strokes = append(strokes, key)
			} else {
				return nil, fmt.Errorf("Unknown key '%s'", name)
			}
		case '>':
			return nil, errors.New("Found '>' without '<'")
		default:
			strokes = append(strokes, KeyStroke{
				Key:  tcell.KeyRune,
				Rune: tok,
			})
		}
	}
	return strokes, nil
}

func ParseBinding(input, output string) (*Binding, error) {
	in, err := ParseKeyStrokes(input)
	if err != nil {
		return nil, err
	}
	out, err := ParseKeyStrokes(output)
	if err != nil {
		return nil, err
	}
	return &Binding{
		Input:  in,
		Output: out,
	}, nil
}

func init() {
	keyNames = make(map[string]KeyStroke)
	keyNames["space"] = KeyStroke{tcell.KeyRune, ' '}
	keyNames["semicolon"] = KeyStroke{tcell.KeyRune, ';'}
	keyNames["enter"] = KeyStroke{tcell.KeyEnter, 0}
	keyNames["up"] = KeyStroke{tcell.KeyUp, 0}
	keyNames["down"] = KeyStroke{tcell.KeyDown, 0}
	keyNames["right"] = KeyStroke{tcell.KeyRight, 0}
	keyNames["left"] = KeyStroke{tcell.KeyLeft, 0}
	keyNames["upleft"] = KeyStroke{tcell.KeyUpLeft, 0}
	keyNames["upright"] = KeyStroke{tcell.KeyUpRight, 0}
	keyNames["downleft"] = KeyStroke{tcell.KeyDownLeft, 0}
	keyNames["downright"] = KeyStroke{tcell.KeyDownRight, 0}
	keyNames["center"] = KeyStroke{tcell.KeyCenter, 0}
	keyNames["pgup"] = KeyStroke{tcell.KeyPgUp, 0}
	keyNames["pgdn"] = KeyStroke{tcell.KeyPgDn, 0}
	keyNames["home"] = KeyStroke{tcell.KeyHome, 0}
	keyNames["end"] = KeyStroke{tcell.KeyEnd, 0}
	keyNames["insert"] = KeyStroke{tcell.KeyInsert, 0}
	keyNames["delete"] = KeyStroke{tcell.KeyDelete, 0}
	keyNames["help"] = KeyStroke{tcell.KeyHelp, 0}
	keyNames["exit"] = KeyStroke{tcell.KeyExit, 0}
	keyNames["clear"] = KeyStroke{tcell.KeyClear, 0}
	keyNames["cancel"] = KeyStroke{tcell.KeyCancel, 0}
	keyNames["print"] = KeyStroke{tcell.KeyPrint, 0}
	keyNames["pause"] = KeyStroke{tcell.KeyPause, 0}
	keyNames["backtab"] = KeyStroke{tcell.KeyBacktab, 0}
	keyNames["f1"] = KeyStroke{tcell.KeyF1, 0}
	keyNames["f2"] = KeyStroke{tcell.KeyF2, 0}
	keyNames["f3"] = KeyStroke{tcell.KeyF3, 0}
	keyNames["f4"] = KeyStroke{tcell.KeyF4, 0}
	keyNames["f5"] = KeyStroke{tcell.KeyF5, 0}
	keyNames["f6"] = KeyStroke{tcell.KeyF6, 0}
	keyNames["f7"] = KeyStroke{tcell.KeyF7, 0}
	keyNames["f8"] = KeyStroke{tcell.KeyF8, 0}
	keyNames["f9"] = KeyStroke{tcell.KeyF9, 0}
	keyNames["f10"] = KeyStroke{tcell.KeyF10, 0}
	keyNames["f11"] = KeyStroke{tcell.KeyF11, 0}
	keyNames["f12"] = KeyStroke{tcell.KeyF12, 0}
	keyNames["f13"] = KeyStroke{tcell.KeyF13, 0}
	keyNames["f14"] = KeyStroke{tcell.KeyF14, 0}
	keyNames["f15"] = KeyStroke{tcell.KeyF15, 0}
	keyNames["f16"] = KeyStroke{tcell.KeyF16, 0}
	keyNames["f17"] = KeyStroke{tcell.KeyF17, 0}
	keyNames["f18"] = KeyStroke{tcell.KeyF18, 0}
	keyNames["f19"] = KeyStroke{tcell.KeyF19, 0}
	keyNames["f20"] = KeyStroke{tcell.KeyF20, 0}
	keyNames["f21"] = KeyStroke{tcell.KeyF21, 0}
	keyNames["f22"] = KeyStroke{tcell.KeyF22, 0}
	keyNames["f23"] = KeyStroke{tcell.KeyF23, 0}
	keyNames["f24"] = KeyStroke{tcell.KeyF24, 0}
	keyNames["f25"] = KeyStroke{tcell.KeyF25, 0}
	keyNames["f26"] = KeyStroke{tcell.KeyF26, 0}
	keyNames["f27"] = KeyStroke{tcell.KeyF27, 0}
	keyNames["f28"] = KeyStroke{tcell.KeyF28, 0}
	keyNames["f29"] = KeyStroke{tcell.KeyF29, 0}
	keyNames["f30"] = KeyStroke{tcell.KeyF30, 0}
	keyNames["f31"] = KeyStroke{tcell.KeyF31, 0}
	keyNames["f32"] = KeyStroke{tcell.KeyF32, 0}
	keyNames["f33"] = KeyStroke{tcell.KeyF33, 0}
	keyNames["f34"] = KeyStroke{tcell.KeyF34, 0}
	keyNames["f35"] = KeyStroke{tcell.KeyF35, 0}
	keyNames["f36"] = KeyStroke{tcell.KeyF36, 0}
	keyNames["f37"] = KeyStroke{tcell.KeyF37, 0}
	keyNames["f38"] = KeyStroke{tcell.KeyF38, 0}
	keyNames["f39"] = KeyStroke{tcell.KeyF39, 0}
	keyNames["f40"] = KeyStroke{tcell.KeyF40, 0}
	keyNames["f41"] = KeyStroke{tcell.KeyF41, 0}
	keyNames["f42"] = KeyStroke{tcell.KeyF42, 0}
	keyNames["f43"] = KeyStroke{tcell.KeyF43, 0}
	keyNames["f44"] = KeyStroke{tcell.KeyF44, 0}
	keyNames["f45"] = KeyStroke{tcell.KeyF45, 0}
	keyNames["f46"] = KeyStroke{tcell.KeyF46, 0}
	keyNames["f47"] = KeyStroke{tcell.KeyF47, 0}
	keyNames["f48"] = KeyStroke{tcell.KeyF48, 0}
	keyNames["f49"] = KeyStroke{tcell.KeyF49, 0}
	keyNames["f50"] = KeyStroke{tcell.KeyF50, 0}
	keyNames["f51"] = KeyStroke{tcell.KeyF51, 0}
	keyNames["f52"] = KeyStroke{tcell.KeyF52, 0}
	keyNames["f53"] = KeyStroke{tcell.KeyF53, 0}
	keyNames["f54"] = KeyStroke{tcell.KeyF54, 0}
	keyNames["f55"] = KeyStroke{tcell.KeyF55, 0}
	keyNames["f56"] = KeyStroke{tcell.KeyF56, 0}
	keyNames["f57"] = KeyStroke{tcell.KeyF57, 0}
	keyNames["f58"] = KeyStroke{tcell.KeyF58, 0}
	keyNames["f59"] = KeyStroke{tcell.KeyF59, 0}
	keyNames["f60"] = KeyStroke{tcell.KeyF60, 0}
	keyNames["f61"] = KeyStroke{tcell.KeyF61, 0}
	keyNames["f62"] = KeyStroke{tcell.KeyF62, 0}
	keyNames["f63"] = KeyStroke{tcell.KeyF63, 0}
	keyNames["f64"] = KeyStroke{tcell.KeyF64, 0}
	keyNames["c-space"] = KeyStroke{tcell.KeyCtrlSpace, 0}
	keyNames["c-a"] = KeyStroke{tcell.KeyCtrlA, 0}
	keyNames["c-b"] = KeyStroke{tcell.KeyCtrlB, 0}
	keyNames["c-c"] = KeyStroke{tcell.KeyCtrlC, 0}
	keyNames["c-d"] = KeyStroke{tcell.KeyCtrlD, 0}
	keyNames["c-e"] = KeyStroke{tcell.KeyCtrlE, 0}
	keyNames["c-f"] = KeyStroke{tcell.KeyCtrlF, 0}
	keyNames["c-g"] = KeyStroke{tcell.KeyCtrlG, 0}
	keyNames["c-h"] = KeyStroke{tcell.KeyCtrlH, 0}
	keyNames["c-i"] = KeyStroke{tcell.KeyCtrlI, 0}
	keyNames["c-j"] = KeyStroke{tcell.KeyCtrlJ, 0}
	keyNames["c-k"] = KeyStroke{tcell.KeyCtrlK, 0}
	keyNames["c-l"] = KeyStroke{tcell.KeyCtrlL, 0}
	keyNames["c-m"] = KeyStroke{tcell.KeyCtrlM, 0}
	keyNames["c-n"] = KeyStroke{tcell.KeyCtrlN, 0}
	keyNames["c-o"] = KeyStroke{tcell.KeyCtrlO, 0}
	keyNames["c-p"] = KeyStroke{tcell.KeyCtrlP, 0}
	keyNames["c-q"] = KeyStroke{tcell.KeyCtrlQ, 0}
	keyNames["c-r"] = KeyStroke{tcell.KeyCtrlR, 0}
	keyNames["c-s"] = KeyStroke{tcell.KeyCtrlS, 0}
	keyNames["c-t"] = KeyStroke{tcell.KeyCtrlT, 0}
	keyNames["c-u"] = KeyStroke{tcell.KeyCtrlU, 0}
	keyNames["c-v"] = KeyStroke{tcell.KeyCtrlV, 0}
	keyNames["c-w"] = KeyStroke{tcell.KeyCtrlW, 0}
	keyNames["c-x"] = KeyStroke{tcell.KeyCtrlX, rune(tcell.KeyCAN)}
	keyNames["c-y"] = KeyStroke{tcell.KeyCtrlY, 0} // TODO: runes for the rest
	keyNames["c-z"] = KeyStroke{tcell.KeyCtrlZ, 0}
	keyNames["c-]"] = KeyStroke{tcell.KeyCtrlLeftSq, 0}
	keyNames["c-\\"] = KeyStroke{tcell.KeyCtrlBackslash, 0}
	keyNames["c-["] = KeyStroke{tcell.KeyCtrlRightSq, 0}
	keyNames["c-^"] = KeyStroke{tcell.KeyCtrlCarat, 0}
	keyNames["c-_"] = KeyStroke{tcell.KeyCtrlUnderscore, 0}
	keyNames["nul"] = KeyStroke{tcell.KeyNUL, 0}
	keyNames["soh"] = KeyStroke{tcell.KeySOH, 0}
	keyNames["stx"] = KeyStroke{tcell.KeySTX, 0}
	keyNames["etx"] = KeyStroke{tcell.KeyETX, 0}
	keyNames["eot"] = KeyStroke{tcell.KeyEOT, 0}
	keyNames["enq"] = KeyStroke{tcell.KeyENQ, 0}
	keyNames["ack"] = KeyStroke{tcell.KeyACK, 0}
	keyNames["bel"] = KeyStroke{tcell.KeyBEL, 0}
	keyNames["bs"] = KeyStroke{tcell.KeyBS, 0}
	keyNames["tab"] = KeyStroke{tcell.KeyTAB, 0}
	keyNames["lf"] = KeyStroke{tcell.KeyLF, 0}
	keyNames["vt"] = KeyStroke{tcell.KeyVT, 0}
	keyNames["ff"] = KeyStroke{tcell.KeyFF, 0}
	keyNames["cr"] = KeyStroke{tcell.KeyCR, 0}
	keyNames["so"] = KeyStroke{tcell.KeySO, 0}
	keyNames["si"] = KeyStroke{tcell.KeySI, 0}
	keyNames["dle"] = KeyStroke{tcell.KeyDLE, 0}
	keyNames["dc1"] = KeyStroke{tcell.KeyDC1, 0}
	keyNames["dc2"] = KeyStroke{tcell.KeyDC2, 0}
	keyNames["dc3"] = KeyStroke{tcell.KeyDC3, 0}
	keyNames["dc4"] = KeyStroke{tcell.KeyDC4, 0}
	keyNames["nak"] = KeyStroke{tcell.KeyNAK, 0}
	keyNames["syn"] = KeyStroke{tcell.KeySYN, 0}
	keyNames["etb"] = KeyStroke{tcell.KeyETB, 0}
	keyNames["can"] = KeyStroke{tcell.KeyCAN, 0}
	keyNames["em"] = KeyStroke{tcell.KeyEM, 0}
	keyNames["sub"] = KeyStroke{tcell.KeySUB, 0}
	keyNames["esc"] = KeyStroke{tcell.KeyESC, 0}
	keyNames["fs"] = KeyStroke{tcell.KeyFS, 0}
	keyNames["gs"] = KeyStroke{tcell.KeyGS, 0}
	keyNames["rs"] = KeyStroke{tcell.KeyRS, 0}
	keyNames["us"] = KeyStroke{tcell.KeyUS, 0}
	keyNames["del"] = KeyStroke{tcell.KeyDEL, 0}
}
