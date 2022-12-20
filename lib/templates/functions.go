package templates

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/format"
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

func names(addresses []*mail.Address) []string {
	n := make([]string, len(addresses))
	for i, addr := range addresses {
		name := addr.Name
		if name == "" {
			name = addr.Address
		}
		n[i] = name
	}
	return n
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
		parts := strings.SplitN(addr.Address, "@", 1)
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

func humanReadable(value uint32) string {
	if value < 1000 {
		return fmt.Sprintf("%d", value)
	}
	val := float64(value)
	unit := ""
	for i := 0; val >= 1000 && i < len(units); i++ {
		unit = units[i]
		val /= 1000.0
	}
	if val < 100.0 {
		return fmt.Sprintf("%.1f%s", val, unit)
	}
	return fmt.Sprintf("%.0f%s", val, unit)
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

var templateFuncs = template.FuncMap{
	"quote":         quote,
	"wrapText":      wrapText,
	"wrap":          wrap,
	"dateFormat":    time.Time.Format,
	"toLocal":       toLocal,
	"exec":          cmd,
	"version":       func() string { return version },
	"names":         names,
	"emails":        emails,
	"mboxes":        mboxes,
	"persons":       persons,
	"humanReadable": humanReadable,
	"cwd":           cwd,
	"join":          join,
}
