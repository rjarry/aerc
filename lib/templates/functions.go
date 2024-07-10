package templates

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/emersion/go-message/mail"
)

var version string

// SetVersion initializes the aerc version displayed in template functions
func SetVersion(v string) {
	version = v
}

// wrap allows to chain wrapText
func wrap(lineWidth int, text string) string {
	return wrapText(text, lineWidth)
}

func wrapLine(text string, lineWidth int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var wrapped strings.Builder
	wrapped.WriteString(words[0])
	spaceLeft := lineWidth - wrapped.Len()
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped.WriteRune('\n')
			wrapped.WriteString(word)
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped.WriteRune(' ')
			wrapped.WriteString(word)
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped.String()
}

func wrapText(text string, lineWidth int) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	lines := strings.Split(text, "\n")
	var wrapped strings.Builder

	for _, line := range lines {
		switch {
		case line == "":
			// deliberately left blank
		case line[0] == '>':
			// leave quoted text alone
			wrapped.WriteString(line)
		default:
			wrapped.WriteString(wrapLine(line, lineWidth))
		}
		wrapped.WriteRune('\n')
	}
	return wrapped.String()
}

// quote prepends "> " in front of every line in text
func quote(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimRight(text, "\n")
	lines := strings.Split(text, "\n")
	var quoted strings.Builder
	for _, line := range lines {
		if line == "" {
			quoted.WriteString(">\n")
			continue
		}
		quoted.WriteString("> ")
		quoted.WriteString(line)
		quoted.WriteRune('\n')
	}

	return quoted.String()
}

// cmd allow to parse reply by shell command
// text have to be passed by cmd param
// if there is error, original string is returned
func cmd(cmd, text string) string {
	var out bytes.Buffer
	c := exec.Command("sh", "-c", cmd)
	c.Stdin = strings.NewReader(text)
	c.Stdout = &out
	err := c.Run()
	if err != nil {
		return text
	}
	return out.String()
}

func toLocal(t time.Time) time.Time {
	return time.Time.In(t, time.Local)
}

func rearrangeNameWithComma(name string) string {
	parts := strings.SplitN(name, ",", 3)
	if len(parts) == 2 {
		return fmt.Sprintf("%s %s", strings.TrimSpace(parts[1]), strings.TrimSpace(parts[0]))
	}
	return name
}

func names(addresses []*mail.Address) []string {
	n := make([]string, len(addresses))
	for i, addr := range addresses {
		name := rearrangeNameWithComma(addr.Name)
		if name == "" {
			parts := strings.SplitN(addr.Address, "@", 2)
			name = parts[0]
		}
		n[i] = name
	}
	return n
}

func firstnames(addresses []*mail.Address) []string {
	n := make([]string, len(addresses))
	for i, addr := range addresses {
		var name string
		if addr.Name == "" {
			parts := strings.SplitN(addr.Address, "@", 2)
			parts = strings.SplitN(parts[0], ".", 2)
			name = parts[0]
		} else {
			name = rearrangeNameWithComma(addr.Name)
			name = strings.SplitN(name, " ", 2)[0] // split by spaces and get the first word
			name = strings.SplitN(name, ",", 2)[0] // split by commas and get the first word
		}
		n[i] = name
	}
	return n
}

func initials(addresses []*mail.Address) []string {
	n := names(addresses)
	ret := make([]string, len(addresses))
	for i, name := range n {
		split := strings.Split(name, " ")
		initial := ""
		for _, s := range split {
			initial += string([]rune(s)[0:1])
		}
		ret[i] = initial
	}
	return ret
}

func emails(addresses []*mail.Address) []string {
	e := make([]string, len(addresses))
	for i, addr := range addresses {
		e[i] = addr.Address
	}
	return e
}

func mboxes(addresses []*mail.Address) []string {
	e := make([]string, len(addresses))
	for i, addr := range addresses {
		parts := strings.SplitN(addr.Address, "@", 2)
		e[i] = parts[0]
	}
	return e
}

func shortmboxes(addresses []*mail.Address) []string {
	e := make([]string, len(addresses))
	for i, addr := range addresses {
		parts := strings.SplitN(addr.Address, "@", 2)
		parts = strings.SplitN(parts[0], ".", 2)
		e[i] = parts[0]
	}
	return e
}

func persons(addresses []*mail.Address) []string {
	e := make([]string, len(addresses))
	for i, addr := range addresses {
		e[i] = format.AddressForHumans(addr)
	}
	return e
}

