package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"

	"git.sr.ht/~rjarry/aerc/logging"
	"github.com/gdamore/tcell/v2"
	"github.com/go-ini/ini"
)

type BindingConfig struct {
	Global                 *KeyBindings
	AccountWizard          *KeyBindings
	Compose                *KeyBindings
	ComposeEditor          *KeyBindings
	ComposeReview          *KeyBindings
	MessageList            *KeyBindings
	MessageView            *KeyBindings
	MessageViewPassthrough *KeyBindings
	Terminal               *KeyBindings
}

type BindingConfigContext struct {
	ContextType ContextType
	Regex       *regexp.Regexp
	Bindings    *KeyBindings
	BindContext string
}

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
	Bindings []*Binding

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

func (config *AercConfig) parseBinds(root string) error {
	// These bindings are not configurable
	config.Bindings.AccountWizard.ExKey = KeyStroke{
		Key: tcell.KeyCtrlE,
	}
	quit, _ := ParseBinding("<C-q>", ":quit<Enter>")
	config.Bindings.AccountWizard.Add(quit)

	filename := path.Join(root, "binds.conf")
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		logging.Debugf("%s not found, installing the system default", filename)
		if err := installTemplate(root, "binds.conf"); err != nil {
			return err
		}
	}
	logging.Infof("Parsing key bindings configuration from %s", filename)
	binds, err := ini.Load(filename)
	if err != nil {
		return err
	}

	baseGroups := map[string]**KeyBindings{
		"default":           &config.Bindings.Global,
		"compose":           &config.Bindings.Compose,
		"messages":          &config.Bindings.MessageList,
		"terminal":          &config.Bindings.Terminal,
		"view":              &config.Bindings.MessageView,
		"view::passthrough": &config.Bindings.MessageViewPassthrough,
		"compose::editor":   &config.Bindings.ComposeEditor,
		"compose::review":   &config.Bindings.ComposeReview,
	}

	// Base Bindings
	for _, sectionName := range binds.SectionStrings() {
		// Handle :: delimeter
		baseSectionName := strings.ReplaceAll(sectionName, "::", "////")
		sections := strings.Split(baseSectionName, ":")
		baseOnly := len(sections) == 1
		baseSectionName = strings.ReplaceAll(sections[0], "////", "::")

		group, ok := baseGroups[strings.ToLower(baseSectionName)]
		if !ok {
			return errors.New("Unknown keybinding group " + sectionName)
		}

		if baseOnly {
			err = config.LoadBinds(binds, baseSectionName, group)
			if err != nil {
				return err
			}
		}
	}

	config.Bindings.Global.Globals = false
	for _, contextBind := range config.ContextualBinds {
		if contextBind.BindContext == "default" {
			contextBind.Bindings.Globals = false
		}
	}

	logging.Debugf("binds.conf: %#v", config.Bindings)
	return nil
}

func LoadBindingSection(sec *ini.Section) (*KeyBindings, error) {
	bindings := NewKeyBindings()
	for key, value := range sec.KeysHash() {
		if key == "$ex" {
			strokes, err := ParseKeyStrokes(value)
			if err != nil {
				return nil, err
			}
			if len(strokes) != 1 {
				return nil, errors.New("Invalid binding")
			}
			bindings.ExKey = strokes[0]
			continue
		}
		if key == "$noinherit" {
			if value == "false" {
				continue
			}
			if value != "true" {
				return nil, errors.New("Invalid binding")
			}
			bindings.Globals = false
			continue
		}
		binding, err := ParseBinding(key, value)
		if err != nil {
			return nil, err
		}
		bindings.Add(binding)
	}
	return bindings, nil
}

