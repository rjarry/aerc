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

	"git.sr.ht/~rjarry/aerc/log"
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

type bindsContextType int

const (
	bindsContextFolder bindsContextType = iota
	bindsContextAccount
)

type BindingConfigContext struct {
	ContextType bindsContextType
	Regex       *regexp.Regexp
	Bindings    *KeyBindings
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
	// Which key triggers completion (default is <tab>)
	CompleteKey KeyStroke

	// private
	contextualBinds  []*BindingConfigContext
	contextualCounts map[bindsContextType]int
	contextualCache  map[bindsContextKey]*KeyBindings
}

type bindsContextKey struct {
	ctxType bindsContextType
	value   string
}

const (
	BINDING_FOUND = iota
	BINDING_INCOMPLETE
	BINDING_NOT_FOUND
)

type BindingSearchResult int

func defaultBindsConfig() *BindingConfig {
	// These bindings are not configurable
	wizard := NewKeyBindings()
	wizard.ExKey = KeyStroke{Key: tcell.KeyCtrlE}
	wizard.Globals = false
	quit, _ := ParseBinding("<C-q>", ":quit<Enter>")
	wizard.Add(quit)
	return &BindingConfig{
		Global:                 NewKeyBindings(),
		AccountWizard:          wizard,
		Compose:                NewKeyBindings(),
		ComposeEditor:          NewKeyBindings(),
		ComposeReview:          NewKeyBindings(),
		MessageList:            NewKeyBindings(),
		MessageView:            NewKeyBindings(),
		MessageViewPassthrough: NewKeyBindings(),
		Terminal:               NewKeyBindings(),
	}
}

var Binds = defaultBindsConfig()

func parseBinds(root string, filename string) error {
	if filename == "" {
		filename = path.Join(root, "binds.conf")
		if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
			fmt.Printf("%s not found, installing the system default\n", filename)
			if err := installTemplate(root, "binds.conf"); err != nil {
				return err
			}
		}
	}
	log.Debugf("Parsing key bindings configuration from %s", filename)
	binds, err := ini.LoadSources(ini.LoadOptions{
		KeyValueDelimiters: "=",
	}, filename)
	if err != nil {
		return err
	}

	baseGroups := map[string]**KeyBindings{
		"default":           &Binds.Global,
		"compose":           &Binds.Compose,
		"messages":          &Binds.MessageList,
		"terminal":          &Binds.Terminal,
		"view":              &Binds.MessageView,
		"view::passthrough": &Binds.MessageViewPassthrough,
		"compose::editor":   &Binds.ComposeEditor,
		"compose::review":   &Binds.ComposeReview,
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
			err = LoadBinds(binds, baseSectionName, group)
			if err != nil {
				return err
			}
		}
	}

	log.Debugf("binds.conf: %#v", Binds)
	return nil
}

func LoadBindingSection(sec *ini.Section) (*KeyBindings, error) {
	bindings := NewKeyBindings()
	for key, value := range sec.KeysHash() {
		switch key {
		case "$ex":
			strokes, err := ParseKeyStrokes(value)
			if err != nil {
				return nil, err
			}
			if len(strokes) != 1 {
				return nil, errors.New("Invalid binding")
			}
			bindings.ExKey = strokes[0]
		case "$noinherit":
			if value == "false" {
				continue
			}
			if value != "true" {
				return nil, errors.New("Invalid binding")
			}
			bindings.Globals = false
		case "$complete":
			strokes, err := ParseKeyStrokes(value)
			if err != nil {
				return nil, err
			}
			if len(strokes) != 1 {
				return nil, errors.New("Invalid binding")
			}
			bindings.CompleteKey = strokes[0]
		default:
			binding, err := ParseBinding(key, value)
			if err != nil {
				return nil, err
			}
			bindings.Add(binding)
		}
	}
	return bindings, nil
}

