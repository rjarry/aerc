package account

import (
	"errors"
	"fmt"
	"strings"

	"github.com/emersion/go-imap"

	"git.sr.ht/~sircmpwn/aerc2/widgets"
)

func init() {
	register("reply", Reply)
}

func Reply(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: reply [-aq]")
	}
	// TODO: Reply all (w/ getopt)

	acct := aerc.SelectedAccount()
	msg := acct.Messages().Selected()
	acct.Logger().Println("Replying to email " + msg.Envelope.MessageId)

	var (
		to     []string
		cc     []string
		toList []*imap.Address
	)
	if len(msg.Envelope.ReplyTo) != 0 {
		toList = msg.Envelope.ReplyTo
	} else {
		toList = msg.Envelope.From
	}
	for _, addr := range toList {
		if addr.PersonalName != "" {
			to = append(to, fmt.Sprintf("%s <%s@%s>",
				addr.PersonalName, addr.MailboxName, addr.HostName))
		} else {
			to = append(to, fmt.Sprintf("<%s@%s>",
				addr.MailboxName, addr.HostName))
		}
	}
	// TODO: Only if reply all
	for _, addr := range msg.Envelope.Cc {
		if addr.PersonalName != "" {
			cc = append(cc, fmt.Sprintf("%s <%s@%s>",
				addr.PersonalName, addr.MailboxName, addr.HostName))
		} else {
			cc = append(cc, fmt.Sprintf("<%s@%s>",
				addr.MailboxName, addr.HostName))
		}
	}

	subject := "Re: " + msg.Envelope.Subject

	composer := widgets.NewComposer(
		aerc.Config(), acct.AccountConfig(), acct.Worker()).
		Defaults(map[string]string{
			"To": strings.Join(to, ","),
			"Cc": strings.Join(cc, ","),
			"Subject": subject,
			"In-Reply-To": msg.Envelope.MessageId,
		}).
		FocusTerminal()

	tab := aerc.NewTab(composer, subject)

	composer.OnSubjectChange(func(subject string) {
		if subject == "" {
			tab.Name = "New email"
		} else {
			tab.Name = subject
		}
		tab.Content.Invalidate()
	})

	return nil
}


