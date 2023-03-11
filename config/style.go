package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

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

type StyleSet struct {
	objects  map[StyleObject]*Style
	selected map[StyleObject]*Style
	user     map[string]*Style
	path     string
}

func NewStyleSet() StyleSet {
	ss := StyleSet{
		objects:  make(map[StyleObject]*Style),
		selected: make(map[StyleObject]*Style),
		user:     make(map[string]*Style),
	}
	for _, so := range StyleNames {
		ss.objects[so] = new(Style)
		ss.selected[so] = new(Style)
	}

	return ss
}

func (ss StyleSet) reset() {
	for _, so := range StyleNames {
		ss.objects[so].Reset()
		ss.selected[so].Reset()
	}
}

func (ss StyleSet) Get(so StyleObject) tcell.Style {
	return ss.objects[so].Get()
}

func (ss StyleSet) Selected(so StyleObject) tcell.Style {
	return ss.selected[so].Get()
}

func (ss StyleSet) UserStyle(name string) tcell.Style {
	if style, found := ss.user[name]; found {
		return style.Get()
	}
	return tcell.StyleDefault
}

func (ss StyleSet) Compose(so StyleObject, sos []StyleObject) tcell.Style {
	base := *ss.objects[so]
	styles := make([]*Style, len(sos))
	for i, so := range sos {
		styles[i] = ss.objects[so]
	}

	return base.composeWith(styles).Get()
}

func (ss StyleSet) ComposeSelected(so StyleObject,
	sos []StyleObject,
) tcell.Style {
	base := *ss.selected[so]
	styles := make([]*Style, len(sos))
	for i, so := range sos {
		styles[i] = ss.selected[so]
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

	selectedKeys := []string{}

	for _, key := range defaultSection.KeyStrings() {
		tokens := strings.Split(key, ".")
		var styleName, attr string
		switch len(tokens) {
		case 2:
			styleName, attr = tokens[0], tokens[1]
		case 3:
			if tokens[1] != "selected" {
				return errors.New("Unknown modifier: " + tokens[1])
			}
			selectedKeys = append(selectedKeys, key)
			continue
		default:
			return errors.New("Style parsing error: " + key)
		}
		val := defaultSection.KeysHash()[key]

		if strings.ContainsAny(styleName, "*?") {
			regex := fnmatchToRegex(styleName)
			for sn, so := range StyleNames {
				matched, err := regexp.MatchString(regex, sn)
				if err != nil {
					return err
				}

				if !matched {
					continue
				}

				if err := ss.objects[so].Set(attr, val); err != nil {
					return err
				}
				if err := ss.selected[so].Set(attr, val); err != nil {
					return err
				}
			}
		} else {
			so, ok := StyleNames[styleName]
			if !ok {
				return errors.New("Unknown style object: " + styleName)
			}
			if err := ss.objects[so].Set(attr, val); err != nil {
				return err
			}
			if err := ss.selected[so].Set(attr, val); err != nil {
				return err
			}
		}
	}

	for _, key := range selectedKeys {
		tokens := strings.Split(key, ".")
		styleName, modifier, attr := tokens[0], tokens[1], tokens[2]
		if modifier != "selected" {
			return errors.New("Unknown modifier: " + modifier)
		}

		val := defaultSection.KeysHash()[key]

		if strings.ContainsAny(styleName, "*?") {
			regex := fnmatchToRegex(styleName)
			for sn, so := range StyleNames {
				matched, err := regexp.MatchString(regex, sn)
				if err != nil {
					return err
				}

				if !matched {
					continue
				}

				if err := ss.selected[so].Set(attr, val); err != nil {
					return err
				}
			}
		} else {
			so, ok := StyleNames[styleName]
			if !ok {
				return errors.New("Unknown style object: " + styleName)
			}
			if err := ss.selected[so].Set(attr, val); err != nil {
				return err
			}
		}
	}

	for _, key := range defaultSection.KeyStrings() {
		tokens := strings.Split(key, ".")
		styleName, attr := tokens[0], tokens[1]
		val := defaultSection.KeysHash()[key]

		if styleName != "selected" {
			continue
		}

		for _, so := range StyleNames {
			if err := ss.selected[so].Set(attr, val); err != nil {
				return err
			}
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

func fnmatchToRegex(pattern string) string {
	n := len(pattern)
	var regex strings.Builder

	for i := 0; i < n; i++ {
		switch pattern[i] {
		case '*':
			regex.WriteString(".*")
		case '?':
			regex.WriteByte('.')
		default:
			regex.WriteByte(pattern[i])
		}
	}

	return regex.String()
}
