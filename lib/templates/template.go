package templates

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/emersion/go-message/mail"

	"git.sr.ht/~rjarry/aerc/models"
	"github.com/mitchellh/go-homedir"
)

var version string

// SetVersion initializes the aerc version displayed in template functions
func SetVersion(v string) {
	version = v
}

type TemplateData struct {
	To      []*mail.Address
	Cc      []*mail.Address
	Bcc     []*mail.Address
	From    []*mail.Address
	Date    time.Time
	Subject string
	// Only available when replying with a quote
	OriginalText     string
	OriginalFrom     []*mail.Address
	OriginalDate     time.Time
	OriginalMIMEType string
}

func ParseTemplateData(h *mail.Header, original models.OriginalMail) TemplateData {
	// we ignore errors as this shouldn't fail the sending / replying even if
	// something is wrong with the message we reply to
	to, _ := h.AddressList("to")
	cc, _ := h.AddressList("cc")
	bcc, _ := h.AddressList("bcc")
	from, _ := h.AddressList("from")
	subject, err := h.Text("subject")
	if err != nil {
		subject = h.Get("subject")
	}

	td := TemplateData{
		To:               to,
		Cc:               cc,
		Bcc:              bcc,
		From:             from,
		Date:             time.Now(),
		Subject:          subject,
		OriginalText:     original.Text,
		OriginalDate:     original.Date,
		OriginalMIMEType: original.MIMEType,
	}
	if original.RFC822Headers != nil {
		origFrom, _ := original.RFC822Headers.AddressList("from")
		td.OriginalFrom = origFrom
	}
	return td
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

func findTemplate(templateName string, templateDirs []string) (string, error) {
	for _, dir := range templateDirs {
		templateFile, err := homedir.Expand(path.Join(dir, templateName))
		if err != nil {
			return "", err
		}

		if _, err := os.Stat(templateFile); os.IsNotExist(err) {
			continue
		}
		return templateFile, nil
	}

	return "", fmt.Errorf(
		"Can't find template %q in any of %v ", templateName, templateDirs)
}

// DummyData provides dummy data to test template validity
func DummyData() interface{} {
	from := &mail.Address{
		Name:    "John Doe",
		Address: "john@example.com",
	}
	to := &mail.Address{
		Name:    "Alice Doe",
		Address: "alice@example.com",
	}
	h := &mail.Header{}
	h.SetAddressList("from", []*mail.Address{from})
	h.SetAddressList("to", []*mail.Address{to})

	oh := &mail.Header{}
	oh.SetAddressList("from", []*mail.Address{to})
	oh.SetAddressList("to", []*mail.Address{from})

	original := models.OriginalMail{
		Date:          time.Now(),
		From:          from.String(),
		Text:          "This is only a test text",
		MIMEType:      "text/plain",
		RFC822Headers: oh,
	}
	return ParseTemplateData(h, original)
}

func ParseTemplateFromFile(templateName string, templateDirs []string, data interface{}) (io.Reader, error) {
	templateFile, err := findTemplate(templateName, templateDirs)
	if err != nil {
		return nil, err
	}
	emailTemplate, err := template.New(templateName).
		Funcs(templateFuncs).ParseFiles(templateFile)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	if err := emailTemplate.Execute(&body, data); err != nil {
		return nil, err
	}
	return &body, nil
}

func CheckTemplate(templateName string, templateDirs []string) error {
	if templateName != "" {
		_, err := ParseTemplateFromFile(templateName, templateDirs, DummyData())
		return err
	}
	return nil
}
