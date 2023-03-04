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
	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/terminfo"
	"github.com/mattn/go-runewidth"
)

var AnsiReg = regexp.MustCompile("\x1B\\[[0-?]*[ -/]*[@-~]")

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
	Style tcell.Style
}

// RuneBuffer is a buffer of runes styled with tcell.Style objects
type RuneBuffer struct {
	buf []*StyledRune
}

// Returns the internal slice of styled runes
func (rb *RuneBuffer) Runes() []*StyledRune {
	return rb.buf
}

// Write writes a rune and it's associated style to the RuneBuffer
func (rb *RuneBuffer) Write(r rune, style tcell.Style) {
	w := runewidth.RuneWidth(r)
	rb.buf = append(rb.buf, &StyledRune{r, w, style})
}

// Prepend inserts the rune at the beginning of the rune buffer
func (rb *RuneBuffer) PadLeft(width int, r rune, style tcell.Style) {
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

func (rb *RuneBuffer) PadRight(width int, r rune, style tcell.Style) {
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
	// Use xterm-256color to generate the string. Ultimately all output will
	// be re-parsed as 'xterm-256color' and tcell will handle the final
	// output sequences based on the user's TERM
	ti, err := terminfo.LookupTerminfo("xterm-256color")
	if err != nil {
		// Who knows what happened
		return ""
	}
	var (
		s        = strings.Builder{}
		style    = tcell.StyleDefault
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
			s.WriteString(ti.AttrOff)
			fg, bg, attrs := style.Decompose()

			switch {
			case fg.IsRGB() && bg.IsRGB() && ti.SetFgBgRGB != "":
				fr, fg, fb := fg.RGB()
				br, bg, bb := bg.RGB()
				s.WriteString(ti.TParm(
					ti.SetFgBgRGB,
					int(fr),
					int(fg),
					int(fb),
					int(br),
					int(bg),
					int(bb),
				))
			case fg.IsRGB() && ti.SetFgRGB != "":
				// RGB
				r, g, b := fg.RGB()
				s.WriteString(ti.TParm(ti.SetFgRGB, int(r), int(g), int(b)))
			case bg.IsRGB() && ti.SetBgRGB != "":
				// RGB
				r, g, b := bg.RGB()
				s.WriteString(ti.TParm(ti.SetBgRGB, int(r), int(g), int(b)))

				// Indexed
			case fg.Valid() && bg.Valid() && ti.SetFgBg != "":
				s.WriteString(ti.TParm(ti.SetFgBg, int(fg&0xff), int(bg&0xff)))
			case fg.Valid() && ti.SetFg != "":
				s.WriteString(ti.TParm(ti.SetFg, int(fg&0xff)))
			case bg.Valid() && ti.SetBg != "":
				s.WriteString(ti.TParm(ti.SetBg, int(bg&0xff)))
			}

			if attrs&tcell.AttrBold != 0 {
				s.WriteString(ti.Bold)
			}
			if attrs&tcell.AttrUnderline != 0 {
				s.WriteString(ti.Underline)
			}
			if attrs&tcell.AttrReverse != 0 {
				s.WriteString(ti.Reverse)
			}
			if attrs&tcell.AttrBlink != 0 {
				s.WriteString(ti.Blink)
			}
			if attrs&tcell.AttrDim != 0 {
				s.WriteString(ti.Dim)
			}
			if attrs&tcell.AttrItalic != 0 {
				s.WriteString(ti.Italic)
			}
			if attrs&tcell.AttrStrikeThrough != 0 {
				s.WriteString(ti.StrikeThrough)
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
		s.WriteString(ti.AttrOff)
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
func (rb *RuneBuffer) ApplyStyle(style tcell.Style) {
	for _, sr := range rb.buf {
		if sr.Style == tcell.StyleDefault {
			sr.Style = style
		}
	}
}

// ApplyAttrs applies the style, and if another style is present ORs the
// attributes
func (rb *RuneBuffer) ApplyAttrs(style tcell.Style) {
	for _, sr := range rb.buf {
		if sr.Style == tcell.StyleDefault {
			sr.Style = style
			continue
		}
		_, _, srAttrs := sr.Style.Decompose()
		_, _, attrs := style.Decompose()
		sr.Style = sr.Style.Attributes(srAttrs | attrs)
	}
}

// Applies a style to a string. Any currently applied styles will not be overwritten
func ApplyStyle(style tcell.Style, str string) string {
	rb := ParseANSI(str)
	for _, sr := range rb.buf {
		if sr.Style == tcell.StyleDefault {
			sr.Style = style
		}
	}
	return rb.String()
}

// Parses a styled string into a RuneBuffer
func ParseANSI(s string) *RuneBuffer {
	p := &parser{
		buf:      &RuneBuffer{},
		curStyle: tcell.StyleDefault,
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
	curStyle tcell.Style
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
			p.curStyle = tcell.StyleDefault
		case 1:
			p.curStyle = p.curStyle.Bold(true)
		case 2:
			p.curStyle = p.curStyle.Dim(true)
		case 3:
			p.curStyle = p.curStyle.Italic(true)
		case 4:
			p.curStyle = p.curStyle.Underline(true)
		case 5:
			p.curStyle = p.curStyle.Blink(true)
		case 6:
			// rapid blink, not supported by tcell. fallback to slow
			// blink
			p.curStyle = p.curStyle.Blink(true)
		case 7:
			p.curStyle = p.curStyle.Reverse(true)
		case 8:
			// Hidden. not supported by tcell
		case 9:
			p.curStyle = p.curStyle.StrikeThrough(true)
		case 21:
			p.curStyle = p.curStyle.Bold(false)
		case 22:
			p.curStyle = p.curStyle.Dim(false)
		case 23:
			p.curStyle = p.curStyle.Italic(false)
		case 24:
			p.curStyle = p.curStyle.Underline(false)
		case 25:
			p.curStyle = p.curStyle.Blink(false)
		case 26:
			// rapid blink, not supported by tcell. fallback to slow
			// blink
			p.curStyle = p.curStyle.Blink(false)
		case 27:
			p.curStyle = p.curStyle.Reverse(false)
		case 28:
			// Hidden. unsupported by tcell
		case 29:
			p.curStyle = p.curStyle.StrikeThrough(false)
		case 30, 31, 32, 33, 34, 35, 36, 37:
			p.curStyle = p.curStyle.Foreground(tcell.PaletteColor(param - 30))
		case 38:
			if i+2 < len(params) && params[i+1] == 5 {
				p.curStyle = p.curStyle.Foreground(tcell.PaletteColor(params[i+2]))
				i += 2
			}
			if i+4 < len(params) && params[i+1] == 2 {
				switch len(params) {
				case 6:
					r := int32(params[i+3])
					g := int32(params[i+4])
					b := int32(params[i+5])
					p.curStyle = p.curStyle.Foreground(tcell.NewRGBColor(r, g, b))
					i += 5
				default:
					r := int32(params[i+2])
					g := int32(params[i+3])
					b := int32(params[i+4])
					p.curStyle = p.curStyle.Foreground(tcell.NewRGBColor(r, g, b))
					i += 4
				}
			}
		case 40, 41, 42, 43, 44, 45, 46, 47:
			p.curStyle = p.curStyle.Background(tcell.PaletteColor(param - 40))
		case 48:
			if i+2 < len(params) && params[i+1] == 5 {
				p.curStyle = p.curStyle.Background(tcell.PaletteColor(params[i+2]))
				i += 2
			}
			if i+4 < len(params) && params[i+1] == 2 {
				switch len(params) {
				case 6:
					r := int32(params[i+3])
					g := int32(params[i+4])
					b := int32(params[i+5])
					p.curStyle = p.curStyle.Background(tcell.NewRGBColor(r, g, b))
					i += 5
				default:
					r := int32(params[i+2])
					g := int32(params[i+3])
					b := int32(params[i+4])
					p.curStyle = p.curStyle.Background(tcell.NewRGBColor(r, g, b))
					i += 4
				}
			}
		case 90, 91, 92, 93, 94, 95, 96, 97:
			p.curStyle = p.curStyle.Foreground(tcell.PaletteColor(param - 90 + 8))
		case 100, 101, 102, 103, 104, 105, 106, 107:
			p.curStyle = p.curStyle.Background(tcell.PaletteColor(param - 100 + 8))
		}
	}
}

func (p *parser) swallow(rdr io.RuneReader, n int) {
	for i := 0; i < n; i++ {
		rdr.ReadRune() //nolint:errcheck // we are throwing these reads away
	}
}
