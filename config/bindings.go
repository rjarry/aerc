package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gdamore/tcell"
)

type KeyStroke struct {
	Key  tcell.Key
	Rune rune
}

type Binding struct {
	Output []KeyStroke
	Input  []KeyStroke
}

type KeyBindings []*Binding

const (
	BINDING_FOUND = iota
	BINDING_INCOMPLETE
	BINDING_NOT_FOUND
)

type BindingSearchResult int

func NewKeyBindings() *KeyBindings {
	return &KeyBindings{}
}

func (bindings *KeyBindings) Add(binding *Binding) {
	// TODO: Search for conflicts?
	*bindings = append(*bindings, binding)
}

func (bindings *KeyBindings) GetBinding(
	input []KeyStroke) (BindingSearchResult, []KeyStroke) {

	incomplete := false
	// TODO: This could probably be a sorted list to speed things up
	// TODO: Deal with bindings that share a prefix
	for _, binding := range *bindings {
		if len(binding.Input) < len(input) {
			continue
		}
		for i, stroke := range input {
			if stroke != binding.Input[i] {
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
	keyNames map[string]tcell.Key
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
				strokes = append(strokes, KeyStroke{
					Key: key,
				})
			} else {
				return nil, errors.New(fmt.Sprintf("Unknown key '%s'", name))
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
	keyNames = make(map[string]tcell.Key)
	keyNames["up"] = tcell.KeyUp
	keyNames["down"] = tcell.KeyDown
	keyNames["right"] = tcell.KeyRight
	keyNames["left"] = tcell.KeyLeft
	keyNames["upleft"] = tcell.KeyUpLeft
	keyNames["upright"] = tcell.KeyUpRight
	keyNames["downleft"] = tcell.KeyDownLeft
	keyNames["downright"] = tcell.KeyDownRight
	keyNames["center"] = tcell.KeyCenter
	keyNames["pgup"] = tcell.KeyPgUp
	keyNames["pgdn"] = tcell.KeyPgDn
	keyNames["home"] = tcell.KeyHome
	keyNames["end"] = tcell.KeyEnd
	keyNames["insert"] = tcell.KeyInsert
	keyNames["delete"] = tcell.KeyDelete
	keyNames["help"] = tcell.KeyHelp
	keyNames["exit"] = tcell.KeyExit
	keyNames["clear"] = tcell.KeyClear
	keyNames["cancel"] = tcell.KeyCancel
	keyNames["print"] = tcell.KeyPrint
	keyNames["pause"] = tcell.KeyPause
	keyNames["backtab"] = tcell.KeyBacktab
	keyNames["f1"] = tcell.KeyF1
	keyNames["f2"] = tcell.KeyF2
	keyNames["f3"] = tcell.KeyF3
	keyNames["f4"] = tcell.KeyF4
	keyNames["f5"] = tcell.KeyF5
	keyNames["f6"] = tcell.KeyF6
	keyNames["f7"] = tcell.KeyF7
	keyNames["f8"] = tcell.KeyF8
	keyNames["f9"] = tcell.KeyF9
	keyNames["f10"] = tcell.KeyF10
	keyNames["f11"] = tcell.KeyF11
	keyNames["f12"] = tcell.KeyF12
	keyNames["f13"] = tcell.KeyF13
	keyNames["f14"] = tcell.KeyF14
	keyNames["f15"] = tcell.KeyF15
	keyNames["f16"] = tcell.KeyF16
	keyNames["f17"] = tcell.KeyF17
	keyNames["f18"] = tcell.KeyF18
	keyNames["f19"] = tcell.KeyF19
	keyNames["f20"] = tcell.KeyF20
	keyNames["f21"] = tcell.KeyF21
	keyNames["f22"] = tcell.KeyF22
	keyNames["f23"] = tcell.KeyF23
	keyNames["f24"] = tcell.KeyF24
	keyNames["f25"] = tcell.KeyF25
	keyNames["f26"] = tcell.KeyF26
	keyNames["f27"] = tcell.KeyF27
	keyNames["f28"] = tcell.KeyF28
	keyNames["f29"] = tcell.KeyF29
	keyNames["f30"] = tcell.KeyF30
	keyNames["f31"] = tcell.KeyF31
	keyNames["f32"] = tcell.KeyF32
	keyNames["f33"] = tcell.KeyF33
	keyNames["f34"] = tcell.KeyF34
	keyNames["f35"] = tcell.KeyF35
	keyNames["f36"] = tcell.KeyF36
	keyNames["f37"] = tcell.KeyF37
	keyNames["f38"] = tcell.KeyF38
	keyNames["f39"] = tcell.KeyF39
	keyNames["f40"] = tcell.KeyF40
	keyNames["f41"] = tcell.KeyF41
	keyNames["f42"] = tcell.KeyF42
	keyNames["f43"] = tcell.KeyF43
	keyNames["f44"] = tcell.KeyF44
	keyNames["f45"] = tcell.KeyF45
	keyNames["f46"] = tcell.KeyF46
	keyNames["f47"] = tcell.KeyF47
	keyNames["f48"] = tcell.KeyF48
	keyNames["f49"] = tcell.KeyF49
	keyNames["f50"] = tcell.KeyF50
	keyNames["f51"] = tcell.KeyF51
	keyNames["f52"] = tcell.KeyF52
	keyNames["f53"] = tcell.KeyF53
	keyNames["f54"] = tcell.KeyF54
	keyNames["f55"] = tcell.KeyF55
	keyNames["f56"] = tcell.KeyF56
	keyNames["f57"] = tcell.KeyF57
	keyNames["f58"] = tcell.KeyF58
	keyNames["f59"] = tcell.KeyF59
	keyNames["f60"] = tcell.KeyF60
	keyNames["f61"] = tcell.KeyF61
	keyNames["f62"] = tcell.KeyF62
	keyNames["f63"] = tcell.KeyF63
	keyNames["f64"] = tcell.KeyF64
	keyNames["c-space"] = tcell.KeyCtrlSpace
	keyNames["c-a"] = tcell.KeyCtrlA
	keyNames["c-b"] = tcell.KeyCtrlB
	keyNames["c-c"] = tcell.KeyCtrlC
	keyNames["c-d"] = tcell.KeyCtrlD
	keyNames["c-e"] = tcell.KeyCtrlE
	keyNames["c-f"] = tcell.KeyCtrlF
	keyNames["c-g"] = tcell.KeyCtrlG
	keyNames["c-h"] = tcell.KeyCtrlH
	keyNames["c-i"] = tcell.KeyCtrlI
	keyNames["c-j"] = tcell.KeyCtrlJ
	keyNames["c-k"] = tcell.KeyCtrlK
	keyNames["c-l"] = tcell.KeyCtrlL
	keyNames["c-m"] = tcell.KeyCtrlM
	keyNames["c-n"] = tcell.KeyCtrlN
	keyNames["c-o"] = tcell.KeyCtrlO
	keyNames["c-p"] = tcell.KeyCtrlP
	keyNames["c-q"] = tcell.KeyCtrlQ
	keyNames["c-r"] = tcell.KeyCtrlR
	keyNames["c-s"] = tcell.KeyCtrlS
	keyNames["c-t"] = tcell.KeyCtrlT
	keyNames["c-u"] = tcell.KeyCtrlU
	keyNames["c-v"] = tcell.KeyCtrlV
	keyNames["c-w"] = tcell.KeyCtrlW
	keyNames["c-x"] = tcell.KeyCtrlX
	keyNames["c-y"] = tcell.KeyCtrlY
	keyNames["c-z"] = tcell.KeyCtrlZ
	keyNames["c-]"] = tcell.KeyCtrlLeftSq
	keyNames["c-\\"] = tcell.KeyCtrlBackslash
	keyNames["c-["] = tcell.KeyCtrlRightSq
	keyNames["c-^"] = tcell.KeyCtrlCarat
	keyNames["c-_"] = tcell.KeyCtrlUnderscore
	keyNames["NUL"] = tcell.KeyNUL
	keyNames["SOH"] = tcell.KeySOH
	keyNames["STX"] = tcell.KeySTX
	keyNames["ETX"] = tcell.KeyETX
	keyNames["EOT"] = tcell.KeyEOT
	keyNames["ENQ"] = tcell.KeyENQ
	keyNames["ACK"] = tcell.KeyACK
	keyNames["BEL"] = tcell.KeyBEL
	keyNames["BS"] = tcell.KeyBS
	keyNames["TAB"] = tcell.KeyTAB
	keyNames["LF"] = tcell.KeyLF
	keyNames["VT"] = tcell.KeyVT
	keyNames["FF"] = tcell.KeyFF
	keyNames["CR"] = tcell.KeyCR
	keyNames["SO"] = tcell.KeySO
	keyNames["SI"] = tcell.KeySI
	keyNames["DLE"] = tcell.KeyDLE
	keyNames["DC1"] = tcell.KeyDC1
	keyNames["DC2"] = tcell.KeyDC2
	keyNames["DC3"] = tcell.KeyDC3
	keyNames["DC4"] = tcell.KeyDC4
	keyNames["NAK"] = tcell.KeyNAK
	keyNames["SYN"] = tcell.KeySYN
	keyNames["ETB"] = tcell.KeyETB
	keyNames["CAN"] = tcell.KeyCAN
	keyNames["EM"] = tcell.KeyEM
	keyNames["SUB"] = tcell.KeySUB
	keyNames["ESC"] = tcell.KeyESC
	keyNames["FS"] = tcell.KeyFS
	keyNames["GS"] = tcell.KeyGS
	keyNames["RS"] = tcell.KeyRS
	keyNames["US"] = tcell.KeyUS
	keyNames["DEL"] = tcell.KeyDEL
}
