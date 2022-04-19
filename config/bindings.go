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
	Modifiers tcell.ModMask
	Key       tcell.Key
	Rune      rune
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
		ExKey:   KeyStroke{tcell.ModNone, tcell.KeyRune, ':'},
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

func (config AercConfig) MergeContextualBinds(baseBinds *KeyBindings,
	contextType ContextType, reg string, bindCtx string) *KeyBindings {

	bindings := baseBinds
	for _, contextualBind := range config.ContextualBinds {
		if contextualBind.ContextType != contextType {
			continue
		}

		if !contextualBind.Regex.Match([]byte(reg)) {
			continue
		}

		if contextualBind.BindContext != bindCtx {
			continue
		}

		bindings = MergeBindings(contextualBind.Bindings, bindings)
	}
	return bindings
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
			if stroke.Modifiers != binding.Input[i].Modifiers {
				goto next
			}
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

func (bindings *KeyBindings) GetReverseBindings(output []KeyStroke) [][]KeyStroke {
	var inputs [][]KeyStroke

	for _, binding := range bindings.bindings {
		if len(binding.Output) != len(output) {
			continue
		}
		for i, stroke := range output {
			if stroke.Modifiers != binding.Output[i].Modifiers {
				goto next
			}
			if stroke.Key != binding.Output[i].Key {
				goto next
			}
			if stroke.Key == tcell.KeyRune && stroke.Rune != binding.Output[i].Rune {
				goto next
			}
		}
		inputs = append(inputs, binding.Input)
	next:
	}
	return inputs
}

func FormatKeyStrokes(keystrokes []KeyStroke) string {
	var sb strings.Builder

	for _, stroke := range keystrokes {
		s := ""
		for name, ks := range keyNames {
			if ks.Modifiers == stroke.Modifiers && ks.Key == stroke.Key && ks.Rune == stroke.Rune {
				if name == "cr" {
					name = "enter"
				}
				s = fmt.Sprintf("<%s>", name)
				break
			}
		}
		if s == "" && stroke.Key == tcell.KeyRune {
			s = string(stroke.Rune)
		}
		sb.WriteString(s)
	}

	return sb.String()
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
		case '\\':
			tok, _, err = buf.ReadRune()
			if err == io.EOF {
				tok = '\\'
			} else if err != nil {
				return nil, err
			}
			fallthrough
		default:
			strokes = append(strokes, KeyStroke{
				Modifiers: tcell.ModNone,
				Key:       tcell.KeyRune,
				Rune:      tok,
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
	keyNames["space"] = KeyStroke{tcell.ModNone, tcell.KeyRune, ' '}
	keyNames["semicolon"] = KeyStroke{tcell.ModNone, tcell.KeyRune, ';'}
	keyNames["enter"] = KeyStroke{tcell.ModNone, tcell.KeyEnter, 0}
	keyNames["c-enter"] = KeyStroke{tcell.ModCtrl, tcell.KeyEnter, 0}
	keyNames["up"] = KeyStroke{tcell.ModNone, tcell.KeyUp, 0}
	keyNames["c-up"] = KeyStroke{tcell.ModCtrl, tcell.KeyUp, 0}
	keyNames["down"] = KeyStroke{tcell.ModNone, tcell.KeyDown, 0}
	keyNames["c-down"] = KeyStroke{tcell.ModCtrl, tcell.KeyDown, 0}
	keyNames["right"] = KeyStroke{tcell.ModNone, tcell.KeyRight, 0}
	keyNames["c-right"] = KeyStroke{tcell.ModCtrl, tcell.KeyRight, 0}
	keyNames["left"] = KeyStroke{tcell.ModNone, tcell.KeyLeft, 0}
	keyNames["c-left"] = KeyStroke{tcell.ModCtrl, tcell.KeyLeft, 0}
	keyNames["upleft"] = KeyStroke{tcell.ModNone, tcell.KeyUpLeft, 0}
	keyNames["upright"] = KeyStroke{tcell.ModNone, tcell.KeyUpRight, 0}
	keyNames["downleft"] = KeyStroke{tcell.ModNone, tcell.KeyDownLeft, 0}
	keyNames["downright"] = KeyStroke{tcell.ModNone, tcell.KeyDownRight, 0}
	keyNames["center"] = KeyStroke{tcell.ModNone, tcell.KeyCenter, 0}
	keyNames["pgup"] = KeyStroke{tcell.ModNone, tcell.KeyPgUp, 0}
	keyNames["c-pgup"] = KeyStroke{tcell.ModCtrl, tcell.KeyPgUp, 0}
	keyNames["pgdn"] = KeyStroke{tcell.ModNone, tcell.KeyPgDn, 0}
	keyNames["c-pgdn"] = KeyStroke{tcell.ModCtrl, tcell.KeyPgDn, 0}
	keyNames["home"] = KeyStroke{tcell.ModNone, tcell.KeyHome, 0}
	keyNames["end"] = KeyStroke{tcell.ModNone, tcell.KeyEnd, 0}
	keyNames["insert"] = KeyStroke{tcell.ModNone, tcell.KeyInsert, 0}
	keyNames["delete"] = KeyStroke{tcell.ModNone, tcell.KeyDelete, 0}
	keyNames["help"] = KeyStroke{tcell.ModNone, tcell.KeyHelp, 0}
	keyNames["exit"] = KeyStroke{tcell.ModNone, tcell.KeyExit, 0}
	keyNames["clear"] = KeyStroke{tcell.ModNone, tcell.KeyClear, 0}
	keyNames["cancel"] = KeyStroke{tcell.ModNone, tcell.KeyCancel, 0}
	keyNames["print"] = KeyStroke{tcell.ModNone, tcell.KeyPrint, 0}
	keyNames["pause"] = KeyStroke{tcell.ModNone, tcell.KeyPause, 0}
	keyNames["backtab"] = KeyStroke{tcell.ModNone, tcell.KeyBacktab, 0}
	keyNames["f1"] = KeyStroke{tcell.ModNone, tcell.KeyF1, 0}
	keyNames["f2"] = KeyStroke{tcell.ModNone, tcell.KeyF2, 0}
	keyNames["f3"] = KeyStroke{tcell.ModNone, tcell.KeyF3, 0}
	keyNames["f4"] = KeyStroke{tcell.ModNone, tcell.KeyF4, 0}
	keyNames["f5"] = KeyStroke{tcell.ModNone, tcell.KeyF5, 0}
	keyNames["f6"] = KeyStroke{tcell.ModNone, tcell.KeyF6, 0}
	keyNames["f7"] = KeyStroke{tcell.ModNone, tcell.KeyF7, 0}
	keyNames["f8"] = KeyStroke{tcell.ModNone, tcell.KeyF8, 0}
	keyNames["f9"] = KeyStroke{tcell.ModNone, tcell.KeyF9, 0}
	keyNames["f10"] = KeyStroke{tcell.ModNone, tcell.KeyF10, 0}
	keyNames["f11"] = KeyStroke{tcell.ModNone, tcell.KeyF11, 0}
	keyNames["f12"] = KeyStroke{tcell.ModNone, tcell.KeyF12, 0}
	keyNames["f13"] = KeyStroke{tcell.ModNone, tcell.KeyF13, 0}
	keyNames["f14"] = KeyStroke{tcell.ModNone, tcell.KeyF14, 0}
	keyNames["f15"] = KeyStroke{tcell.ModNone, tcell.KeyF15, 0}
	keyNames["f16"] = KeyStroke{tcell.ModNone, tcell.KeyF16, 0}
	keyNames["f17"] = KeyStroke{tcell.ModNone, tcell.KeyF17, 0}
	keyNames["f18"] = KeyStroke{tcell.ModNone, tcell.KeyF18, 0}
	keyNames["f19"] = KeyStroke{tcell.ModNone, tcell.KeyF19, 0}
	keyNames["f20"] = KeyStroke{tcell.ModNone, tcell.KeyF20, 0}
	keyNames["f21"] = KeyStroke{tcell.ModNone, tcell.KeyF21, 0}
	keyNames["f22"] = KeyStroke{tcell.ModNone, tcell.KeyF22, 0}
	keyNames["f23"] = KeyStroke{tcell.ModNone, tcell.KeyF23, 0}
	keyNames["f24"] = KeyStroke{tcell.ModNone, tcell.KeyF24, 0}
	keyNames["f25"] = KeyStroke{tcell.ModNone, tcell.KeyF25, 0}
	keyNames["f26"] = KeyStroke{tcell.ModNone, tcell.KeyF26, 0}
	keyNames["f27"] = KeyStroke{tcell.ModNone, tcell.KeyF27, 0}
	keyNames["f28"] = KeyStroke{tcell.ModNone, tcell.KeyF28, 0}
	keyNames["f29"] = KeyStroke{tcell.ModNone, tcell.KeyF29, 0}
	keyNames["f30"] = KeyStroke{tcell.ModNone, tcell.KeyF30, 0}
	keyNames["f31"] = KeyStroke{tcell.ModNone, tcell.KeyF31, 0}
	keyNames["f32"] = KeyStroke{tcell.ModNone, tcell.KeyF32, 0}
	keyNames["f33"] = KeyStroke{tcell.ModNone, tcell.KeyF33, 0}
	keyNames["f34"] = KeyStroke{tcell.ModNone, tcell.KeyF34, 0}
	keyNames["f35"] = KeyStroke{tcell.ModNone, tcell.KeyF35, 0}
	keyNames["f36"] = KeyStroke{tcell.ModNone, tcell.KeyF36, 0}
	keyNames["f37"] = KeyStroke{tcell.ModNone, tcell.KeyF37, 0}
	keyNames["f38"] = KeyStroke{tcell.ModNone, tcell.KeyF38, 0}
	keyNames["f39"] = KeyStroke{tcell.ModNone, tcell.KeyF39, 0}
	keyNames["f40"] = KeyStroke{tcell.ModNone, tcell.KeyF40, 0}
	keyNames["f41"] = KeyStroke{tcell.ModNone, tcell.KeyF41, 0}
	keyNames["f42"] = KeyStroke{tcell.ModNone, tcell.KeyF42, 0}
	keyNames["f43"] = KeyStroke{tcell.ModNone, tcell.KeyF43, 0}
	keyNames["f44"] = KeyStroke{tcell.ModNone, tcell.KeyF44, 0}
	keyNames["f45"] = KeyStroke{tcell.ModNone, tcell.KeyF45, 0}
	keyNames["f46"] = KeyStroke{tcell.ModNone, tcell.KeyF46, 0}
	keyNames["f47"] = KeyStroke{tcell.ModNone, tcell.KeyF47, 0}
	keyNames["f48"] = KeyStroke{tcell.ModNone, tcell.KeyF48, 0}
	keyNames["f49"] = KeyStroke{tcell.ModNone, tcell.KeyF49, 0}
	keyNames["f50"] = KeyStroke{tcell.ModNone, tcell.KeyF50, 0}
	keyNames["f51"] = KeyStroke{tcell.ModNone, tcell.KeyF51, 0}
	keyNames["f52"] = KeyStroke{tcell.ModNone, tcell.KeyF52, 0}
	keyNames["f53"] = KeyStroke{tcell.ModNone, tcell.KeyF53, 0}
	keyNames["f54"] = KeyStroke{tcell.ModNone, tcell.KeyF54, 0}
	keyNames["f55"] = KeyStroke{tcell.ModNone, tcell.KeyF55, 0}
	keyNames["f56"] = KeyStroke{tcell.ModNone, tcell.KeyF56, 0}
	keyNames["f57"] = KeyStroke{tcell.ModNone, tcell.KeyF57, 0}
	keyNames["f58"] = KeyStroke{tcell.ModNone, tcell.KeyF58, 0}
	keyNames["f59"] = KeyStroke{tcell.ModNone, tcell.KeyF59, 0}
	keyNames["f60"] = KeyStroke{tcell.ModNone, tcell.KeyF60, 0}
	keyNames["f61"] = KeyStroke{tcell.ModNone, tcell.KeyF61, 0}
	keyNames["f62"] = KeyStroke{tcell.ModNone, tcell.KeyF62, 0}
	keyNames["f63"] = KeyStroke{tcell.ModNone, tcell.KeyF63, 0}
	keyNames["f64"] = KeyStroke{tcell.ModNone, tcell.KeyF64, 0}
	keyNames["c-space"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlSpace, 0}
	keyNames["c-a"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlA, 0}
	keyNames["c-b"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlB, 0}
	keyNames["c-c"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlC, 0}
	keyNames["c-d"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlD, 0}
	keyNames["c-e"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlE, 0}
	keyNames["c-f"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlF, 0}
	keyNames["c-g"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlG, 0}
	keyNames["c-h"] = KeyStroke{tcell.ModNone, tcell.KeyCtrlH, 0}
	keyNames["c-i"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlI, 0}
	keyNames["c-j"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlJ, 0}
	keyNames["c-k"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlK, 0}
	keyNames["c-l"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlL, 0}
	keyNames["c-m"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlM, 0}
	keyNames["c-n"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlN, 0}
	keyNames["c-o"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlO, 0}
	keyNames["c-p"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlP, 0}
	keyNames["c-q"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlQ, 0}
	keyNames["c-r"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlR, 0}
	keyNames["c-s"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlS, 0}
	keyNames["c-t"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlT, 0}
	keyNames["c-u"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlU, 0}
	keyNames["c-v"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlV, 0}
	keyNames["c-w"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlW, 0}
	keyNames["c-x"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlX, rune(tcell.KeyCAN)}
	keyNames["c-y"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlY, 0} // TODO: runes for the rest
	keyNames["c-z"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlZ, 0}
	keyNames["c-]"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlRightSq, 0}
	keyNames["c-\\"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlBackslash, 0}
	keyNames["c-["] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlLeftSq, 0}
	keyNames["c-^"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlCarat, 0}
	keyNames["c-_"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlUnderscore, 0}
	keyNames["nul"] = KeyStroke{tcell.ModNone, tcell.KeyNUL, 0}
	keyNames["soh"] = KeyStroke{tcell.ModNone, tcell.KeySOH, 0}
	keyNames["stx"] = KeyStroke{tcell.ModNone, tcell.KeySTX, 0}
	keyNames["etx"] = KeyStroke{tcell.ModNone, tcell.KeyETX, 0}
	keyNames["eot"] = KeyStroke{tcell.ModNone, tcell.KeyEOT, 0}
	keyNames["enq"] = KeyStroke{tcell.ModNone, tcell.KeyENQ, 0}
	keyNames["ack"] = KeyStroke{tcell.ModNone, tcell.KeyACK, 0}
	keyNames["bel"] = KeyStroke{tcell.ModNone, tcell.KeyBEL, 0}
	keyNames["bs"] = KeyStroke{tcell.ModNone, tcell.KeyBS, 0}
	keyNames["tab"] = KeyStroke{tcell.ModNone, tcell.KeyTAB, 0}
	keyNames["lf"] = KeyStroke{tcell.ModNone, tcell.KeyLF, 0}
	keyNames["vt"] = KeyStroke{tcell.ModNone, tcell.KeyVT, 0}
	keyNames["ff"] = KeyStroke{tcell.ModNone, tcell.KeyFF, 0}
	keyNames["cr"] = KeyStroke{tcell.ModNone, tcell.KeyCR, 0}
	keyNames["so"] = KeyStroke{tcell.ModNone, tcell.KeySO, 0}
	keyNames["si"] = KeyStroke{tcell.ModNone, tcell.KeySI, 0}
	keyNames["dle"] = KeyStroke{tcell.ModNone, tcell.KeyDLE, 0}
	keyNames["dc1"] = KeyStroke{tcell.ModNone, tcell.KeyDC1, 0}
	keyNames["dc2"] = KeyStroke{tcell.ModNone, tcell.KeyDC2, 0}
	keyNames["dc3"] = KeyStroke{tcell.ModNone, tcell.KeyDC3, 0}
	keyNames["dc4"] = KeyStroke{tcell.ModNone, tcell.KeyDC4, 0}
	keyNames["nak"] = KeyStroke{tcell.ModNone, tcell.KeyNAK, 0}
	keyNames["syn"] = KeyStroke{tcell.ModNone, tcell.KeySYN, 0}
	keyNames["etb"] = KeyStroke{tcell.ModNone, tcell.KeyETB, 0}
	keyNames["can"] = KeyStroke{tcell.ModNone, tcell.KeyCAN, 0}
	keyNames["em"] = KeyStroke{tcell.ModNone, tcell.KeyEM, 0}
	keyNames["sub"] = KeyStroke{tcell.ModNone, tcell.KeySUB, 0}
	keyNames["esc"] = KeyStroke{tcell.ModNone, tcell.KeyESC, 0}
	keyNames["fs"] = KeyStroke{tcell.ModNone, tcell.KeyFS, 0}
	keyNames["gs"] = KeyStroke{tcell.ModNone, tcell.KeyGS, 0}
	keyNames["rs"] = KeyStroke{tcell.ModNone, tcell.KeyRS, 0}
	keyNames["us"] = KeyStroke{tcell.ModNone, tcell.KeyUS, 0}
	keyNames["del"] = KeyStroke{tcell.ModNone, tcell.KeyDEL, 0}
}
