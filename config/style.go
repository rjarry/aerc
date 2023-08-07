package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/emersion/go-message/mail"
	"github.com/gdamore/tcell/v2"
	"github.com/go-ini/ini"
	"github.com/mitchellh/go-homedir"
)

type StyleObject int32

const (
	STYLE_DEFAULT StyleObject = iota
	STYLE_ERROR
	STYLE_WARNING
	STYLE_SUCCESS

	STYLE_TITLE
	STYLE_HEADER

	STYLE_STATUSLINE_DEFAULT
	STYLE_STATUSLINE_ERROR
	STYLE_STATUSLINE_WARNING
	STYLE_STATUSLINE_SUCCESS

	STYLE_MSGLIST_DEFAULT
	STYLE_MSGLIST_UNREAD
	STYLE_MSGLIST_READ
	STYLE_MSGLIST_FLAGGED
	STYLE_MSGLIST_DELETED
	STYLE_MSGLIST_MARKED
	STYLE_MSGLIST_RESULT
	STYLE_MSGLIST_ANSWERED
	STYLE_MSGLIST_THREAD_FOLDED
	STYLE_MSGLIST_GUTTER
	STYLE_MSGLIST_PILL

	STYLE_DIRLIST_DEFAULT
	STYLE_DIRLIST_UNREAD
	STYLE_DIRLIST_RECENT

	STYLE_COMPLETION_DEFAULT
	STYLE_COMPLETION_GUTTER
	STYLE_COMPLETION_PILL

	STYLE_TAB
	STYLE_STACK
	STYLE_SPINNER
	STYLE_BORDER

	STYLE_SELECTOR_DEFAULT
	STYLE_SELECTOR_FOCUSED
	STYLE_SELECTOR_CHOOSER
)

var StyleNames = map[string]StyleObject{
	"default": STYLE_DEFAULT,
	"error":   STYLE_ERROR,
	"warning": STYLE_WARNING,
	"success": STYLE_SUCCESS,

	"title":  STYLE_TITLE,
	"header": STYLE_HEADER,

	"statusline_default": STYLE_STATUSLINE_DEFAULT,
	"statusline_error":   STYLE_STATUSLINE_ERROR,
	"statusline_warning": STYLE_STATUSLINE_WARNING,
	"statusline_success": STYLE_STATUSLINE_SUCCESS,

	"msglist_default":  STYLE_MSGLIST_DEFAULT,
	"msglist_unread":   STYLE_MSGLIST_UNREAD,
	"msglist_read":     STYLE_MSGLIST_READ,
	"msglist_flagged":  STYLE_MSGLIST_FLAGGED,
	"msglist_deleted":  STYLE_MSGLIST_DELETED,
	"msglist_marked":   STYLE_MSGLIST_MARKED,
	"msglist_result":   STYLE_MSGLIST_RESULT,
	"msglist_answered": STYLE_MSGLIST_ANSWERED,
	"msglist_gutter":   STYLE_MSGLIST_GUTTER,
	"msglist_pill":     STYLE_MSGLIST_PILL,

	"msglist_thread_folded": STYLE_MSGLIST_THREAD_FOLDED,

	"dirlist_default": STYLE_DIRLIST_DEFAULT,
	"dirlist_unread":  STYLE_DIRLIST_UNREAD,
	"dirlist_recent":  STYLE_DIRLIST_RECENT,

	"completion_default": STYLE_COMPLETION_DEFAULT,
	"completion_gutter":  STYLE_COMPLETION_GUTTER,
	"completion_pill":    STYLE_COMPLETION_PILL,

	"tab":     STYLE_TAB,
	"stack":   STYLE_STACK,
	"spinner": STYLE_SPINNER,
	"border":  STYLE_BORDER,

	"selector_default": STYLE_SELECTOR_DEFAULT,
	"selector_focused": STYLE_SELECTOR_FOCUSED,
	"selector_chooser": STYLE_SELECTOR_CHOOSER,
}

type Style struct {
	Fg        tcell.Color
	Bg        tcell.Color
	Bold      bool
	Blink     bool
	Underline bool
	Reverse   bool
	Italic    bool
	Dim       bool
	header    string         // only for msglist
	pattern   string         // only for msglist
	re        *regexp.Regexp // only for msglist
}

func (s Style) Get() tcell.Style {
	return tcell.StyleDefault.
		Foreground(s.Fg).
		Background(s.Bg).
		Bold(s.Bold).
		Blink(s.Blink).
		Underline(s.Underline).
		Reverse(s.Reverse).
		Italic(s.Italic).
		Dim(s.Dim)
}

func (s *Style) Normal() {
	s.Bold = false
	s.Blink = false
	s.Underline = false
	s.Reverse = false
	s.Italic = false
	s.Dim = false
}

