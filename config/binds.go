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
	"unicode"
	"unicode/utf8"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rockorager/vaxis"
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
	Modifiers vaxis.ModifierMask
	Key       rune
}

type Binding struct {
	Output []KeyStroke
	Input  []KeyStroke

	Annotation string
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
	wizard.ExKey = KeyStroke{Key: 'e', Modifiers: vaxis.ModCtrl}
	wizard.Globals = false
	quit, _ := ParseBinding("<C-q>", ":quit<Enter>", "Quit aerc")
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

func parseBindsFromFile(root string, filename string) error {
	log.Debugf("Parsing key bindings configuration from %s", filename)
	binds, err := ini.LoadSources(ini.LoadOptions{
		KeyValueDelimiters: "=",
		// IgnoreInlineComment is set to true which tells ini's parser
		// to treat comments (#) on the same line as part of the value;
		// hence we need cut the comment off ourselves later
		IgnoreInlineComment: true,
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
		// Handle :: delimiter
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
	SetBindsFilename(filename)
	if err := parseBindsFromFile(root, filename); err != nil {
		return fmt.Errorf("%s: %w", filename, err)
	}

	return nil
}

func LoadBindingSection(sec *ini.Section) (*KeyBindings, error) {
	bindings := NewKeyBindings()
	for _, k := range sec.Keys() {
		key := k.Name()
		value, annotation, _ := strings.Cut(k.String(), " # ")
		value = strings.TrimSpace(value)
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
			annotation = strings.TrimSpace(annotation)
			binding, err := ParseBinding(key, value, annotation)
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
		ExKey:            KeyStroke{0, ':'},
		CompleteKey:      KeyStroke{0, vaxis.KeyTab},
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
	for _, b := range bindings {
		merged.contextualBinds = append(merged.contextualBinds, b.contextualBinds...)
		for t, c := range b.contextualCounts {
			merged.contextualCounts[t] += c
		}
	}
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
			if (ks.Modifiers == stroke.Modifiers || ks.Modifiers == vaxis.ModifierMask(0)) && ks.Key == stroke.Key {
				switch name {
				case "cr":
					s = "<enter>"
				case "space":
					s = " "
				case "semicolon":
					s = ";"
				default:
					s = fmt.Sprintf("<%s>", name)
				}
				// remove any modifiers this named key comes
				// with so we format properly
				stroke.Modifiers &^= ks.Modifiers
				break
			}
		}
		if stroke.Modifiers&vaxis.ModCtrl > 0 {
			sb.WriteString("c-")
		}
		if stroke.Modifiers&vaxis.ModAlt > 0 {
			sb.WriteString("a-")
		}
		if stroke.Modifiers&vaxis.ModShift > 0 {
			sb.WriteString("s-")
		}
		if s == "" && stroke.Key < unicode.MaxRune {
			s = string(stroke.Key)
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
	"space":     {vaxis.ModifierMask(0), ' '},
	"semicolon": {vaxis.ModifierMask(0), ';'},
	"enter":     {vaxis.ModifierMask(0), vaxis.KeyEnter},
	"up":        {vaxis.ModifierMask(0), vaxis.KeyUp},
	"down":      {vaxis.ModifierMask(0), vaxis.KeyDown},
	"right":     {vaxis.ModifierMask(0), vaxis.KeyRight},
	"left":      {vaxis.ModifierMask(0), vaxis.KeyLeft},
	"upleft":    {vaxis.ModifierMask(0), vaxis.KeyUpLeft},
	"upright":   {vaxis.ModifierMask(0), vaxis.KeyUpRight},
	"downleft":  {vaxis.ModifierMask(0), vaxis.KeyDownLeft},
	"downright": {vaxis.ModifierMask(0), vaxis.KeyDownRight},
	"center":    {vaxis.ModifierMask(0), vaxis.KeyCenter},
	"pgup":      {vaxis.ModifierMask(0), vaxis.KeyPgUp},
	"pgdn":      {vaxis.ModifierMask(0), vaxis.KeyPgDown},
	"home":      {vaxis.ModifierMask(0), vaxis.KeyHome},
	"end":       {vaxis.ModifierMask(0), vaxis.KeyEnd},
	"insert":    {vaxis.ModifierMask(0), vaxis.KeyInsert},
	"delete":    {vaxis.ModifierMask(0), vaxis.KeyDelete},
	"backspace": {vaxis.ModifierMask(0), vaxis.KeyBackspace},
	// "help":      {vaxis.ModifierMask(0), vaxis.KeyHelp},
	"exit":    {vaxis.ModifierMask(0), vaxis.KeyExit},
	"clear":   {vaxis.ModifierMask(0), vaxis.KeyClear},
	"cancel":  {vaxis.ModifierMask(0), vaxis.KeyCancel},
	"print":   {vaxis.ModifierMask(0), vaxis.KeyPrint},
	"pause":   {vaxis.ModifierMask(0), vaxis.KeyPause},
	"backtab": {vaxis.ModShift, vaxis.KeyTab},
	"f1":      {vaxis.ModifierMask(0), vaxis.KeyF01},
	"f2":      {vaxis.ModifierMask(0), vaxis.KeyF02},
	"f3":      {vaxis.ModifierMask(0), vaxis.KeyF03},
	"f4":      {vaxis.ModifierMask(0), vaxis.KeyF04},
	"f5":      {vaxis.ModifierMask(0), vaxis.KeyF05},
	"f6":      {vaxis.ModifierMask(0), vaxis.KeyF06},
	"f7":      {vaxis.ModifierMask(0), vaxis.KeyF07},
	"f8":      {vaxis.ModifierMask(0), vaxis.KeyF08},
	"f9":      {vaxis.ModifierMask(0), vaxis.KeyF09},
	"f10":     {vaxis.ModifierMask(0), vaxis.KeyF10},
	"f11":     {vaxis.ModifierMask(0), vaxis.KeyF11},
	"f12":     {vaxis.ModifierMask(0), vaxis.KeyF12},
	"f13":     {vaxis.ModifierMask(0), vaxis.KeyF13},
	"f14":     {vaxis.ModifierMask(0), vaxis.KeyF14},
	"f15":     {vaxis.ModifierMask(0), vaxis.KeyF15},
	"f16":     {vaxis.ModifierMask(0), vaxis.KeyF16},
	"f17":     {vaxis.ModifierMask(0), vaxis.KeyF17},
	"f18":     {vaxis.ModifierMask(0), vaxis.KeyF18},
	"f19":     {vaxis.ModifierMask(0), vaxis.KeyF19},
	"f20":     {vaxis.ModifierMask(0), vaxis.KeyF20},
	"f21":     {vaxis.ModifierMask(0), vaxis.KeyF21},
	"f22":     {vaxis.ModifierMask(0), vaxis.KeyF22},
	"f23":     {vaxis.ModifierMask(0), vaxis.KeyF23},
	"f24":     {vaxis.ModifierMask(0), vaxis.KeyF24},
	"f25":     {vaxis.ModifierMask(0), vaxis.KeyF25},
	"f26":     {vaxis.ModifierMask(0), vaxis.KeyF26},
	"f27":     {vaxis.ModifierMask(0), vaxis.KeyF27},
	"f28":     {vaxis.ModifierMask(0), vaxis.KeyF28},
	"f29":     {vaxis.ModifierMask(0), vaxis.KeyF29},
	"f30":     {vaxis.ModifierMask(0), vaxis.KeyF30},
	"f31":     {vaxis.ModifierMask(0), vaxis.KeyF31},
	"f32":     {vaxis.ModifierMask(0), vaxis.KeyF32},
	"f33":     {vaxis.ModifierMask(0), vaxis.KeyF33},
	"f34":     {vaxis.ModifierMask(0), vaxis.KeyF34},
	"f35":     {vaxis.ModifierMask(0), vaxis.KeyF35},
	"f36":     {vaxis.ModifierMask(0), vaxis.KeyF36},
	"f37":     {vaxis.ModifierMask(0), vaxis.KeyF37},
	"f38":     {vaxis.ModifierMask(0), vaxis.KeyF38},
	"f39":     {vaxis.ModifierMask(0), vaxis.KeyF39},
	"f40":     {vaxis.ModifierMask(0), vaxis.KeyF40},
	"f41":     {vaxis.ModifierMask(0), vaxis.KeyF41},
	"f42":     {vaxis.ModifierMask(0), vaxis.KeyF42},
	"f43":     {vaxis.ModifierMask(0), vaxis.KeyF43},
	"f44":     {vaxis.ModifierMask(0), vaxis.KeyF44},
	"f45":     {vaxis.ModifierMask(0), vaxis.KeyF45},
	"f46":     {vaxis.ModifierMask(0), vaxis.KeyF46},
	"f47":     {vaxis.ModifierMask(0), vaxis.KeyF47},
	"f48":     {vaxis.ModifierMask(0), vaxis.KeyF48},
	"f49":     {vaxis.ModifierMask(0), vaxis.KeyF49},
	"f50":     {vaxis.ModifierMask(0), vaxis.KeyF50},
	"f51":     {vaxis.ModifierMask(0), vaxis.KeyF51},
	"f52":     {vaxis.ModifierMask(0), vaxis.KeyF52},
	"f53":     {vaxis.ModifierMask(0), vaxis.KeyF53},
	"f54":     {vaxis.ModifierMask(0), vaxis.KeyF54},
	"f55":     {vaxis.ModifierMask(0), vaxis.KeyF55},
	"f56":     {vaxis.ModifierMask(0), vaxis.KeyF56},
	"f57":     {vaxis.ModifierMask(0), vaxis.KeyF57},
	"f58":     {vaxis.ModifierMask(0), vaxis.KeyF58},
	"f59":     {vaxis.ModifierMask(0), vaxis.KeyF59},
	"f60":     {vaxis.ModifierMask(0), vaxis.KeyF60},
	"f61":     {vaxis.ModifierMask(0), vaxis.KeyF61},
	"f62":     {vaxis.ModifierMask(0), vaxis.KeyF62},
	"f63":     {vaxis.ModifierMask(0), vaxis.KeyF63},
	"nul":     {vaxis.ModCtrl, ' '},
	"soh":     {vaxis.ModCtrl, 'a'},
	"stx":     {vaxis.ModCtrl, 'b'},
	"etx":     {vaxis.ModCtrl, 'c'},
	"eot":     {vaxis.ModCtrl, 'd'},
	"enq":     {vaxis.ModCtrl, 'e'},
	"ack":     {vaxis.ModCtrl, 'f'},
	"bel":     {vaxis.ModCtrl, 'g'},
	"bs":      {vaxis.ModCtrl, 'h'},
	"tab":     {vaxis.ModifierMask(0), vaxis.KeyTab},
	"lf":      {vaxis.ModCtrl, 'j'},
	"vt":      {vaxis.ModCtrl, 'k'},
	"ff":      {vaxis.ModCtrl, 'l'},
	"cr":      {vaxis.ModifierMask(0), vaxis.KeyEnter},
	"so":      {vaxis.ModCtrl, 'n'},
	"si":      {vaxis.ModCtrl, 'o'},
	"dle":     {vaxis.ModCtrl, 'p'},
	"dc1":     {vaxis.ModCtrl, 'q'},
	"dc2":     {vaxis.ModCtrl, 'r'},
	"dc3":     {vaxis.ModCtrl, 's'},
	"dc4":     {vaxis.ModCtrl, 't'},
	"nak":     {vaxis.ModCtrl, 'u'},
	"syn":     {vaxis.ModCtrl, 'v'},
	"etb":     {vaxis.ModCtrl, 'w'},
	"can":     {vaxis.ModCtrl, 'x'},
	"em":      {vaxis.ModCtrl, 'y'},
	"sub":     {vaxis.ModCtrl, 'z'},
	"esc":     {vaxis.ModifierMask(0), vaxis.KeyEsc},
	"fs":      {vaxis.ModCtrl, '\\'},
	"gs":      {vaxis.ModCtrl, ']'},
	"rs":      {vaxis.ModCtrl, '^'},
	"us":      {vaxis.ModCtrl, '_'},
	"del":     {vaxis.ModifierMask(0), vaxis.KeyDelete},
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
			args := strings.Split(name, "-")
			// check if the last char was a '-' and we'll add it
			// back. We check for "--" in case it was an invalid
			// keystroke (ie <C->)
			if strings.HasSuffix(name, "--") {
				args = append(args, "-")
			}
			ks := KeyStroke{}
			for i, arg := range args {
				if i == len(args)-1 {
					key, ok := keyNames[strings.ToLower(arg)]
					if !ok {
						r, n := utf8.DecodeRuneInString(arg)
						if n != len(arg) {
							return nil, fmt.Errorf("Unknown key '%s'", name)
						}
						key = KeyStroke{Key: r}
					}
					ks.Key = key.Key
					ks.Modifiers |= key.Modifiers
					strokes = append(strokes, ks)
				}
				switch strings.ToLower(arg) {
				case "s", "S":
					ks.Modifiers |= vaxis.ModShift
				case "a", "A":
					ks.Modifiers |= vaxis.ModAlt
				case "c", "C":
					ks.Modifiers |= vaxis.ModCtrl
				}
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
				Modifiers: vaxis.ModifierMask(0),
				Key:       tok,
			})
		}
	}
	return strokes, nil
}

func ParseBinding(input, output, annotation string) (*Binding, error) {
	in, err := ParseKeyStrokes(input)
	if err != nil {
		return nil, err
	}
	out, err := ParseKeyStrokes(output)
	if err != nil {
		return nil, err
	}
	return &Binding{
		Input:      in,
		Output:     out,
		Annotation: annotation,
	}, nil
}
