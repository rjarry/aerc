package templates

import (
	"bytes"
	"os/exec"
	"strings"
	"text/template"
	"time"
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

var templateFuncs = template.FuncMap{
	"quote":      quote,
	"wrapText":   wrapText,
	"wrap":       wrap,
	"dateFormat": time.Time.Format,
	"toLocal":    toLocal,
	"exec":       cmd,
	"version":    func() string { return version },
}