func (config *AercConfig) LoadBinds(binds *ini.File, baseName string, baseGroup **KeyBindings) error {
	if sec, err := binds.GetSection(baseName); err == nil {
		binds, err := LoadBindingSection(sec)
		if err != nil {
			return err
		}
		*baseGroup = MergeBindings(binds, *baseGroup)
	}

	for _, sectionName := range binds.SectionStrings() {
		if !strings.Contains(sectionName, baseName+":") ||
			strings.Contains(sectionName, baseName+"::") {
			continue
		}

		bindSection, err := binds.GetSection(sectionName)
		if err != nil {
			return err
		}

		binds, err := LoadBindingSection(bindSection)
		if err != nil {
			return err
		}

		contextualBind := BindingConfigContext{
			Bindings:    binds,
			BindContext: baseName,
		}

		var index int
		if strings.Contains(sectionName, "=") {
			index = strings.Index(sectionName, "=")
			value := string(sectionName[index+1:])
			contextualBind.Regex, err = regexp.Compile(value)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Invalid Bind Context regex in %s", sectionName)
		}

		switch sectionName[len(baseName)+1 : index] {
		case "account":
			acctName := sectionName[index+1:]
			valid := false
			for _, acctConf := range config.Accounts {
				matches := contextualBind.Regex.FindString(acctConf.Name)
				if matches != "" {
					valid = true
				}
			}
			if !valid {
				logging.Warnf("binds.conf: unexistent account: %s", acctName)
				continue
			}
			contextualBind.ContextType = BIND_CONTEXT_ACCOUNT
		case "folder":
			// No validation needed. If the folder doesn't exist, the binds
			// never get used
			contextualBind.ContextType = BIND_CONTEXT_FOLDER
		default:
			return fmt.Errorf("Unknown Context Bind Section: %s", sectionName)
		}
		config.ContextualBinds = append(config.ContextualBinds, contextualBind)
	}

	return nil
}

func defaultBindsConfig() BindingConfig {
	return BindingConfig{
		Global:                 NewKeyBindings(),
		AccountWizard:          NewKeyBindings(),
		Compose:                NewKeyBindings(),
		ComposeEditor:          NewKeyBindings(),
		ComposeReview:          NewKeyBindings(),
		MessageList:            NewKeyBindings(),
		MessageView:            NewKeyBindings(),
		MessageViewPassthrough: NewKeyBindings(),
		Terminal:               NewKeyBindings(),
	}
}

func NewKeyBindings() *KeyBindings {
	return &KeyBindings{
		ExKey:   KeyStroke{tcell.ModNone, tcell.KeyRune, ':'},
		Globals: true,
	}
}

func MergeBindings(bindings ...*KeyBindings) *KeyBindings {
	merged := NewKeyBindings()
	for _, b := range bindings {
		merged.Bindings = append(merged.Bindings, b.Bindings...)
	}
	merged.ExKey = bindings[0].ExKey
	merged.Globals = bindings[0].Globals
	return merged
}

func (config AercConfig) MergeContextualBinds(baseBinds *KeyBindings,
	contextType ContextType, reg string, bindCtx string,
) *KeyBindings {
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
	bindings.Bindings = append(bindings.Bindings, binding)
}