func LoadBinds(binds *ini.File, baseName string, baseGroup **KeyBindings) error {
	if sec, err := binds.GetSection(baseName); err == nil {
		binds, err := LoadBindingSection(sec)
		if err != nil {
			return err
		}
		*baseGroup = MergeBindings(binds, *baseGroup)
	}

	b := *baseGroup

	if baseName == "default" {
		b.Globals = false
	}

	for _, sectionName := range binds.SectionStrings() {
		if !strings.HasPrefix(sectionName, baseName+":") ||
			strings.HasPrefix(sectionName, baseName+"::") {
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
		if baseName == "default" {
			binds.Globals = false
		}

		contextualBind := BindingConfigContext{
			Bindings: binds,
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
			for _, acctConf := range Accounts {
				matches := contextualBind.Regex.FindString(acctConf.Name)
				if matches != "" {
					valid = true
				}
			}
			if !valid {
				log.Warnf("binds.conf: unexistent account: %s", acctName)
				continue
			}
			contextualBind.ContextType = bindsContextAccount
		case "folder":
			// No validation needed. If the folder doesn't exist, the binds
			// never get used
			contextualBind.ContextType = bindsContextFolder
		default:
			return fmt.Errorf("Unknown Context Bind Section: %s", sectionName)
		}
		b.contextualBinds = append(b.contextualBinds, &contextualBind)
		b.contextualCounts[contextualBind.ContextType]++
	}

	return nil
}

func NewKeyBindings() *KeyBindings {
	return &KeyBindings{
		ExKey:            KeyStroke{tcell.ModNone, tcell.KeyRune, ':'},
		CompleteKey:      KeyStroke{tcell.ModNone, tcell.KeyTab, 0},
		Globals:          true,
		contextualCache:  make(map[bindsContextKey]*KeyBindings),
		contextualCounts: make(map[bindsContextType]int),
	}
}

func areBindingsInputsEqual(a, b *Binding) bool {
	if len(a.Input) != len(b.Input) {
		return false
	}

	for idx := range a.Input {
		if a.Input[idx] != b.Input[idx] {
			return false
		}
	}

	return true
}

// this scans the bindings slice for copies and leaves just the first ones
// it also removes empty bindings, the ones that do nothing, so you can
// override and erase parent bindings with the context ones
func filterAndCleanBindings(bindings []*Binding) []*Binding {
	// 1. remove a binding if we already have one with the same input
	res1 := []*Binding{}
	for _, b := range bindings {
		// do we already have one here?
		found := false
		for _, r := range res1 {
			if areBindingsInputsEqual(b, r) {
				found = true
				break
			}
		}

		// add it if we don't
		if !found {
			res1 = append(res1, b)
		}
	}

	// 2. clean up the empty bindings
	res2 := []*Binding{}
	for _, b := range res1 {
		if len(b.Output) > 0 {
			res2 = append(res2, b)
		}
	}

	return res2
}

func MergeBindings(bindings ...*KeyBindings) *KeyBindings {
	merged := NewKeyBindings()
	for _, b := range bindings {
		merged.Bindings = append(merged.Bindings, b.Bindings...)
		if !b.Globals {
			break
		}
	}
	merged.Bindings = filterAndCleanBindings(merged.Bindings)
	merged.ExKey = bindings[0].ExKey
	merged.CompleteKey = bindings[0].CompleteKey
	merged.Globals = bindings[0].Globals
	return merged
}

func (base *KeyBindings) contextual(
	contextType bindsContextType, reg string,
) *KeyBindings {
	if base.contextualCounts[contextType] == 0 {
		// shortcut if no contextual binds for that type
		return base
	}

	key := bindsContextKey{ctxType: contextType, value: reg}
	c, found := base.contextualCache[key]
	if found {
		return c
	}

	c = base
	for _, contextualBind := range base.contextualBinds {
		if contextualBind.ContextType != contextType {
			continue
		}
		if !contextualBind.Regex.Match([]byte(reg)) {
			continue
		}
		c = MergeBindings(contextualBind.Bindings, c)
	}
	base.contextualCache[key] = c

	return c
}

func (bindings *KeyBindings) ForAccount(account string) *KeyBindings {
	return bindings.contextual(bindsContextAccount, account)
}

func (bindings *KeyBindings) ForFolder(folder string) *KeyBindings {
	return bindings.contextual(bindsContextFolder, folder)
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
					s = "<enter>"
				case "c-i":
					s = "<tab>"
				case "space":
					s = " "
				case "semicolon":
					s = ";"
				default:
					s = fmt.Sprintf("<%s>", name)
				}
				break
			}
		}
		if s == "" && stroke.Key == tcell.KeyRune {
			s = string(stroke.Rune)
		}
		sb.WriteString(s)
	}

	// replace leading & trailing spaces with explicit <space> keystrokes
	buf := sb.String()
	match := spaceTrimRe.FindStringSubmatch(buf)
	if len(match) == 4 {
		prefix := strings.ReplaceAll(match[1], " ", "<space>")
		suffix := strings.ReplaceAll(match[3], " ", "<space>")
		buf = prefix + match[2] + suffix
	}

	return buf
}

