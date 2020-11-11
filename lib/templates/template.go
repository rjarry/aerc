package templates

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"
	"time"

	"git.sr.ht/~sircmpwn/aerc/models"
	"github.com/mitchellh/go-homedir"
)

var version string

//SetVersion initializes the aerc version displayed in template functions
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

func TestTemplateData() TemplateData {
	defaults := map[string]string{
		"To":      "John Doe <john@example.com>",
		"Cc":      "Josh Doe <josh@example.com>",
		"From":    "Jane Smith <jane@example.com>",
		"Subject": "This is only a test",
	}

	original := models.OriginalMail{
		Date:     time.Now(),
		From:     "John Doe <john@example.com>",
		Text:     "This is only a test text",
		MIMEType: "text/plain",
	}

	return ParseTemplateData(defaults, original)
}

func ParseTemplateData(defaults map[string]string, original models.OriginalMail) TemplateData {
	td := TemplateData{
		To:               parseAddressList(defaults["To"]),
		Cc:               parseAddressList(defaults["Cc"]),
		Bcc:              parseAddressList(defaults["Bcc"]),
		From:             parseAddressList(defaults["From"]),
		Date:             time.Now(),
		Subject:          defaults["Subject"],
		OriginalText:     original.Text,
		OriginalFrom:     parseAddressList(original.From),
		OriginalDate:     original.Date,
		OriginalMIMEType: original.MIMEType,
	}
	return td
}

func parseAddressList(list string) []*mail.Address {
	addrs, err := mail.ParseAddressList(list)
	if err != nil {
		return nil
	}

	return addrs
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

func ParseTemplate(templateText string, data interface{}) ([]byte, error) {
	emailTemplate, err :=
		template.New("email_template").Funcs(templateFuncs).Parse(templateText)
	if err != nil {
		return nil, err
	}

	var outString bytes.Buffer
	if err := emailTemplate.Execute(&outString, data); err != nil {
		return nil, err
	}
	return outString.Bytes(), nil
}
