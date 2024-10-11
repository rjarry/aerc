package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/lib/xdg"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/emersion/go-message/mail"
	"github.com/go-ini/ini"
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
	STYLE_MSGLIST_FORWARDED
	STYLE_MSGLIST_THREAD_FOLDED
	STYLE_MSGLIST_GUTTER
	STYLE_MSGLIST_PILL
	STYLE_MSGLIST_THREAD_CONTEXT
	STYLE_MSGLIST_THREAD_ORPHAN

	STYLE_DIRLIST_DEFAULT
	STYLE_DIRLIST_UNREAD
	STYLE_DIRLIST_RECENT

	STYLE_PART_SWITCHER
	STYLE_PART_FILENAME
	STYLE_PART_MIMETYPE

	STYLE_COMPLETION_DEFAULT
	STYLE_COMPLETION_DESCRIPTION
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

	"msglist_default":   STYLE_MSGLIST_DEFAULT,
	"msglist_unread":    STYLE_MSGLIST_UNREAD,
	"msglist_read":      STYLE_MSGLIST_READ,
	"msglist_flagged":   STYLE_MSGLIST_FLAGGED,
	"msglist_deleted":   STYLE_MSGLIST_DELETED,
	"msglist_marked":    STYLE_MSGLIST_MARKED,
	"msglist_result":    STYLE_MSGLIST_RESULT,
	"msglist_answered":  STYLE_MSGLIST_ANSWERED,
	"msglist_forwarded": STYLE_MSGLIST_FORWARDED,
	"msglist_gutter":    STYLE_MSGLIST_GUTTER,
	"msglist_pill":      STYLE_MSGLIST_PILL,

	"msglist_thread_folded":  STYLE_MSGLIST_THREAD_FOLDED,
	"msglist_thread_context": STYLE_MSGLIST_THREAD_CONTEXT,
	"msglist_thread_orphan":  STYLE_MSGLIST_THREAD_ORPHAN,

	"dirlist_default": STYLE_DIRLIST_DEFAULT,
	"dirlist_unread":  STYLE_DIRLIST_UNREAD,
	"dirlist_recent":  STYLE_DIRLIST_RECENT,

	"part_switcher": STYLE_PART_SWITCHER,
	"part_filename": STYLE_PART_FILENAME,
	"part_mimetype": STYLE_PART_MIMETYPE,

	"completion_default":     STYLE_COMPLETION_DEFAULT,
	"completion_description": STYLE_COMPLETION_DESCRIPTION,
	"completion_gutter":      STYLE_COMPLETION_GUTTER,
	"completion_pill":        STYLE_COMPLETION_PILL,

	"tab":     STYLE_TAB,
	"stack":   STYLE_STACK,
	"spinner": STYLE_SPINNER,
	"border":  STYLE_BORDER,

	"selector_default": STYLE_SELECTOR_DEFAULT,
	"selector_focused": STYLE_SELECTOR_FOCUSED,
	"selector_chooser": STYLE_SELECTOR_CHOOSER,
}

