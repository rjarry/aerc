package msg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/account"
	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/crypto"
	"git.sr.ht/~rjarry/aerc/lib/format"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/danwakefield/fnmatch"
	"github.com/emersion/go-message/mail"
)

type reply struct {
	All        bool   `opt:"-a" desc:"Reply to all recipients."`
	Close      bool   `opt:"-c" desc:"Close the view tab when replying."`
	From       bool   `opt:"-f" desc:"Reply to all addresses in From and Reply-To headers."`
	Quote      bool   `opt:"-q" desc:"Alias of -T quoted-reply."`
	Template   string `opt:"-T" complete:"CompleteTemplate" desc:"Template name."`
	Edit       bool   `opt:"-e" desc:"Force [compose].edit-headers = true."`
	NoEdit     bool   `opt:"-E" desc:"Force [compose].edit-headers = false."`
	Account    string `opt:"-A" complete:"CompleteAccount" desc:"Reply with the specified account."`
	SkipEditor bool   `opt:"-s" desc:"Skip the editor and go directly to the review screen."`
}

func init() {
	commands.Register(reply{})
}

func (reply) Description() string {
	return "Open the composer to reply to the selected message."
}

func (reply) Context() commands.CommandContext {
	return commands.MESSAGE_LIST | commands.MESSAGE_VIEWER
}

func (reply) Aliases() []string {
	return []string{"reply"}
}

func (*reply) CompleteTemplate(arg string) []string {
	return commands.GetTemplates(arg)
}

func (*reply) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, nil)
}

func (r reply) Execute(args []string) error {
	editHeaders := (config.Compose().EditHeaders || r.Edit) && !r.NoEdit

	widget := app.SelectedTabContent().(app.ProvidesMessage)

	var acct *app.AccountView
	var err error

	if r.Account == "" {
		acct = widget.SelectedAccount()
		if acct == nil {
			return errors.New("No account selected")
		}
	} else {
		acct, err = app.Account(r.Account)
		if err != nil {
			return err
		}
	}
	conf := acct.AccountConfig()

	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}

	from, err := chooseFromAddr(conf, msg)
	if err != nil {
		return err
	}

	var (
		to []*mail.Address
		cc []*mail.Address
	)

	recSet := newAddrSet() // used for de-duping
	dedupe := func(addrs []*mail.Address) []*mail.Address {
		deduped := make([]*mail.Address, 0, len(addrs))
		for _, addr := range addrs {
			if recSet.Contains(addr) {
				continue
			}
			recSet.Add(addr)
			deduped = append(deduped, addr)
		}
		return deduped
	}

	if config.Compose().ReplyToSelf {
		// We accept to reply to ourselves, so don't exclude our own address
		// from the reply's recipients.
	} else {
		recSet.Add(from)
	}

	switch {
	case len(msg.Envelope.ReplyTo) != 0:
		to = dedupe(msg.Envelope.ReplyTo)
	case len(msg.Envelope.From) != 0:
		to = dedupe(msg.Envelope.From)
	default:
		to = dedupe(msg.Envelope.Sender)
	}

	if r.From {
		to = append(to, dedupe(msg.Envelope.From)...)
	}

	if !config.Compose().ReplyToSelf && len(to) == 0 {
		recSet = newAddrSet()
		to = dedupe(msg.Envelope.To)
	}

	if r.All {
		// order matters, due to the deduping
		// in order of importance, first parse the To, then the Cc header

		to = append(to, dedupe(msg.Envelope.To)...)

		cc = append(cc, dedupe(msg.Envelope.Cc)...)
		cc = append(cc, dedupe(msg.Envelope.Sender)...)
	}

	subject := "Re: " + trimLocalizedRe(msg.Envelope.Subject, conf.LocalizedRe)

	h := &mail.Header{}
	h.SetAddressList("to", to)
	h.SetAddressList("cc", cc)
	h.SetAddressList("from", []*mail.Address{from})
	h.SetSubject(subject)
	h.SetMsgIDList("in-reply-to", []string{msg.Envelope.MessageId})
	err = setReferencesHeader(h, msg.RFC822Headers)
	if err != nil {
		app.PushError(fmt.Sprintf("could not set references: %v", err))
	}
	original := models.OriginalMail{
		From:          format.FormatAddresses(msg.Envelope.From),
		Date:          msg.Envelope.Date,
		RFC822Headers: msg.RFC822Headers,
	}

	mv, isMsgViewer := app.SelectedTabContent().(*app.MessageViewer)

	store := widget.Store()
	noStore := store == nil
	switch {
	case noStore && isMsgViewer:
		app.PushWarning("No message store found: answered flag cannot be set")
	case noStore:
		return errors.New("Cannot perform action. Messages still loading")
	default:
		original.Folder = store.Name
	}

	addTab := func() error {
		composer, err := app.NewComposer(acct,
			acct.AccountConfig(), acct.Worker(), editHeaders,
			r.Template, h, &original, nil)
		if err != nil {
			app.PushError("Error: " + err.Error())
			return err
		}
		if mv != nil && r.Close {
			app.RemoveTab(mv, true)
		}

		if r.SkipEditor {
			composer.Terminal().Close()
		} else if args[0] == "reply" {
			composer.FocusTerminal()
		}

		composer.Tab = app.NewTab(composer, subject)

		composer.OnClose(func(c *app.Composer) {
			switch {
			case c.Sent() && c.Archive() != "" && !noStore:
				store.Answered([]models.UID{msg.Uid}, true, nil)
				err := archive([]*models.MessageInfo{msg}, nil, c.Archive(), acct, store)
				if err != nil {
					app.PushStatus("Archive failed", 10*time.Second)
				}
			case c.Sent() && !noStore:
				store.Answered([]models.UID{msg.Uid}, true, nil)
			case mv != nil && r.Close:
				view := account.ViewMessage{Peek: true}
				//nolint:errcheck // who cares?
				view.Execute([]string{"view", "-p"})
			}
		})

		return nil
	}

	if r.Quote && r.Template == "" {
		r.Template = config.Templates().QuotedReply
	}

	if r.Template != "" {
		var fetchBodyPart func([]int, func(io.Reader))

		if isMsgViewer {
			fetchBodyPart = mv.MessageView().FetchBodyPart
		} else {
			fetchBodyPart = func(part []int, cb func(io.Reader)) {
				store.FetchBodyPart(context.TODO(), msg.Uid, part, cb)
			}
		}

		if crypto.IsEncrypted(msg.BodyStructure) && !isMsgViewer {
			return fmt.Errorf("message is encrypted. " +
				"can only include reply from the message viewer")
		}

		part := getMessagePart(msg, widget)
		if part == nil {
			// mkey... let's get the first thing that isn't a container
			// if that's still nil it's either not a multipart msg (ok) or
			// broken (containers only)
			part = lib.FindFirstNonMultipart(msg.BodyStructure, nil)
		}

		err = addMimeType(msg, part, &original)
		if err != nil {
			return err
		}

		fetchBodyPart(part, func(reader io.Reader) {
			data, err := io.ReadAll(reader)
			if err != nil {
				log.Warnf("failed to read bodypart: %v", err)
			}
			original.Text = string(data)
			err = addTab()
			if err != nil {
				log.Warnf("failed to add tab: %v", err)
			}
		})

		return nil
	} else {
		r.Template = config.Templates().NewMessage
		return addTab()
	}
}