func (s *Style) Default() *Style {
	s.Fg = tcell.ColorDefault
	s.Bg = tcell.ColorDefault
	return s
}

func (s *Style) Reset() *Style {
	s.Default()
	s.Normal()
	return s
}

func boolSwitch(val string, cur_val bool) (bool, error) {
	switch val {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "toggle":
		return !cur_val, nil
	default:
		return cur_val, errors.New(
			"Bool Switch attribute must be true, false, or toggle")
	}
}

func extractColor(val string) tcell.Color {
	// Check if the string can be interpreted as a number, indicating a
	// reference to the color number. Otherwise retrieve the number based
	// on the name.
	if i, err := strconv.ParseUint(val, 10, 8); err == nil {
		return tcell.PaletteColor(int(i))
	} else {
		return tcell.GetColor(val)
	}
}

func (s *Style) Set(attr, val string) error {
	switch attr {
	case "fg":
		s.Fg = extractColor(val)
	case "bg":
		s.Bg = extractColor(val)
	case "bold":
		if state, err := boolSwitch(val, s.Bold); err != nil {
			return err
		} else {
			s.Bold = state
		}
	case "blink":
		if state, err := boolSwitch(val, s.Blink); err != nil {
			return err
		} else {
			s.Blink = state
		}
	case "underline":
		if state, err := boolSwitch(val, s.Underline); err != nil {
			return err
		} else {
			s.Underline = state
		}
	case "reverse":
		if state, err := boolSwitch(val, s.Reverse); err != nil {
			return err
		} else {
			s.Reverse = state
		}
	case "italic":
		if state, err := boolSwitch(val, s.Italic); err != nil {
			return err
		} else {
			s.Italic = state
		}
	case "dim":
		if state, err := boolSwitch(val, s.Dim); err != nil {
			return err
		} else {
			s.Dim = state
		}
	case "default":
		s.Default()
	case "normal":
		s.Normal()
	default:
		return errors.New("Unknown style attribute: " + attr)
	}

	return nil
}

func (s Style) composeWith(styles []*Style) Style {
	newStyle := s
	for _, st := range styles {
		if st.Fg != s.Fg && st.Fg != tcell.ColorDefault {
			newStyle.Fg = st.Fg
		}
		if st.Bg != s.Bg && st.Bg != tcell.ColorDefault {
			newStyle.Bg = st.Bg
		}
		if st.Bold != s.Bold {
			newStyle.Bold = st.Bold
		}
		if st.Blink != s.Blink {
			newStyle.Blink = st.Blink
		}
		if st.Underline != s.Underline {
			newStyle.Underline = st.Underline
		}
		if st.Reverse != s.Reverse {
			newStyle.Reverse = st.Reverse
		}
		if st.Italic != s.Italic {
			newStyle.Italic = st.Italic
		}
		if st.Dim != s.Dim {
			newStyle.Dim = st.Dim
		}
	}
	return newStyle
}

type StyleConf struct {
	base    Style
	dynamic []Style
}

type StyleSet struct {
	objects  map[StyleObject]*StyleConf
	selected map[StyleObject]*StyleConf
	user     map[string]*Style
	path     string
}

func NewStyleSet() StyleSet {
	ss := StyleSet{
		objects:  make(map[StyleObject]*StyleConf),
		selected: make(map[StyleObject]*StyleConf),
		user:     make(map[string]*Style),
	}
	for _, so := range StyleNames {
		ss.objects[so] = new(StyleConf)
		ss.selected[so] = new(StyleConf)
	}
	return ss
}

func (ss StyleSet) reset() {
	for _, so := range StyleNames {
		ss.objects[so].base.Reset()
		for _, d := range ss.objects[so].dynamic {
			d.Reset()
		}
		ss.selected[so].base.Reset()
		for _, d := range ss.selected[so].dynamic {
			d.Reset()
		}
	}
}

func (c *StyleConf) getStyle(h *mail.Header) *Style {
	if h == nil {
		return &c.base
	}
	for _, s := range c.dynamic {
		val, _ := h.Text(s.header)
		if s.re.MatchString(val) {
			s = c.base.composeWith([]*Style{&s})
			return &s
		}
	}
	return &c.base
}

func (ss StyleSet) Get(so StyleObject, h *mail.Header) tcell.Style {
	return ss.objects[so].getStyle(h).Get()
}

func (ss StyleSet) Selected(so StyleObject, h *mail.Header) tcell.Style {
	return ss.selected[so].getStyle(h).Get()
}

func (ss StyleSet) UserStyle(name string) tcell.Style {
	if style, found := ss.user[name]; found {
		return style.Get()
	}
	return tcell.StyleDefault
}

func (ss StyleSet) Compose(
	so StyleObject, sos []StyleObject, h *mail.Header,
) tcell.Style {
	base := *ss.objects[so].getStyle(h)
	styles := make([]*Style, len(sos))
	for i, so := range sos {
		styles[i] = ss.objects[so].getStyle(h)
	}

	return base.composeWith(styles).Get()
}

