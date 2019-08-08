package account

import (
	"regexp"
	"strings"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Compose struct{}

func init() {
	register(Compose{})
}

func (_ Compose) Aliases() []string {
	return []string{"compose"}
}

func (_ Compose) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

// TODO: Accept arguments for message body
func (_ Compose) Execute(aerc *widgets.Aerc, args []string) error {
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
	opts, _, err := getopt.Getopts(args, "H:")
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
	if headers != "" {
		body = headers + "\n\n"
	}
	return body, nil
}