type Style struct {
	Fg        vaxis.Color
	Bg        vaxis.Color
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

func (s Style) Get() vaxis.Style {
	vx := vaxis.Style{
		Foreground: s.Fg,
		Background: s.Bg,
	}
	if s.Bold {
		vx.Attribute |= vaxis.AttrBold
	}
	if s.Blink {
		vx.Attribute |= vaxis.AttrBlink
	}
	if s.Underline {
		vx.UnderlineStyle |= vaxis.UnderlineSingle
	}
	if s.Reverse {
		vx.Attribute |= vaxis.AttrReverse
	}
	if s.Italic {
		vx.Attribute |= vaxis.AttrItalic
	}
	if s.Dim {
		vx.Attribute |= vaxis.AttrDim
	}
	return vx
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
	s.Fg = 0
	s.Bg = 0
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

func extractColor(val string) vaxis.Color {
	// Check if the string can be interpreted as a number, indicating a
	// reference to the color number. Otherwise retrieve the number based
	// on the name.
	if i, err := strconv.ParseUint(val, 10, 8); err == nil {
		return vaxis.IndexColor(uint8(i))
	}
	if strings.HasPrefix(val, "#") {
		val = strings.TrimPrefix(val, "#")
		hex, err := strconv.ParseUint(val, 16, 32)
		if err != nil {
			return 0
		}
		return vaxis.HexColor(uint32(hex))
	}
	return colorNames[val]
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
		if st.Fg != s.Fg && st.Fg != 0 {
			newStyle.Fg = st.Fg
		}
		if st.Bg != s.Bg && st.Bg != 0 {
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

const defaultStyleset string = `
*.selected.bg = 12
*.selected.fg = 15
*.selected.bold = true
statusline_*.dim = true
statusline_*.bg = 8
statusline_*.fg = 15
*warning.fg = 3
*success.fg = 2
*error.fg = 1
*error.bold = true
border.fg = 12
border.bold = true
title.bg = 12
title.fg = 15
title.bold = true
header.fg = 4
header.bold = true
msglist_unread.bold = true
msglist_deleted.dim = true
msglist_marked.bg = 6
msglist_marked.fg = 15
msglist_pill.bg = 12
msglist_pill.fg = 15
part_mimetype.fg = 12
selector_chooser.bold = true
selector_focused.bold = true
selector_focused.bg = 12
selector_focused.fg = 15
completion_*.bg = 8
completion_pill.bg = 12
completion_default.fg = 15
completion_description.fg = 15
completion_description.dim = true
`

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
	f, err := ini.Load([]byte(defaultStyleset))
	if err == nil {
		err = ss.ParseStyleSet(f)
	}
	if err != nil {
		panic(err)
	}
	return ss
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

func (ss StyleSet) Get(so StyleObject, h *mail.Header) vaxis.Style {
	return ss.objects[so].getStyle(h).Get()
}

func (ss StyleSet) Selected(so StyleObject, h *mail.Header) vaxis.Style {
	return ss.selected[so].getStyle(h).Get()
}

func (ss StyleSet) UserStyle(name string) vaxis.Style {
	if style, found := ss.user[name]; found {
		return style.Get()
	}
	return vaxis.Style{}
}

func (ss StyleSet) Compose(
	so StyleObject, sos []StyleObject, h *mail.Header,
) vaxis.Style {
	base := *ss.objects[so].getStyle(h)
	styles := make([]*Style, len(sos))
	for i, so := range sos {
		styles[i] = ss.objects[so].getStyle(h)
	}

	return base.composeWith(styles).Get()
}

func (ss StyleSet) ComposeSelected(
	so StyleObject, sos []StyleObject, h *mail.Header,
) vaxis.Style {
	base := *ss.selected[so].getStyle(h)
	styles := make([]*Style, len(sos))
	for i, so := range sos {
		styles[i] = ss.selected[so].getStyle(h)
	}

	return base.composeWith(styles).Get()
}

func findStyleSet(stylesetName string, stylesetsDir []string) (string, error) {
	for _, dir := range stylesetsDir {
		stylesetPath := xdg.ExpandHome(dir, stylesetName)
		if _, err := os.Stat(stylesetPath); os.IsNotExist(err) {
			continue
		}

		return stylesetPath, nil
	}

	return "", fmt.Errorf(
		"Can't find styleset %q in any of %v", stylesetName, stylesetsDir)
}

func (ss *StyleSet) ParseStyleSet(file *ini.File) error {
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
			return fmt.Errorf("[user].%s=%s: %w", key, val, err)
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
				return fmt.Errorf("%s=%s: %w", key.Name(), key.Value(), err)
			}
		}
		err = ss.selected[so].update(header, pattern, attr, key.Value())
		if err != nil {
			return fmt.Errorf("%s=%s: %w", key.Name(), key.Value(), err)
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

var colorNames = map[string]vaxis.Color{
	"black":                vaxis.IndexColor(0),
	"maroon":               vaxis.IndexColor(1),
	"green":                vaxis.IndexColor(2),
	"olive":                vaxis.IndexColor(3),
	"navy":                 vaxis.IndexColor(4),
	"purple":               vaxis.IndexColor(5),
	"teal":                 vaxis.IndexColor(6),
	"silver":               vaxis.IndexColor(7),
	"gray":                 vaxis.IndexColor(8),
	"red":                  vaxis.IndexColor(9),
	"lime":                 vaxis.IndexColor(10),
	"yellow":               vaxis.IndexColor(11),
	"blue":                 vaxis.IndexColor(12),
	"fuchsia":              vaxis.IndexColor(13),
	"aqua":                 vaxis.IndexColor(14),
	"white":                vaxis.IndexColor(15),
	"aliceblue":            vaxis.HexColor(0xF0F8FF),
	"antiquewhite":         vaxis.HexColor(0xFAEBD7),
	"aquamarine":           vaxis.HexColor(0x7FFFD4),
	"azure":                vaxis.HexColor(0xF0FFFF),
	"beige":                vaxis.HexColor(0xF5F5DC),
	"bisque":               vaxis.HexColor(0xFFE4C4),
	"blanchedalmond":       vaxis.HexColor(0xFFEBCD),
	"blueviolet":           vaxis.HexColor(0x8A2BE2),
	"brown":                vaxis.HexColor(0xA52A2A),
	"burlywood":            vaxis.HexColor(0xDEB887),
	"cadetblue":            vaxis.HexColor(0x5F9EA0),
	"chartreuse":           vaxis.HexColor(0x7FFF00),
	"chocolate":            vaxis.HexColor(0xD2691E),
	"coral":                vaxis.HexColor(0xFF7F50),
	"cornflowerblue":       vaxis.HexColor(0x6495ED),
	"cornsilk":             vaxis.HexColor(0xFFF8DC),
	"crimson":              vaxis.HexColor(0xDC143C),
	"darkblue":             vaxis.HexColor(0x00008B),
	"darkcyan":             vaxis.HexColor(0x008B8B),
	"darkgoldenrod":        vaxis.HexColor(0xB8860B),
	"darkgray":             vaxis.HexColor(0xA9A9A9),
	"darkgreen":            vaxis.HexColor(0x006400),
	"darkkhaki":            vaxis.HexColor(0xBDB76B),
	"darkmagenta":          vaxis.HexColor(0x8B008B),
	"darkolivegreen":       vaxis.HexColor(0x556B2F),
	"darkorange":           vaxis.HexColor(0xFF8C00),
	"darkorchid":           vaxis.HexColor(0x9932CC),
	"darkred":              vaxis.HexColor(0x8B0000),
	"darksalmon":           vaxis.HexColor(0xE9967A),
	"darkseagreen":         vaxis.HexColor(0x8FBC8F),
	"darkslateblue":        vaxis.HexColor(0x483D8B),
	"darkslategray":        vaxis.HexColor(0x2F4F4F),
	"darkturquoise":        vaxis.HexColor(0x00CED1),
	"darkviolet":           vaxis.HexColor(0x9400D3),
	"deeppink":             vaxis.HexColor(0xFF1493),
	"deepskyblue":          vaxis.HexColor(0x00BFFF),
	"dimgray":              vaxis.HexColor(0x696969),
	"dodgerblue":           vaxis.HexColor(0x1E90FF),
	"firebrick":            vaxis.HexColor(0xB22222),
	"floralwhite":          vaxis.HexColor(0xFFFAF0),
	"forestgreen":          vaxis.HexColor(0x228B22),
	"gainsboro":            vaxis.HexColor(0xDCDCDC),
	"ghostwhite":           vaxis.HexColor(0xF8F8FF),
	"gold":                 vaxis.HexColor(0xFFD700),
	"goldenrod":            vaxis.HexColor(0xDAA520),
	"greenyellow":          vaxis.HexColor(0xADFF2F),
	"honeydew":             vaxis.HexColor(0xF0FFF0),
	"hotpink":              vaxis.HexColor(0xFF69B4),
	"indianred":            vaxis.HexColor(0xCD5C5C),
	"indigo":               vaxis.HexColor(0x4B0082),
	"ivory":                vaxis.HexColor(0xFFFFF0),
	"khaki":                vaxis.HexColor(0xF0E68C),
	"lavender":             vaxis.HexColor(0xE6E6FA),
	"lavenderblush":        vaxis.HexColor(0xFFF0F5),
	"lawngreen":            vaxis.HexColor(0x7CFC00),
	"lemonchiffon":         vaxis.HexColor(0xFFFACD),
	"lightblue":            vaxis.HexColor(0xADD8E6),
	"lightcoral":           vaxis.HexColor(0xF08080),
	"lightcyan":            vaxis.HexColor(0xE0FFFF),
	"lightgoldenrodyellow": vaxis.HexColor(0xFAFAD2),
	"lightgray":            vaxis.HexColor(0xD3D3D3),
	"lightgreen":           vaxis.HexColor(0x90EE90),
	"lightpink":            vaxis.HexColor(0xFFB6C1),
	"lightsalmon":          vaxis.HexColor(0xFFA07A),
	"lightseagreen":        vaxis.HexColor(0x20B2AA),
	"lightskyblue":         vaxis.HexColor(0x87CEFA),
	"lightslategray":       vaxis.HexColor(0x778899),
	"lightsteelblue":       vaxis.HexColor(0xB0C4DE),
	"lightyellow":          vaxis.HexColor(0xFFFFE0),
	"limegreen":            vaxis.HexColor(0x32CD32),
	"linen":                vaxis.HexColor(0xFAF0E6),
	"mediumaquamarine":     vaxis.HexColor(0x66CDAA),
	"mediumblue":           vaxis.HexColor(0x0000CD),
	"mediumorchid":         vaxis.HexColor(0xBA55D3),
	"mediumpurple":         vaxis.HexColor(0x9370DB),
	"mediumseagreen":       vaxis.HexColor(0x3CB371),
	"mediumslateblue":      vaxis.HexColor(0x7B68EE),
	"mediumspringgreen":    vaxis.HexColor(0x00FA9A),
	"mediumturquoise":      vaxis.HexColor(0x48D1CC),
	"mediumvioletred":      vaxis.HexColor(0xC71585),
	"midnightblue":         vaxis.HexColor(0x191970),
	"mintcream":            vaxis.HexColor(0xF5FFFA),
	"mistyrose":            vaxis.HexColor(0xFFE4E1),
	"moccasin":             vaxis.HexColor(0xFFE4B5),
	"navajowhite":          vaxis.HexColor(0xFFDEAD),
	"oldlace":              vaxis.HexColor(0xFDF5E6),
	"olivedrab":            vaxis.HexColor(0x6B8E23),
	"orange":               vaxis.HexColor(0xFFA500),
	"orangered":            vaxis.HexColor(0xFF4500),
	"orchid":               vaxis.HexColor(0xDA70D6),
	"palegoldenrod":        vaxis.HexColor(0xEEE8AA),
	"palegreen":            vaxis.HexColor(0x98FB98),
	"paleturquoise":        vaxis.HexColor(0xAFEEEE),
	"palevioletred":        vaxis.HexColor(0xDB7093),
	"papayawhip":           vaxis.HexColor(0xFFEFD5),
	"peachpuff":            vaxis.HexColor(0xFFDAB9),
	"peru":                 vaxis.HexColor(0xCD853F),
	"pink":                 vaxis.HexColor(0xFFC0CB),
	"plum":                 vaxis.HexColor(0xDDA0DD),
	"powderblue":           vaxis.HexColor(0xB0E0E6),
	"rebeccapurple":        vaxis.HexColor(0x663399),
	"rosybrown":            vaxis.HexColor(0xBC8F8F),
	"royalblue":            vaxis.HexColor(0x4169E1),
	"saddlebrown":          vaxis.HexColor(0x8B4513),
	"salmon":               vaxis.HexColor(0xFA8072),
	"sandybrown":           vaxis.HexColor(0xF4A460),
	"seagreen":             vaxis.HexColor(0x2E8B57),
	"seashell":             vaxis.HexColor(0xFFF5EE),
	"sienna":               vaxis.HexColor(0xA0522D),
	"skyblue":              vaxis.HexColor(0x87CEEB),
	"slateblue":            vaxis.HexColor(0x6A5ACD),
	"slategray":            vaxis.HexColor(0x708090),
	"snow":                 vaxis.HexColor(0xFFFAFA),
	"springgreen":          vaxis.HexColor(0x00FF7F),
	"steelblue":            vaxis.HexColor(0x4682B4),
	"tan":                  vaxis.HexColor(0xD2B48C),
	"thistle":              vaxis.HexColor(0xD8BFD8),
	"tomato":               vaxis.HexColor(0xFF6347),
	"turquoise":            vaxis.HexColor(0x40E0D0),
	"violet":               vaxis.HexColor(0xEE82EE),
	"wheat":                vaxis.HexColor(0xF5DEB3),
	"whitesmoke":           vaxis.HexColor(0xF5F5F5),
	"yellowgreen":          vaxis.HexColor(0x9ACD32),
}
