package msg

import (
	"errors"
	"fmt"
	"io"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/calendar"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"github.com/emersion/go-message/mail"
)

type invite struct{}

func init() {
	register(invite{})
}

func (invite) Aliases() []string {
	return []string{"accept", "accept-tentative", "decline"}
}

func (invite) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (invite) Execute(aerc *widgets.Aerc, args []string) error {
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("no account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("cannot perform action: messages still loading")
	}
	msg, err := acct.SelectedMessage()
	if err != nil {
		return err
	}

	part := lib.FindCalendartext(msg.BodyStructure, nil)
	if part == nil {
		return fmt.Errorf("no invitation found (missing text/calendar)")
	}

	subject := trimLocalizedRe(msg.Envelope.Subject, acct.AccountConfig().LocalizedRe)
	switch args[0] {
	case "accept":
		subject = "Accepted: " + subject
	case "accept-tentative":
		subject = "Tentatively Accepted: " + subject
	case "decline":
		subject = "Declined: " + subject
	default:
		return fmt.Errorf("no participation status defined")
	}

	conf := acct.AccountConfig()
	from, err := mail.ParseAddress(conf.From)
	if err != nil {
		return err
	}
	var aliases []*mail.Address
	if conf.Aliases != "" {
		aliases, err = mail.ParseAddressList(conf.Aliases)
		if err != nil {
			return err
		}
	}

	// figure out the sending from address if we have aliases
	if len(aliases) != 0 {
		rec := newAddrSet()
		rec.AddList(msg.Envelope.To)
		rec.AddList(msg.Envelope.Cc)
		// test the from first, it has priority over any present alias
		if rec.Contains(from) {
			// do nothing
		} else {
			for _, a := range aliases {
				if rec.Contains(a) {
					from = a
					break
				}
			}
		}
	}

	var to []*mail.Address

	if len(msg.Envelope.ReplyTo) != 0 {
		to = msg.Envelope.ReplyTo
	} else {
		to = msg.Envelope.From
	}

	if !aerc.Config().Compose.ReplyToSelf {
		for i, v := range to {
			if v.Address == from.Address {
				to = append(to[:i], to[i+1:]...)
				break
			}
		}
		if len(to) == 0 {
			to = msg.Envelope.To
		}
	}

	recSet := newAddrSet() // used for de-duping
	recSet.AddList(to)

	h := &mail.Header{}
	h.SetAddressList("from", []*mail.Address{from})
	h.SetSubject(subject)
	h.SetMsgIDList("in-reply-to", []string{msg.Envelope.MessageId})
	err = setReferencesHeader(h, msg.RFC822Headers)
	if err != nil {
		aerc.PushError(fmt.Sprintf("could not set references: %v", err))
	}
	original := models.OriginalMail{
		From:          format.FormatAddresses(msg.Envelope.From),
		Date:          msg.Envelope.Date,
		RFC822Headers: msg.RFC822Headers,
	}

	handleInvite := func(reader io.Reader) (*calendar.Reply, error) {
		cr, err := calendar.CreateReply(reader, from, args[0])
		if err != nil {
			return nil, err
		}
		for _, org := range cr.Organizers {
			organizer, err := mail.ParseAddress(org)
			if err != nil {
				continue
			}
			if !recSet.Contains(organizer) {
				to = append(to, organizer)
			}
		}
		h.SetAddressList("to", to)
		return cr, nil
	}

	addTab := func(cr *calendar.Reply) error {
		composer, err := widgets.NewComposer(aerc, acct, aerc.Config(),
			acct.AccountConfig(), acct.Worker(), "", h, original)
		if err != nil {
			aerc.PushError("Error: " + err.Error())
			return err
		}

		composer.SetContents(cr.PlainText)
		err = composer.AppendPart(cr.MimeType, cr.Params, cr.CalendarText)
		if err != nil {
			return fmt.Errorf("failed to write invitation: %w", err)
		}
		composer.FocusTerminal()

		tab := aerc.NewTab(composer, subject)
		composer.OnHeaderChange("Subject", func(subject string) {
			if subject == "" {
				tab.Name = "New email"
			} else {
				tab.Name = subject
			}
			ui.Invalidate()
		})

		composer.OnClose(func(c *widgets.Composer) {
			if c.Sent() {
				store.Answered([]uint32{msg.Uid}, true, nil)
			}
		})

		return nil
	}

	store.FetchBodyPart(msg.Uid, part, func(reader io.Reader) {
		if cr, err := handleInvite(reader); err != nil {
			aerc.PushError(err.Error())
			return
		} else {
			err := addTab(cr)
			if err != nil {
				log.Warnf("failed to add tab: %v", err)
			}
		}
	})
	return nil
}