func (ss StyleSet) ComposeSelected(
	so StyleObject, sos []StyleObject, h *mail.Header,
) tcell.Style {
	base := *ss.selected[so].getStyle(h)
	styles := make([]*Style, len(sos))
	for i, so := range sos {
		styles[i] = ss.selected[so].getStyle(h)
	}

	return base.composeWith(styles).Get()
}

func findStyleSet(stylesetName string, stylesetsDir []string) (string, error) {
	for _, dir := range stylesetsDir {
		stylesetPath, err := homedir.Expand(path.Join(dir, stylesetName))
		if err != nil {
			return "", err
		}

		if _, err := os.Stat(stylesetPath); os.IsNotExist(err) {
			continue
		}

		return stylesetPath, nil
	}

	return "", fmt.Errorf(
		"Can't find styleset %q in any of %v", stylesetName, stylesetsDir)
}

func (ss *StyleSet) ParseStyleSet(file *ini.File) error {
	ss.reset()

	defaultSection, err := file.GetSection(ini.DefaultSection)
	if err != nil {
		return err
	}

	// parse non-selected items first
	for _, key := range defaultSection.Keys() {
		err = ss.parseKey(key, false)
		if err != nil {
			return err
		}
	}
	// override with selected items afterwards
	for _, key := range defaultSection.Keys() {
		err = ss.parseKey(key, true)
		if err != nil {
			return err
		}
	}

	user, err := file.GetSection("user")
	if err != nil {
		// This errors if the section doesn't exist, which is ok
		return nil
	}
	for _, key := range user.KeyStrings() {
		tokens := strings.Split(key, ".")
		var styleName, attr string
		switch len(tokens) {
		case 2:
			styleName, attr = tokens[0], tokens[1]
		default:
			return errors.New("Style parsing error: " + key)
		}
		val := user.KeysHash()[key]
		s, ok := ss.user[styleName]
		if !ok {
			// Haven't seen this name before, add it to the map
			s = &Style{}
			ss.user[styleName] = s
		}
		if err := s.Set(attr, val); err != nil {
			return err
		}
	}

	return nil
}

var styleObjRe = regexp.MustCompile(`^([\w\*\?]+)(?:\.([\w-]+),(.+?))?(\.selected)?\.(\w+)$`)

func (ss *StyleSet) parseKey(key *ini.Key, selected bool) error {
	groups := styleObjRe.FindStringSubmatch(key.Name())
	if groups == nil {
		return errors.New("invalid style syntax: " + key.Name())
	}
	if (groups[4] == ".selected") != selected {
		return nil
	}
	obj, attr := groups[1], groups[5]
	header, pattern := groups[2], groups[3]

	objRe, err := fnmatchToRegex(obj)
	if err != nil {
		return err
	}
	num := 0
	for sn, so := range StyleNames {
		if !objRe.MatchString(sn) {
			continue
		}
		if !selected {
			err = ss.objects[so].update(header, pattern, attr, key.Value())
			if err != nil {
				return err
			}
		}
		err = ss.selected[so].update(header, pattern, attr, key.Value())
		if err != nil {
			return err
		}
		num++
	}
	if num == 0 {
		return errors.New("unknown style object: " + obj)
	}
	return nil
}

func (c *StyleConf) update(header, pattern, attr, val string) error {
	if header == "" || pattern == "" {
		return (&c.base).Set(attr, val)
	}
	for i := range c.dynamic {
		s := &c.dynamic[i]
		if s.header == header && s.pattern == pattern {
			return s.Set(attr, val)
		}
	}
	s := Style{
		header:  header,
		pattern: pattern,
	}
	if strings.HasPrefix(pattern, "~") {
		pattern = pattern[1:]
	} else {
		pattern = "^" + regexp.QuoteMeta(pattern) + "$"
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	err = (&s).Set(attr, val)
	if err != nil {
		return err
	}
	s.re = re
	c.dynamic = append(c.dynamic, s)
	return nil
}

func (ss *StyleSet) LoadStyleSet(stylesetName string, stylesetDirs []string) error {
	filepath, err := findStyleSet(stylesetName, stylesetDirs)
	if err != nil {
		return err
	}

	var options ini.LoadOptions
	options.SpaceBeforeInlineComment = true

	file, err := ini.LoadSources(options, filepath)
	if err != nil {
		return err
	}

	ss.path = filepath

	return ss.ParseStyleSet(file)
}

func fnmatchToRegex(pattern string) (*regexp.Regexp, error) {
	p := regexp.QuoteMeta(pattern)
	p = strings.ReplaceAll(p, `\*`, `.*`)
	return regexp.Compile(strings.ReplaceAll(p, `\?`, `.`))
}