func (bindings *KeyBindings) GetBinding(
	input []KeyStroke,
) (BindingSearchResult, []KeyStroke) {
	incomplete := false
	// TODO: This could probably be a sorted list to speed things up
	// TODO: Deal with bindings that share a prefix
	for _, binding := range bindings.Bindings {
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

	for _, binding := range bindings.Bindings {
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
				switch name {
				case "cr", "c-m":
					name = "enter"
				case "c-i":
					name = "tab"
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

var keyNames map[string]KeyStroke

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
			switch {
			case err == io.EOF:
				return nil, errors.New("Expecting '>'")
			case err != nil:
				return nil, err
			case name == ">":
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
	keyNames["a-enter"] = KeyStroke{tcell.ModAlt, tcell.KeyEnter, 0}
	keyNames["up"] = KeyStroke{tcell.ModNone, tcell.KeyUp, 0}
	keyNames["c-up"] = KeyStroke{tcell.ModCtrl, tcell.KeyUp, 0}
	keyNames["a-up"] = KeyStroke{tcell.ModAlt, tcell.KeyUp, 0}
	keyNames["down"] = KeyStroke{tcell.ModNone, tcell.KeyDown, 0}
	keyNames["c-down"] = KeyStroke{tcell.ModCtrl, tcell.KeyDown, 0}
	keyNames["a-down"] = KeyStroke{tcell.ModAlt, tcell.KeyDown, 0}
	keyNames["right"] = KeyStroke{tcell.ModNone, tcell.KeyRight, 0}
	keyNames["c-right"] = KeyStroke{tcell.ModCtrl, tcell.KeyRight, 0}
	keyNames["a-right"] = KeyStroke{tcell.ModAlt, tcell.KeyRight, 0}
	keyNames["left"] = KeyStroke{tcell.ModNone, tcell.KeyLeft, 0}
	keyNames["c-left"] = KeyStroke{tcell.ModCtrl, tcell.KeyLeft, 0}
	keyNames["a-left"] = KeyStroke{tcell.ModAlt, tcell.KeyLeft, 0}
	keyNames["upleft"] = KeyStroke{tcell.ModNone, tcell.KeyUpLeft, 0}
	keyNames["upright"] = KeyStroke{tcell.ModNone, tcell.KeyUpRight, 0}
	keyNames["downleft"] = KeyStroke{tcell.ModNone, tcell.KeyDownLeft, 0}
	keyNames["downright"] = KeyStroke{tcell.ModNone, tcell.KeyDownRight, 0}
	keyNames["center"] = KeyStroke{tcell.ModNone, tcell.KeyCenter, 0}
	keyNames["pgup"] = KeyStroke{tcell.ModNone, tcell.KeyPgUp, 0}
	keyNames["c-pgup"] = KeyStroke{tcell.ModCtrl, tcell.KeyPgUp, 0}
	keyNames["a-pgup"] = KeyStroke{tcell.ModAlt, tcell.KeyPgUp, 0}
	keyNames["pgdn"] = KeyStroke{tcell.ModNone, tcell.KeyPgDn, 0}
	keyNames["c-pgdn"] = KeyStroke{tcell.ModCtrl, tcell.KeyPgDn, 0}
	keyNames["a-pgdn"] = KeyStroke{tcell.ModAlt, tcell.KeyPgDn, 0}
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
	keyNames["c-i"] = KeyStroke{tcell.ModNone, tcell.KeyCtrlI, 0}
	keyNames["c-j"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlJ, 0}
	keyNames["c-k"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlK, 0}
	keyNames["c-l"] = KeyStroke{tcell.ModCtrl, tcell.KeyCtrlL, 0}
	keyNames["c-m"] = KeyStroke{tcell.ModNone, tcell.KeyCtrlM, 0}
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
	keyNames["a-space"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, ' '}
	keyNames["a-a"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'a'}
	keyNames["a-b"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'b'}
	keyNames["a-c"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'c'}
	keyNames["a-d"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'd'}
	keyNames["a-e"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'e'}
	keyNames["a-f"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'f'}
	keyNames["a-g"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'g'}
	keyNames["a-h"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'h'}
	keyNames["a-i"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'i'}
	keyNames["a-j"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'j'}
	keyNames["a-k"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'k'}
	keyNames["a-l"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'l'}
	keyNames["a-m"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'm'}
	keyNames["a-n"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'n'}
	keyNames["a-o"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'o'}
	keyNames["a-p"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'p'}
	keyNames["a-q"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'q'}
	keyNames["a-r"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'r'}
	keyNames["a-s"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 's'}
	keyNames["a-t"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 't'}
	keyNames["a-u"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'u'}
	keyNames["a-v"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'v'}
	keyNames["a-w"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'w'}
	keyNames["a-x"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'x'}
	keyNames["a-y"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'y'}
	keyNames["a-z"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, 'z'}
	keyNames["a-]"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, ']'}
	keyNames["a-\\"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, '\\'}
	keyNames["a-["] = KeyStroke{tcell.ModAlt, tcell.KeyRune, '['}
	keyNames["a-^"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, '^'}
	keyNames["a-_"] = KeyStroke{tcell.ModAlt, tcell.KeyRune, '_'}
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
