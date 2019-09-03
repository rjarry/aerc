package account

import (
	"errors"
	"regexp"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/widgets"
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
	body, err := buildBody(args)
	if err != nil {
		return err
	}
	acct := aerc.SelectedAccount()
	composer := widgets.NewComposer(
		aerc.Config(), acct.AccountConfig(), acct.Worker(), nil)
	tab := aerc.NewTab(composer, "New email")
	composer.OnHeaderChange("Subject", func(subject string) {
		if subject == "" {
			tab.Name = "New email"
		} else {
			tab.Name = subject
		}
		tab.Content.Invalidate()
	})
	go composer.SetContents(strings.NewReader(body))
	return nil
}

func buildBody(args []string) (string, error) {
	var body, headers string
	opts, optind, err := getopt.Getopts(args, "H:")
	if err != nil {
		return "", err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'H':
			if strings.Index(opt.Value, ":") != -1 {
				// ensure first colon is followed by a single space
				re := regexp.MustCompile(`^(.*?):\s*(.*)`)
				headers += re.ReplaceAllString(opt.Value, "$1: $2") + "\n"
			} else {
				headers += opt.Value + ":\n"
			}
		}
	}
	posargs := args[optind:]
	if len(posargs) > 1 {
		return "", errors.New("Usage: compose [-H] [body]")
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
	return body, nil
}
