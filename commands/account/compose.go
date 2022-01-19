package account

import (
	"errors"
	"regexp"
	"strings"

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
	if template == "" {
		template = aerc.Config().Templates.NewMessage
	}

	composer, err := widgets.NewComposer(aerc, acct,
		aerc.Config(), acct.AccountConfig(), acct.Worker(),
		template, nil, models.OriginalMail{})
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
		tab.Content.Invalidate()
	})
	go composer.AppendContents(strings.NewReader(body))
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
