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
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~sircmpwn/getopt"
)

type Compose struct{}

func init() {
	register(Compose{})
}

func (Compose) Aliases() []string {
	return []string{"compose"}
}

func (Compose) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (Compose) Execute(aerc *app.Aerc, args []string) error {
	body, template, editHeaders, err := buildBody(args)
	if err != nil {
		return err
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if template == "" {
		template = config.Templates.NewMessage
	}

	msg, err := gomail.ReadMessage(strings.NewReader(body))
	if errors.Is(err, io.EOF) { // completely empty
		msg = &gomail.Message{Body: strings.NewReader("")}
	} else if err != nil {
		return fmt.Errorf("mail.ReadMessage: %w", err)
	}
	headers := mail.HeaderFromMap(msg.Header)

	composer, err := app.NewComposer(aerc, acct,
		acct.AccountConfig(), acct.Worker(), editHeaders,
		template, &headers, nil, msg.Body)
	if err != nil {
		return err
	}
	composer.Tab = aerc.NewTab(composer, "New email")
	return nil
}

func buildBody(args []string) (string, string, bool, error) {
	var body, template, headers string
	editHeaders := config.Compose.EditHeaders
	opts, optind, err := getopt.Getopts(args, "H:T:eE")
	if err != nil {
		return "", "", false, err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'H':
			if strings.Contains(opt.Value, ":") {
				// ensure first colon is followed by a single space
				re := regexp.MustCompile(`^(.*?):\s*(.*)`)
				headers += re.ReplaceAllString(opt.Value, "$1: $2") + "\n"
			} else {
				headers += opt.Value + ":\n"
			}
		case 'T':
			template = opt.Value
		case 'e':
			editHeaders = true
		case 'E':
			editHeaders = false
		}
	}
	posargs := args[optind:]
	if len(posargs) > 1 {
		return "", "", false, errors.New("Usage: compose [-H header] [-T template] [-e|-E] [body]")
	}
	if len(posargs) == 1 {
		body = posargs[0]
	}
	if headers != "" {
		if len(body) > 0 {
			body = headers + "\n" + body
		} else {
			body = headers + "\n\n"
		}
	}
	return body, template, editHeaders, nil
}