var units = []string{"K", "M", "G", "T"}

func humanReadable(value int) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	if value < 1000 {
		return fmt.Sprintf("%s%d", sign, value)
	}
	val := float64(value)
	unit := ""
	for i := 0; val >= 1000 && i < len(units); i++ {
		unit = units[i]
		val /= 1000.0
	}
	if val < 100.0 {
		return fmt.Sprintf("%s%.1f%s", sign, val, unit)
	}
	return fmt.Sprintf("%s%.0f%s", sign, val, unit)
}

func cwd() string {
	path, err := os.Getwd()
	if err != nil {
		return err.Error()
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err.Error()
	}
	if strings.HasPrefix(path, home) {
		path = strings.Replace(path, home, "~", 1)
	}
	return path
}

func join(sep string, elems []string) string {
	return strings.Join(elems, sep)
}

func split(sep string, s string) []string {
	return strings.Split(s, sep)
}

// removes a signature from the piped in message
func trimSignature(message string) string {
	var res strings.Builder

	input := bufio.NewScanner(strings.NewReader(message))

	for input.Scan() {
		line := input.Text()
		if line == "-- " {
			break
		}
		res.WriteString(line)
		res.WriteRune('\n')
	}
	return res.String()
}

func compactDir(path string) string {
	return format.CompactPath(path, os.PathSeparator)
}

type (
	Case    struct{ expr, value string }
	Default struct{ value string }
	Exclude struct{ expr string }
)

func (c *Case) Matches(s string) bool    { return parse.MatchCache(s, c.expr) }
func (c *Case) Value() string            { return c.value }
func (c *Case) Skip() bool               { return false }
func (d *Default) Matches(s string) bool { return true }
func (d *Default) Value() string         { return d.value }
func (d *Default) Skip() bool            { return false }
func (e *Exclude) Matches(s string) bool { return parse.MatchCache(s, e.expr) }
func (e *Exclude) Value() string         { return "" }
func (e *Exclude) Skip() bool            { return true }

func switch_(value string, cases ...models.Case) string {
	for _, c := range cases {
		if c.Matches(value) {
			return c.Value()
		}
	}
	return ""
}

func case_(expr, value string) models.Case {
	return &Case{expr: expr, value: value}
}

func default_(value string) models.Case {
	return &Default{value: value}
}

func exclude(expr string) models.Case {
	return &Exclude{expr: expr}
}

func map_(elements []string, cases ...models.Case) []string {
	mapped := make([]string, 0, len(elements))
top:
	for _, e := range elements {
		for _, c := range cases {
			if c.Matches(e) {
				if c.Skip() {
					continue top
				}
				e = c.Value()
				break
			}
		}
		mapped = append(mapped, e)
	}
	return mapped
}

func replace(pattern, subst, value string) string {
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(value, subst)
}

func contains(substring, s string) bool {
	return strings.Contains(s, substring)
}

func hasPrefix(prefix, s string) bool {
	return strings.HasPrefix(s, prefix)
}

func head(n uint, s string) string {
	r := []rune(s)
	length := uint(len(r))
	if length >= n {
		return string(r[:n])
	}
	return s
}

func tail(n uint, s string) string {
	r := []rune(s)
	length := uint(len(r))
	if length >= n {
		return string(r[length-n:])
	}
	return s
}

var templateFuncs = template.FuncMap{
	"quote":         quote,
	"wrapText":      wrapText,
	"wrap":          wrap,
	"now":           time.Now,
	"dateFormat":    time.Time.Format,
	"toLocal":       toLocal,
	"exec":          cmd,
	"version":       func() string { return version },
	"names":         names,
	"firstnames":    firstnames,
	"initials":      initials,
	"emails":        emails,
	"mboxes":        mboxes,
	"shortmboxes":   shortmboxes,
	"persons":       persons,
	"humanReadable": humanReadable,
	"cwd":           cwd,
	"join":          join,
	"split":         split,
	"trimSignature": trimSignature,
	"compactDir":    compactDir,
	"match":         parse.MatchCache,
	"switch":        switch_,
	"case":          case_,
	"default":       default_,
	"map":           map_,
	"exclude":       exclude,
	"contains":      contains,
	"hasPrefix":     hasPrefix,
	"toLower":       strings.ToLower,
	"toUpper":       strings.ToUpper,
	"replace":       replace,
	"head":          head,
	"tail":          tail,
}