var spaceTrimRe = regexp.MustCompile(`^(\s*)(.*?)(\s*)$`)

var keyNames = map[string]KeyStroke{
	"space":     {tcell.ModNone, tcell.KeyRune, ' '},
	"semicolon": {tcell.ModNone, tcell.KeyRune, ';'},
	"enter":     {tcell.ModNone, tcell.KeyEnter, 0},
	"c-enter":   {tcell.ModCtrl, tcell.KeyEnter, 0},
	"a-enter":   {tcell.ModAlt, tcell.KeyEnter, 0},
	"up":        {tcell.ModNone, tcell.KeyUp, 0},
	"c-up":      {tcell.ModCtrl, tcell.KeyUp, 0},
	"a-up":      {tcell.ModAlt, tcell.KeyUp, 0},
	"down":      {tcell.ModNone, tcell.KeyDown, 0},
	"c-down":    {tcell.ModCtrl, tcell.KeyDown, 0},
	"a-down":    {tcell.ModAlt, tcell.KeyDown, 0},
	"right":     {tcell.ModNone, tcell.KeyRight, 0},
	"c-right":   {tcell.ModCtrl, tcell.KeyRight, 0},
	"a-right":   {tcell.ModAlt, tcell.KeyRight, 0},
	"left":      {tcell.ModNone, tcell.KeyLeft, 0},
	"c-left":    {tcell.ModCtrl, tcell.KeyLeft, 0},
	"a-left":    {tcell.ModAlt, tcell.KeyLeft, 0},
	"upleft":    {tcell.ModNone, tcell.KeyUpLeft, 0},
	"upright":   {tcell.ModNone, tcell.KeyUpRight, 0},
	"downleft":  {tcell.ModNone, tcell.KeyDownLeft, 0},
	"downright": {tcell.ModNone, tcell.KeyDownRight, 0},
	"center":    {tcell.ModNone, tcell.KeyCenter, 0},
	"pgup":      {tcell.ModNone, tcell.KeyPgUp, 0},
	"c-pgup":    {tcell.ModCtrl, tcell.KeyPgUp, 0},
	"a-pgup":    {tcell.ModAlt, tcell.KeyPgUp, 0},
	"pgdn":      {tcell.ModNone, tcell.KeyPgDn, 0},
	"c-pgdn":    {tcell.ModCtrl, tcell.KeyPgDn, 0},
	"a-pgdn":    {tcell.ModAlt, tcell.KeyPgDn, 0},
	"home":      {tcell.ModNone, tcell.KeyHome, 0},
	"end":       {tcell.ModNone, tcell.KeyEnd, 0},
	"insert":    {tcell.ModNone, tcell.KeyInsert, 0},
	"delete":    {tcell.ModNone, tcell.KeyDelete, 0},
	"c-delete":  {tcell.ModCtrl, tcell.KeyDelete, 0},
	"a-delete":  {tcell.ModAlt, tcell.KeyDelete, 0},
	"backspace": {tcell.ModNone, tcell.KeyBackspace2, 0},
	"help":      {tcell.ModNone, tcell.KeyHelp, 0},
	"exit":      {tcell.ModNone, tcell.KeyExit, 0},
	"clear":     {tcell.ModNone, tcell.KeyClear, 0},
	"cancel":    {tcell.ModNone, tcell.KeyCancel, 0},
	"print":     {tcell.ModNone, tcell.KeyPrint, 0},
	"pause":     {tcell.ModNone, tcell.KeyPause, 0},
	"backtab":   {tcell.ModNone, tcell.KeyBacktab, 0},
	"f1":        {tcell.ModNone, tcell.KeyF1, 0},
	"f2":        {tcell.ModNone, tcell.KeyF2, 0},
	"f3":        {tcell.ModNone, tcell.KeyF3, 0},
	"f4":        {tcell.ModNone, tcell.KeyF4, 0},
	"f5":        {tcell.ModNone, tcell.KeyF5, 0},
	"f6":        {tcell.ModNone, tcell.KeyF6, 0},
	"f7":        {tcell.ModNone, tcell.KeyF7, 0},
	"f8":        {tcell.ModNone, tcell.KeyF8, 0},
	"f9":        {tcell.ModNone, tcell.KeyF9, 0},
	"f10":       {tcell.ModNone, tcell.KeyF10, 0},
	"f11":       {tcell.ModNone, tcell.KeyF11, 0},
	"f12":       {tcell.ModNone, tcell.KeyF12, 0},
	"f13":       {tcell.ModNone, tcell.KeyF13, 0},
	"f14":       {tcell.ModNone, tcell.KeyF14, 0},
	"f15":       {tcell.ModNone, tcell.KeyF15, 0},
	"f16":       {tcell.ModNone, tcell.KeyF16, 0},
	"f17":       {tcell.ModNone, tcell.KeyF17, 0},
	"f18":       {tcell.ModNone, tcell.KeyF18, 0},
	"f19":       {tcell.ModNone, tcell.KeyF19, 0},
	"f20":       {tcell.ModNone, tcell.KeyF20, 0},
	"f21":       {tcell.ModNone, tcell.KeyF21, 0},
	"f22":       {tcell.ModNone, tcell.KeyF22, 0},
	"f23":       {tcell.ModNone, tcell.KeyF23, 0},
	"f24":       {tcell.ModNone, tcell.KeyF24, 0},
	"f25":       {tcell.ModNone, tcell.KeyF25, 0},
	"f26":       {tcell.ModNone, tcell.KeyF26, 0},
	"f27":       {tcell.ModNone, tcell.KeyF27, 0},
	"f28":       {tcell.ModNone, tcell.KeyF28, 0},
	"f29":       {tcell.ModNone, tcell.KeyF29, 0},
	"f30":       {tcell.ModNone, tcell.KeyF30, 0},
	"f31":       {tcell.ModNone, tcell.KeyF31, 0},
	"f32":       {tcell.ModNone, tcell.KeyF32, 0},
	"f33":       {tcell.ModNone, tcell.KeyF33, 0},
	"f34":       {tcell.ModNone, tcell.KeyF34, 0},
	"f35":       {tcell.ModNone, tcell.KeyF35, 0},
	"f36":       {tcell.ModNone, tcell.KeyF36, 0},
	"f37":       {tcell.ModNone, tcell.KeyF37, 0},
	"f38":       {tcell.ModNone, tcell.KeyF38, 0},
	"f39":       {tcell.ModNone, tcell.KeyF39, 0},
	"f40":       {tcell.ModNone, tcell.KeyF40, 0},
	"f41":       {tcell.ModNone, tcell.KeyF41, 0},
	"f42":       {tcell.ModNone, tcell.KeyF42, 0},
	"f43":       {tcell.ModNone, tcell.KeyF43, 0},
	"f44":       {tcell.ModNone, tcell.KeyF44, 0},
	"f45":       {tcell.ModNone, tcell.KeyF45, 0},
	"f46":       {tcell.ModNone, tcell.KeyF46, 0},
	"f47":       {tcell.ModNone, tcell.KeyF47, 0},
	"f48":       {tcell.ModNone, tcell.KeyF48, 0},
	"f49":       {tcell.ModNone, tcell.KeyF49, 0},
	"f50":       {tcell.ModNone, tcell.KeyF50, 0},
	"f51":       {tcell.ModNone, tcell.KeyF51, 0},
	"f52":       {tcell.ModNone, tcell.KeyF52, 0},
	"f53":       {tcell.ModNone, tcell.KeyF53, 0},
	"f54":       {tcell.ModNone, tcell.KeyF54, 0},
	"f55":       {tcell.ModNone, tcell.KeyF55, 0},
	"f56":       {tcell.ModNone, tcell.KeyF56, 0},
	"f57":       {tcell.ModNone, tcell.KeyF57, 0},
	"f58":       {tcell.ModNone, tcell.KeyF58, 0},
	"f59":       {tcell.ModNone, tcell.KeyF59, 0},
	"f60":       {tcell.ModNone, tcell.KeyF60, 0},
	"f61":       {tcell.ModNone, tcell.KeyF61, 0},
	"f62":       {tcell.ModNone, tcell.KeyF62, 0},
	"f63":       {tcell.ModNone, tcell.KeyF63, 0},
	"f64":       {tcell.ModNone, tcell.KeyF64, 0},
	"c-space":   {tcell.ModCtrl, tcell.KeyCtrlSpace, 0},
	"c-a":       {tcell.ModCtrl, tcell.KeyCtrlA, 0},
	"c-b":       {tcell.ModCtrl, tcell.KeyCtrlB, 0},
	"c-c":       {tcell.ModCtrl, tcell.KeyCtrlC, 0},
	"c-d":       {tcell.ModCtrl, tcell.KeyCtrlD, 0},
	"c-e":       {tcell.ModCtrl, tcell.KeyCtrlE, 0},
	"c-f":       {tcell.ModCtrl, tcell.KeyCtrlF, 0},
	"c-g":       {tcell.ModCtrl, tcell.KeyCtrlG, 0},
	"c-h":       {tcell.ModNone, tcell.KeyCtrlH, 0},
	"c-i":       {tcell.ModNone, tcell.KeyCtrlI, 0},
	"c-j":       {tcell.ModCtrl, tcell.KeyCtrlJ, 0},
	"c-k":       {tcell.ModCtrl, tcell.KeyCtrlK, 0},
	"c-l":       {tcell.ModCtrl, tcell.KeyCtrlL, 0},
	"c-m":       {tcell.ModNone, tcell.KeyCtrlM, 0},
	"c-n":       {tcell.ModCtrl, tcell.KeyCtrlN, 0},
	"c-o":       {tcell.ModCtrl, tcell.KeyCtrlO, 0},
	"c-p":       {tcell.ModCtrl, tcell.KeyCtrlP, 0},
	"c-q":       {tcell.ModCtrl, tcell.KeyCtrlQ, 0},
	"c-r":       {tcell.ModCtrl, tcell.KeyCtrlR, 0},
	"c-s":       {tcell.ModCtrl, tcell.KeyCtrlS, 0},
	"c-t":       {tcell.ModCtrl, tcell.KeyCtrlT, 0},
	"c-u":       {tcell.ModCtrl, tcell.KeyCtrlU, 0},
	"c-v":       {tcell.ModCtrl, tcell.KeyCtrlV, 0},
	"c-w":       {tcell.ModCtrl, tcell.KeyCtrlW, 0},
	"c-x":       {tcell.ModCtrl, tcell.KeyCtrlX, rune(tcell.KeyCAN)},
	"c-y":       {tcell.ModCtrl, tcell.KeyCtrlY, 0}, // TODO: runes for the rest
	"c-z":       {tcell.ModCtrl, tcell.KeyCtrlZ, 0},
	"c-]":       {tcell.ModCtrl, tcell.KeyCtrlRightSq, 0},
	"c-\\":      {tcell.ModCtrl, tcell.KeyCtrlBackslash, 0},
	"c-[":       {tcell.ModCtrl, tcell.KeyCtrlLeftSq, 0},
	"c-^":       {tcell.ModCtrl, tcell.KeyCtrlCarat, 0},
	"c-_":       {tcell.ModCtrl, tcell.KeyCtrlUnderscore, 0},
	"a-space":   {tcell.ModAlt, tcell.KeyRune, ' '},
	"a-0":       {tcell.ModAlt, tcell.KeyRune, '0'},
	"a-1":       {tcell.ModAlt, tcell.KeyRune, '1'},
	"a-2":       {tcell.ModAlt, tcell.KeyRune, '2'},
	"a-3":       {tcell.ModAlt, tcell.KeyRune, '3'},
	"a-4":       {tcell.ModAlt, tcell.KeyRune, '4'},
	"a-5":       {tcell.ModAlt, tcell.KeyRune, '5'},
	"a-6":       {tcell.ModAlt, tcell.KeyRune, '6'},
	"a-7":       {tcell.ModAlt, tcell.KeyRune, '7'},
	"a-8":       {tcell.ModAlt, tcell.KeyRune, '8'},
	"a-9":       {tcell.ModAlt, tcell.KeyRune, '9'},
	"a-a":       {tcell.ModAlt, tcell.KeyRune, 'a'},
	"a-b":       {tcell.ModAlt, tcell.KeyRune, 'b'},
	"a-c":       {tcell.ModAlt, tcell.KeyRune, 'c'},
	"a-d":       {tcell.ModAlt, tcell.KeyRune, 'd'},
	"a-e":       {tcell.ModAlt, tcell.KeyRune, 'e'},
	"a-f":       {tcell.ModAlt, tcell.KeyRune, 'f'},
	"a-g":       {tcell.ModAlt, tcell.KeyRune, 'g'},
	"a-h":       {tcell.ModAlt, tcell.KeyRune, 'h'},
	"a-i":       {tcell.ModAlt, tcell.KeyRune, 'i'},
	"a-j":       {tcell.ModAlt, tcell.KeyRune, 'j'},
	"a-k":       {tcell.ModAlt, tcell.KeyRune, 'k'},
	"a-l":       {tcell.ModAlt, tcell.KeyRune, 'l'},
	"a-m":       {tcell.ModAlt, tcell.KeyRune, 'm'},
	"a-n":       {tcell.ModAlt, tcell.KeyRune, 'n'},
	"a-o":       {tcell.ModAlt, tcell.KeyRune, 'o'},
	"a-p":       {tcell.ModAlt, tcell.KeyRune, 'p'},
	"a-q":       {tcell.ModAlt, tcell.KeyRune, 'q'},
	"a-r":       {tcell.ModAlt, tcell.KeyRune, 'r'},
	"a-s":       {tcell.ModAlt, tcell.KeyRune, 's'},
	"a-t":       {tcell.ModAlt, tcell.KeyRune, 't'},
	"a-u":       {tcell.ModAlt, tcell.KeyRune, 'u'},
	"a-v":       {tcell.ModAlt, tcell.KeyRune, 'v'},
	"a-w":       {tcell.ModAlt, tcell.KeyRune, 'w'},
	"a-x":       {tcell.ModAlt, tcell.KeyRune, 'x'},
	"a-y":       {tcell.ModAlt, tcell.KeyRune, 'y'},
	"a-z":       {tcell.ModAlt, tcell.KeyRune, 'z'},
	"a-]":       {tcell.ModAlt, tcell.KeyRune, ']'},
	"a-\\":      {tcell.ModAlt, tcell.KeyRune, '\\'},
	"a-[":       {tcell.ModAlt, tcell.KeyRune, '['},
	"a-^":       {tcell.ModAlt, tcell.KeyRune, '^'},
	"a-_":       {tcell.ModAlt, tcell.KeyRune, '_'},
	"nul":       {tcell.ModNone, tcell.KeyNUL, 0},
	"soh":       {tcell.ModNone, tcell.KeySOH, 0},
	"stx":       {tcell.ModNone, tcell.KeySTX, 0},
	"etx":       {tcell.ModNone, tcell.KeyETX, 0},
	"eot":       {tcell.ModNone, tcell.KeyEOT, 0},
	"enq":       {tcell.ModNone, tcell.KeyENQ, 0},
	"ack":       {tcell.ModNone, tcell.KeyACK, 0},
	"bel":       {tcell.ModNone, tcell.KeyBEL, 0},
	"bs":        {tcell.ModNone, tcell.KeyBS, 0},
	"tab":       {tcell.ModNone, tcell.KeyTAB, 0},
	"lf":        {tcell.ModNone, tcell.KeyLF, 0},
	"vt":        {tcell.ModNone, tcell.KeyVT, 0},
	"ff":        {tcell.ModNone, tcell.KeyFF, 0},
	"cr":        {tcell.ModNone, tcell.KeyCR, 0},
	"so":        {tcell.ModNone, tcell.KeySO, 0},
	"si":        {tcell.ModNone, tcell.KeySI, 0},
	"dle":       {tcell.ModNone, tcell.KeyDLE, 0},
	"dc1":       {tcell.ModNone, tcell.KeyDC1, 0},
	"dc2":       {tcell.ModNone, tcell.KeyDC2, 0},
	"dc3":       {tcell.ModNone, tcell.KeyDC3, 0},
	"dc4":       {tcell.ModNone, tcell.KeyDC4, 0},
	"nak":       {tcell.ModNone, tcell.KeyNAK, 0},
	"syn":       {tcell.ModNone, tcell.KeySYN, 0},
	"etb":       {tcell.ModNone, tcell.KeyETB, 0},
	"can":       {tcell.ModNone, tcell.KeyCAN, 0},
	"em":        {tcell.ModNone, tcell.KeyEM, 0},
	"sub":       {tcell.ModNone, tcell.KeySUB, 0},
	"esc":       {tcell.ModNone, tcell.KeyESC, 0},
	"fs":        {tcell.ModNone, tcell.KeyFS, 0},
	"gs":        {tcell.ModNone, tcell.KeyGS, 0},
	"rs":        {tcell.ModNone, tcell.KeyRS, 0},
	"us":        {tcell.ModNone, tcell.KeyUS, 0},
	"del":       {tcell.ModNone, tcell.KeyDEL, 0},
}

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
