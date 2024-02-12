package parse

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rockorager/vaxis"
	"github.com/mattn/go-runewidth"
)

var AnsiReg = regexp.MustCompile("\x1B\\[[0-?]*[ -/]*[@-~]")

const (
	setfgbgrgb    = "\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dm"
	setfgrgb      = "\x1b[38;2;%d;%d;%dm"
	setbgrgb      = "\x1b[48;2;%d;%d;%dm"
	setfgbg       = "\x1b[38;5;%d;48;5;%dm"
	setfg         = "\x1b[38;5;%dm"
	setbg         = "\x1b[48;5;%dm"
	attrOff       = "\x1B[m"
	bold          = "\x1B[1m"
	dim           = "\x1B[2m"
	italic        = "\x1B[3m"
	underline     = "\x1B[4m"
	blink         = "\x1B[5m"
	reverse       = "\x1B[7m"
	strikethrough = "\x1B[9m"
)

// StripAnsi strips ansi escape codes from the reader
func StripAnsi(r io.Reader) io.Reader {
	buf := bytes.NewBuffer(nil)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, 1024*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		line = AnsiReg.ReplaceAll(line, []byte(""))
		_, err := buf.Write(line)
		if err != nil {
			log.Warnf("failed write ", err)
		}
		_, err = buf.Write([]byte("\n"))
		if err != nil {
			log.Warnf("failed write ", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read line: %v\n", err)
	}
	return buf
}

// StyledRune is a rune and it's associated style. The rune has already been
// measured using go-runewidth
type StyledRune struct {
	Value rune
	Width int
	Style vaxis.Style
}

// RuneBuffer is a buffer of runes styled with vaxis.Style objects
type RuneBuffer struct {
	buf []*StyledRune
}

// Returns the internal slice of styled runes
func (rb *RuneBuffer) Runes() []*StyledRune {
	return rb.buf
}

// Write writes a rune and it's associated style to the RuneBuffer
func (rb *RuneBuffer) Write(r rune, style vaxis.Style) {
	w := runewidth.RuneWidth(r)
	rb.buf = append(rb.buf, &StyledRune{r, w, style})
}

// Prepend inserts the rune at the beginning of the rune buffer
func (rb *RuneBuffer) PadLeft(width int, r rune, style vaxis.Style) {
	w := rb.Len()
	if w >= width {
		return
	}
	w = width - w
	for w > 0 {
		ww := runewidth.RuneWidth(r)
		w -= ww
		rb.buf = append([]*StyledRune{{r, ww, style}}, rb.buf...)
	}
}

func (rb *RuneBuffer) PadRight(width int, r rune, style vaxis.Style) {
	w := rb.Len()
	if w >= width {
		return
	}
	w = width - w
	for w > 0 {
		ww := runewidth.RuneWidth(r)
		w -= ww
		rb.buf = append(rb.buf, &StyledRune{r, ww, style})
	}
}

// String outputs a styled-string using TERM=xterm-256color
func (rb *RuneBuffer) String() string {
	return rb.string(0, false, 0)
}

// string returns a string no longer than n runes. If 'left' is true, the left
// side of the text is truncated. Pass 0 to return the full string
func (rb *RuneBuffer) string(n int, left bool, char rune) string {
	var (
		s        = bytes.NewBuffer(nil)
		style    = vaxis.Style{}
		hasStyle = false
		// w will track the length we have written, or would have
		// written in the case of left truncate
		w      = 0
		offset = 0
	)

	if left {
		offset = rb.Len() - n
	}

	for _, r := range rb.buf {
		if style != r.Style {
			hasStyle = true
			style = r.Style
			s.WriteString(attrOff)
			// fg, bg, attrs := style.Decompose()
			fg := style.Foreground.Params()
			switch len(fg) {
			case 0:
				// default
			case 1:
				// indexed
				fmt.Fprintf(s, setfg, fg[0])
			case 3:
				// rgb
				fmt.Fprintf(s, setfgrgb, fg[0], fg[1], fg[2])
			}

			bg := style.Background.Params()
			switch len(bg) {
			case 0:
				// default
			case 1:
				// indexed
				fmt.Fprintf(s, setbg, bg[0])
			case 3:
				// rgb
				fmt.Fprintf(s, setbgrgb, bg[0], bg[1], bg[2])
			}

			attrs := style.Attribute

			if attrs&vaxis.AttrBold != 0 {
				s.WriteString(bold)
			}
			if attrs&vaxis.AttrReverse != 0 {
				s.WriteString(reverse)
			}
			if attrs&vaxis.AttrBlink != 0 {
				s.WriteString(blink)
			}
			if attrs&vaxis.AttrDim != 0 {
				s.WriteString(dim)
			}
			if attrs&vaxis.AttrItalic != 0 {
				s.WriteString(italic)
			}
			if attrs&vaxis.AttrStrikethrough != 0 {
				s.WriteString(strikethrough)
			}

			if style.UnderlineStyle != vaxis.UnderlineOff {
				s.WriteString(underline)
			}
		}

		w += r.Width
		if left && w <= offset {
			if w == offset && char != 0 {
				s.WriteRune(char)
			}
			continue
		}
		s.WriteRune(r.Value)
		if n != 0 && !left && w == n {
			if char != 0 {
				s.WriteRune(char)
			}
			break
		}
	}
	if hasStyle {
		s.WriteString(attrOff)
	}
	return s.String()
}

// Len is the length of the string, without ansi sequences
func (rb *RuneBuffer) Len() int {
	l := 0
	for _, r := range rb.buf {
		l += r.Width
	}
	return l
}

// Truncates to a width of n, optionally append a character to the string.
// Appending via Truncate allows the character to retain the same style as the
// string at the truncated location
func (rb *RuneBuffer) Truncate(n int, char rune) string {
	return rb.string(n, false, char)
}

// Truncates a width of n off the beginning of the string, optionally append a
// character to the string. Appending via Truncate allows the character to
// retain the same style as the string at the truncated location
func (rb *RuneBuffer) TruncateHead(n int, char rune) string {
	return rb.string(n, true, char)
}

// Applies a style to the buffer. Any currently applied styles will not be
// overwritten
func (rb *RuneBuffer) ApplyStyle(style vaxis.Style) {
	d := vaxis.Style{}
	for _, sr := range rb.buf {
		if sr.Style == d {
			sr.Style = style
		}
	}
}

// ApplyAttrs applies the style, and if another style is present ORs the
// attributes
func (rb *RuneBuffer) ApplyAttrs(style vaxis.Style) {
	for _, sr := range rb.buf {
		if style.Foreground != 0 {
			sr.Style.Foreground = style.Foreground
		}
		if style.Background != 0 {
			sr.Style.Background = style.Background
		}
		sr.Style.Attribute |= style.Attribute
		if style.UnderlineColor != 0 {
			sr.Style.UnderlineColor = style.UnderlineColor
		}
		if style.UnderlineStyle != vaxis.UnderlineOff {
			sr.Style.UnderlineStyle = style.UnderlineStyle
		}
	}
}

// Applies a style to a string. Any currently applied styles will not be overwritten
func ApplyStyle(style vaxis.Style, str string) string {
	rb := ParseANSI(str)
	d := vaxis.Style{}
	for _, sr := range rb.buf {
		if sr.Style == d {
			sr.Style = style
		}
	}
	return rb.String()
}

// Parses a styled string into a RuneBuffer
func ParseANSI(s string) *RuneBuffer {
	p := &parser{
		buf:      &RuneBuffer{},
		curStyle: vaxis.Style{},
	}
	rdr := strings.NewReader(s)

	for {
		r, _, err := rdr.ReadRune()
		if err == io.EOF {
			break
		}
		switch r {
		case 0x1b:
			p.handleSeq(rdr)
		default:
			p.buf.Write(r, p.curStyle)
		}
	}
	return p.buf
}

// A parser parses a string into a RuneBuffer
type parser struct {
	buf      *RuneBuffer
	curStyle vaxis.Style
}

func (p *parser) handleSeq(rdr io.RuneReader) {
	r, _, err := rdr.ReadRune()
	if errors.Is(err, io.EOF) {
		return
	}
	switch r {
	case '[': // CSI
		p.handleCSI(rdr)
	case ']': // OSC
	case '(': // Designate G0 charset
		p.swallow(rdr, 1)
	}
}

func (p *parser) handleCSI(rdr io.RuneReader) {
	var (
		params []int
		param  []rune
		hasErr bool
		er     error
	)
outer:
	for {
		r, _, err := rdr.ReadRune()
		if errors.Is(err, io.EOF) {
			return
		}
		switch {
		case r >= 0x30 && r <= 0x39:
			param = append(param, r)
		case r == ':' || r == ';':
			var ps int
			if len(param) > 0 {
				ps, er = strconv.Atoi(string(param))
				if er != nil {
					hasErr = true
					continue
				}
			}
			params = append(params, ps)
			param = []rune{}
		case r == 'm':
			var ps int
			if len(param) > 0 {
				ps, er = strconv.Atoi(string(param))
				if er != nil {
					hasErr = true
					continue
				}
			}
			params = append(params, ps)
			break outer
		}
	}
	if hasErr {
		// leave the cursor unchanged
		return
	}
	for i := 0; i < len(params); i++ {
		param := params[i]
		switch param {
		case 0:
			p.curStyle = vaxis.Style{}
		case 1:
			p.curStyle.Attribute |= vaxis.AttrBold
		case 2:
			p.curStyle.Attribute |= vaxis.AttrDim
		case 3:
			p.curStyle.Attribute |= vaxis.AttrItalic
		case 4:
			p.curStyle.UnderlineStyle = vaxis.UnderlineSingle
		case 5:
			p.curStyle.Attribute |= vaxis.AttrBlink
		case 6:
			// rapid blink, not supported by vaxis. fallback to slow
			// blink
			p.curStyle.Attribute |= vaxis.AttrBlink
		case 7:
			p.curStyle.Attribute |= vaxis.AttrReverse
		case 8:
			// Hidden. not supported by vaxis
		case 9:
			p.curStyle.Attribute |= vaxis.AttrStrikethrough
		case 21:
			p.curStyle.Attribute &^= vaxis.AttrBold
		case 22:
			p.curStyle.Attribute &^= vaxis.AttrDim
		case 23:
			p.curStyle.Attribute &^= vaxis.AttrItalic
		case 24:
			p.curStyle.UnderlineStyle = vaxis.UnderlineOff
		case 25:
			p.curStyle.Attribute &^= vaxis.AttrBlink
		case 26:
			// rapid blink, not supported by vaxis. fallback to slow
			// blink
			p.curStyle.Attribute &^= vaxis.AttrBlink
		case 27:
			p.curStyle.Attribute &^= vaxis.AttrReverse
		case 28:
			// Hidden. unsupported by vaxis
		case 29:
			p.curStyle.Attribute &^= vaxis.AttrStrikethrough
		case 30, 31, 32, 33, 34, 35, 36, 37:
			p.curStyle.Foreground = vaxis.IndexColor(uint8(param - 30))
		case 38:
			if i+2 < len(params) && params[i+1] == 5 {
				p.curStyle.Foreground = vaxis.IndexColor(uint8(params[i+2]))
				i += 2
			}
			if i+4 < len(params) && params[i+1] == 2 {
				switch len(params) {
				case 6:
					r := uint8(params[i+3])
					g := uint8(params[i+4])
					b := uint8(params[i+5])
					p.curStyle.Foreground = vaxis.RGBColor(r, g, b)
					i += 5
				default:
					r := uint8(params[i+2])
					g := uint8(params[i+3])
					b := uint8(params[i+4])
					p.curStyle.Foreground = vaxis.RGBColor(r, g, b)
					i += 4
				}
			}
		case 40, 41, 42, 43, 44, 45, 46, 47:
			p.curStyle.Background = vaxis.IndexColor(uint8(param - 40))
		case 48:
			if i+2 < len(params) && params[i+1] == 5 {
				p.curStyle.Background = vaxis.IndexColor(uint8(params[i+2]))
				i += 2
			}
			if i+4 < len(params) && params[i+1] == 2 {
				switch len(params) {
				case 6:
					r := uint8(params[i+3])
					g := uint8(params[i+4])
					b := uint8(params[i+5])
					p.curStyle.Background = vaxis.RGBColor(r, g, b)
					i += 5
				default:
					r := uint8(params[i+2])
					g := uint8(params[i+3])
					b := uint8(params[i+4])
					p.curStyle.Background = vaxis.RGBColor(r, g, b)
					i += 4
				}
			}
		case 90, 91, 92, 93, 94, 95, 96, 97:
			p.curStyle.Foreground = vaxis.IndexColor(uint8(param - 90 + 8))
		case 100, 101, 102, 103, 104, 105, 106, 107:
			p.curStyle.Background = vaxis.IndexColor(uint8(param - 100 + 8))
		}
	}
}

func (p *parser) swallow(rdr io.RuneReader, n int) {
	for i := 0; i < n; i++ {
		rdr.ReadRune() //nolint:errcheck // we are throwing these reads away
	}
}