func chooseFromAddr(conf *config.AccountConfig, msg *models.MessageInfo) (*mail.Address, error) {
	if len(conf.Aliases) == 0 {
		return conf.From, nil
	}

	rec := newAddrSet()
	rec.AddList(msg.Envelope.From)
	rec.AddList(msg.Envelope.To)
	rec.AddList(msg.Envelope.Cc)
	if conf.OriginalToHeader != "" && msg.RFC822Headers.Has(conf.OriginalToHeader) {
		origTo, err := msg.RFC822Headers.Text(conf.OriginalToHeader)
		if err != nil {
			return nil, err
		}

		origToAddress, err := mail.ParseAddressList(origTo)
		if err != nil {
			return nil, err
		}
		rec.AddList(origToAddress)
	}
	// test the from first, it has priority over any present alias
	if rec.Contains(conf.From) {
		// do nothing
	} else {
		for _, a := range conf.Aliases {
			if match := rec.FindMatch(a); match != "" {
				return &mail.Address{Name: a.Name, Address: match}, nil
			}
		}
	}

	return conf.From, nil
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

func (s addrSet) FindMatch(a *mail.Address) string {
	for addr := range s {
		if fnmatch.Match(a.Address, addr, 0) {
			return addr
		}
	}

	return ""
}

// setReferencesHeader adds the references header to target based on parent
// according to RFC2822
func setReferencesHeader(target, parent *mail.Header) error {
	refs := parse.MsgIDList(parent, "references")
	if len(refs) == 0 {
		// according to the RFC we need to fall back to in-reply-to only if
		// References is not set
		refs = parse.MsgIDList(parent, "in-reply-to")
	}
	msgID, err := parent.MessageID()
	if err != nil {
		return err
	}
	refs = append(refs, msgID)
	target.SetMsgIDList("references", refs)
	return nil
}

// addMimeType adds the proper mime type of the part to the originalMail struct
func addMimeType(msg *models.MessageInfo, part []int,
	orig *models.OriginalMail,
) error {
	// caution, :forward uses the code as well, keep that in mind when modifying
	bs, err := msg.BodyStructure.PartAtIndex(part)
	if err != nil {
		return err
	}
	orig.MIMEType = bs.FullMIMEType()
	return nil
}

// trimLocalizedRe removes known localizations of Re: commonly used by Outlook.
func trimLocalizedRe(subject string, localizedRe *regexp.Regexp) string {
	return strings.TrimPrefix(subject, localizedRe.FindString(subject))
}
