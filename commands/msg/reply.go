package msg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	gomail "net/mail"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type reply struct{}

func init() {
	register(reply{})
}

func (reply) Aliases() []string {
	return []string{"reply"}
}

func (reply) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (reply) Execute(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, "aqT:")
	if err != nil {
		return err
	}
	if optind != len(args) {
		return errors.New("Usage: reply [-aq -T <template>]")
	}
	var (
		quote    bool
		replyAll bool
		template string
	)
	for _, opt := range opts {
		switch opt.Option {
		case 'a':
			replyAll = true
		case 'q':
			quote = true
		case 'T':
			template = opt.Value
		}
	}

	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()

	if acct == nil {
		return errors.New("No account selected")
	}
	conf := acct.AccountConfig()
	us, _ := gomail.ParseAddress(conf.From)
	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	acct.Logger().Println("Replying to email " + msg.Envelope.MessageId)

	var (
		to     []string
		cc     []string
		toList []*models.Address
	)
	if args[0] == "reply" {
		if len(msg.Envelope.ReplyTo) != 0 {
			toList = msg.Envelope.ReplyTo
		} else {
			toList = msg.Envelope.From
		}
		for _, addr := range toList {
			if addr.Name != "" {
				to = append(to, fmt.Sprintf("%s <%s@%s>",
					addr.Name, addr.Mailbox, addr.Host))
			} else {
				to = append(to, fmt.Sprintf("<%s@%s>", addr.Mailbox, addr.Host))
			}
		}
		isMainRecipient := func(a *models.Address) bool {
			for _, ta := range toList {
				if ta.Mailbox == a.Mailbox && ta.Host == a.Host {
					return true
				}
			}
			return false
		}
		if replyAll {
			for _, addr := range msg.Envelope.Cc {
				//dedupe stuff already in the to: header, no need to repeat
				if isMainRecipient(addr) {
					continue
				}
				cc = append(cc, addr.Format())
			}
			for _, addr := range msg.Envelope.To {
				address := fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
				if strings.EqualFold(address, us.Address) {
					continue
				}
				to = append(to, addr.Format())
			}
		}
	}

	var subject string
	if !strings.HasPrefix(strings.ToLower(msg.Envelope.Subject), "re: ") {
		subject = "Re: " + msg.Envelope.Subject
	} else {
		subject = msg.Envelope.Subject
	}

	defaults := map[string]string{
		"To":          strings.Join(to, ", "),
		"Cc":          strings.Join(cc, ", "),
		"Subject":     subject,
		"In-Reply-To": msg.Envelope.MessageId,
	}
	original := models.OriginalMail{}

	addTab := func() error {
		if template != "" {
			original.From = models.FormatAddresses(msg.Envelope.From)
			original.Date = msg.Envelope.Date.Format("Mon Jan 2, 2006 at 3:04 PM")
		}

		composer, err := widgets.NewComposer(aerc, acct, aerc.Config(),
			acct.AccountConfig(), acct.Worker(), template, defaults, original)
		if err != nil {
			aerc.PushError("Error: "+err.Error(), 10*time.Second)
			return err
		}

		if args[0] == "reply" {
			composer.FocusTerminal()
		}

		tab := aerc.NewTab(composer, subject)
		composer.OnHeaderChange("Subject", func(subject string) {
			if subject == "" {
				tab.Name = "New email"
			} else {
				tab.Name = subject
			}
			tab.Content.Invalidate()
		})

		composer.OnClose(func(c *widgets.Composer) {
			store.Answered([]uint32{msg.Uid}, c.Sent(), nil)
		})

		return nil
	}

	if quote {
		if template == "" {
			template = aerc.Config().Templates.QuotedReply
		}

		part := findPlaintext(msg.BodyStructure, nil)
		if part == nil {
			//mkey... let's get the first thing that isn't a container
			part = findFirstNonMultipart(msg.BodyStructure, nil)
			if part == nil {
				// give up, use whatever is first
				part = []int{1}
			}
		}
		store.FetchBodyPart(msg.Uid, part, func(reader io.Reader) {
			buf := new(bytes.Buffer)
			buf.ReadFrom(reader)
			original.Text = buf.String()
			if len(msg.BodyStructure.Parts) == 0 {
				original.MIMEType = fmt.Sprintf("%s/%s",
					msg.BodyStructure.MIMEType, msg.BodyStructure.MIMESubType)
			} else {
				// TODO: still will be "multipart/mixed" for mixed mails with
				// attachments, fix this after aerc could handle responding to
				// such mails
				original.MIMEType = fmt.Sprintf("%s/%s",
					msg.BodyStructure.Parts[0].MIMEType,
					msg.BodyStructure.Parts[0].MIMESubType)
			}
			addTab()
		})
		return nil
	} else {
		return addTab()
	}
}
