package account

import (
	"errors"
	"fmt"
	"io"
	gomail "net/mail"
	"regexp"
	"strings"

	"github.com/emersion/go-message/mail"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
)

type Compose struct {
	Headers    string `opt:"-H" action:"ParseHeader" desc:"Add the specified header to the message."`
	Template   string `opt:"-T" complete:"CompleteTemplate" desc:"Template name."`
	Edit       bool   `opt:"-e" desc:"Force [compose].edit-headers = true."`
	NoEdit     bool   `opt:"-E" desc:"Force [compose].edit-headers = false."`
	SkipEditor bool   `opt:"-s" desc:"Skip the editor and go directly to the review screen."`
	Body       string `opt:"..." required:"false"`
}

func init() {
	commands.Register(Compose{})
}

func (Compose) Description() string {
	return "Open the compose window to write a new email."
}

func (Compose) Context() commands.CommandContext {
	return commands.MESSAGE_LIST
}

func (c *Compose) ParseHeader(arg string) error {
	if strings.Contains(arg, ":") {
		// ensure first colon is followed by a single space
		re := regexp.MustCompile(`^(.*?):\s*(.*)`)
		c.Headers += re.ReplaceAllString(arg, "$1: $2\r\n")
	} else {
		c.Headers += arg + ":\r\n"
	}
	return nil
}

func (*Compose) CompleteTemplate(arg string) []string {
	return commands.GetTemplates(arg)
}

func (Compose) Aliases() []string {
	return []string{"compose"}
}

func (c Compose) Execute(args []string) error {
	if c.Headers != "" {
		if c.Body != "" {
			c.Body = c.Headers + "\r\n" + c.Body
		} else {
			c.Body = c.Headers + "\r\n\r\n"
		}
	}
	if c.Template == "" {
		c.Template = config.Templates.NewMessage
	}
	editHeaders := (config.Compose.EditHeaders || c.Edit) && !c.NoEdit

	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	msg, err := gomail.ReadMessage(strings.NewReader(c.Body))
	if errors.Is(err, io.EOF) { // completely empty
		msg = &gomail.Message{Body: strings.NewReader("")}
	} else if err != nil {
		return fmt.Errorf("mail.ReadMessage: %w", err)
	}
	headers := mail.HeaderFromMap(msg.Header)

	composer, err := app.NewComposer(acct,
		acct.AccountConfig(), acct.Worker(), editHeaders,
		c.Template, &headers, nil, msg.Body)
	if err != nil {
		return err
	}
	composer.Tab = app.NewTab(composer, "New email")
	if c.SkipEditor {
		composer.Terminal().Close()
	}
	return nil
}
