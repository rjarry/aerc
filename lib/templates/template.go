package templates

import (
	"bytes"
	"errors"
	"net/mail"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/mitchellh/go-homedir"
)

type TemplateData struct {
	To      []*mail.Address
	Cc      []*mail.Address
	Bcc     []*mail.Address
	From    []*mail.Address
	Date    time.Time
	Subject string
	// Only available when replying with a quote
	OriginalText string
	OriginalFrom []*mail.Address
	OriginalDate time.Time
}

func TestTemplateData() TemplateData {
	defaults := map[string]string{
		"To":           "John Doe <john@example.com>",
		"Cc":           "Josh Doe <josh@example.com>",
		"From":         "Jane Smith <jane@example.com>",
		"Subject":      "This is only a test",
		"OriginalText": "This is only a test text",
		"OriginalFrom": "John Doe <john@example.com>",
		"OriginalDate": time.Now().Format("Mon Jan 2, 2006 at 3:04 PM"),
	}

	return ParseTemplateData(defaults)
}

func ParseTemplateData(defaults map[string]string) TemplateData {
	originalDate, _ := time.Parse("Mon Jan 2, 2006 at 3:04 PM", defaults["OriginalDate"])
	td := TemplateData{
		To:           parseAddressList(defaults["To"]),
		Cc:           parseAddressList(defaults["Cc"]),
		Bcc:          parseAddressList(defaults["Bcc"]),
		From:         parseAddressList(defaults["From"]),
		Date:         time.Now(),
		Subject:      defaults["Subject"],
		OriginalText: defaults["Original"],
		OriginalFrom: parseAddressList(defaults["OriginalFrom"]),
		OriginalDate: originalDate,
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

func wrapLine(text string, lineWidth int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	wrapped := words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}

	return wrapped
}

func wrapText(text string, lineWidth int) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	var wrapped string

	for _, line := range lines {
		wrapped += wrapLine(line, lineWidth) + "\n"
	}
	return wrapped
}

// Wraping lines at 70 so that with the "> " of the quote it is under 72
func quote(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")

	quoted := "> " + wrapText(text, 70)
	quoted = strings.ReplaceAll(quoted, "\n", "\n> ")
	return quoted
}

var templateFuncs = template.FuncMap{
	"quote":      quote,
	"wrapText":   wrapText,
	"dateFormat": time.Time.Format,
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

	return "", errors.New("Can't find template - " + templateName)
}

func ParseTemplateFromFile(templateName string, templateDirs []string, data interface{}) ([]byte, error) {
	templateFile, err := findTemplate(templateName, templateDirs)
	if err != nil {
		return nil, err
	}
	emailTemplate, err := template.New(templateName).
		Funcs(templateFuncs).ParseFiles(templateFile)
	if err != nil {
		return nil, err
	}

	var outString bytes.Buffer
	if err := emailTemplate.Execute(&outString, data); err != nil {
		return nil, err
	}
	return outString.Bytes(), nil
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
