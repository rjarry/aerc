package account

import (
	"errors"
	"fmt"
	"io"
	gomail "net/mail"
	"regexp"
	"strings"

	"github.com/emersion/go-message/mail"

	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Compose struct{}

func init() {
	register(Compose{})
}

func (Compose) Aliases() []string {
	return []string{"compose"}
}

func (Compose) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Compose) Execute(aerc *widgets.Aerc, args []string) error {
	body, template, err := buildBody(args)
	if err != nil {
		return err
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if template == "" {
		template = aerc.Config().Templates.NewMessage
	}

	msg, err := gomail.ReadMessage(strings.NewReader(body))
	if errors.Is(err, io.EOF) { // completely empty
		msg = &gomail.Message{Body: strings.NewReader("")}
	} else if err != nil {
		return fmt.Errorf("mail.ReadMessage: %w", err)
	}
	headers := mail.HeaderFromMap(msg.Header)

	composer, err := widgets.NewComposer(aerc, acct,
		aerc.Config(), acct.AccountConfig(), acct.Worker(),
		template, &headers, models.OriginalMail{})
	if err != nil {
		return err
	}
	tab := aerc.NewTab(composer, "New email")
	composer.OnHeaderChange("Subject", func(subject string) {
		if subject == "" {
			tab.Name = "New email"
		} else {
			tab.Name = subject
		}
		ui.Invalidate()
	})
	go func() {
		defer logging.PanicHandler()

		composer.AppendContents(msg.Body)
	}()
	return nil
}

func buildBody(args []string) (string, string, error) {
	var body, template, headers string
	opts, optind, err := getopt.Getopts(args, "H:T:")
	if err != nil {
		return "", "", err
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
		}
	}
	posargs := args[optind:]
	if len(posargs) > 1 {
		return "", template, errors.New("Usage: compose [-H header] [-T template] [body]")
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
	return body, template, nil
}
