package msg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/lib/format"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"github.com/emersion/go-message/mail"
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
	from, err := format.ParseAddress(conf.From)
	if err != nil {
		return err
	}
	aliases, err := format.ParseAddressList(conf.Aliases)
	if err != nil {
		return err
	}

	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
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

	var (
		to []*mail.Address
		cc []*mail.Address
	)

	recSet := newAddrSet() // used for de-duping

	if len(msg.Envelope.ReplyTo) != 0 {
		to = msg.Envelope.ReplyTo
	} else {
		to = msg.Envelope.From
	}
	recSet.AddList(to)

	if replyAll {
		// order matters, due to the deduping
		// in order of importance, first parse the To, then the Cc header

		// we add our from address, so that we don't self address ourselves
		recSet.Add(from)

		envTos := make([]*mail.Address, 0, len(msg.Envelope.To))
		for _, addr := range msg.Envelope.To {
			if recSet.Contains(addr) {
				continue
			}
			envTos = append(envTos, addr)
		}
		recSet.AddList(envTos)
		to = append(to, envTos...)

		for _, addr := range msg.Envelope.Cc {
			//dedupe stuff from the to/from headers
			if recSet.Contains(addr) {
				continue
			}
			cc = append(cc, addr)
		}
		recSet.AddList(cc)
	}

	var subject string
	if !strings.HasPrefix(strings.ToLower(msg.Envelope.Subject), "re: ") {
		subject = "Re: " + msg.Envelope.Subject
	} else {
		subject = msg.Envelope.Subject
	}

	defaults := map[string]string{
		"To":          format.FormatAddresses(to),
		"Cc":          format.FormatAddresses(cc),
		"From":        format.AddressForHumans(from),
		"Subject":     subject,
		"In-Reply-To": msg.Envelope.MessageId,
	}
	original := models.OriginalMail{
		From: format.FormatAddresses(msg.Envelope.From),
		Date: msg.Envelope.Date,
		RFC822Headers: msg.RFC822Headers,
	}

	addTab := func() error {
		composer, err := widgets.NewComposer(aerc, acct, aerc.Config(),
			acct.AccountConfig(), acct.Worker(), template, defaults, original)
		if err != nil {
			aerc.PushError("Error: " + err.Error())
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
			if c.Sent() {
				store.Answered([]uint32{msg.Uid}, true, nil)
			}
		})

		return nil
	}

	if quote {
		if template == "" {
			template = aerc.Config().Templates.QuotedReply
		}

		part := lib.FindPlaintext(msg.BodyStructure, nil)
		if part == nil {
			// mkey... let's get the first thing that isn't a container
			// if that's still nil it's either not a multipart msg (ok) or
			// broken (containers only)
			part = lib.FindFirstNonMultipart(msg.BodyStructure, nil)
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

type addrSet map[string]struct{}

func newAddrSet() addrSet {
	s := make(map[string]struct{})
	return addrSet(s)
}

func (s addrSet) Add(a *mail.Address) {
	s[a.Address] = struct{}{}
}

func (s addrSet) AddList(al []*mail.Address) {
	for _, a := range al {
		s[a.Address] = struct{}{}
	}
}

func (s addrSet) Contains(a *mail.Address) bool {
	_, ok := s[a.Address]
	return ok
}
