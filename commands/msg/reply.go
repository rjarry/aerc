package msg

import (
	"bytes"
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
	"git.sr.ht/~rjarry/aerc/lib/parse"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"github.com/danwakefield/fnmatch"
	"github.com/emersion/go-message/mail"
)

type reply struct {
	All      bool   `opt:"-a"`
	Close    bool   `opt:"-c"`
	Quote    bool   `opt:"-q"`
	Template string `opt:"-T" complete:"CompleteTemplate"`
	Edit     bool   `opt:"-e"`
	NoEdit   bool   `opt:"-E"`
	Account  string `opt:"-A" complete:"CompleteAccount"`
}

func init() {
	register(reply{})
}

func (reply) Aliases() []string {
	return []string{"reply"}
}

func (*reply) CompleteTemplate(arg string) []string {
	return commands.GetTemplates(arg)
}

func (*reply) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, commands.QuoteSpace)
}

func (r reply) Execute(args []string) error {
	editHeaders := (config.Compose.EditHeaders || r.Edit) && !r.NoEdit

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

	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}

	from := chooseFromAddr(conf, msg)

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

	if !config.Compose.ReplyToSelf {
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

	recSet.AddList(to)

	if r.All {
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
			// dedupe stuff from the to/from headers
			if recSet.Contains(addr) {
				continue
			}
			cc = append(cc, addr)
		}
		recSet.AddList(cc)
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

	mv, _ := app.SelectedTabContent().(*app.MessageViewer)
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

		if args[0] == "reply" {
			composer.FocusTerminal()
		}

		composer.Tab = app.NewTab(composer, subject)

		composer.OnClose(func(c *app.Composer) {
			switch {
			case c.Sent() && c.Archive() != "":
				store.Answered([]uint32{msg.Uid}, true, nil)
				err := archive([]*models.MessageInfo{msg}, c.Archive())
				if err != nil {
					app.PushStatus("Archive failed", 10*time.Second)
				}
			case c.Sent():
				store.Answered([]uint32{msg.Uid}, true, nil)
			case mv != nil && r.Close:
				view := account.ViewMessage{Peek: true}
				//nolint:errcheck // who cares?
				view.Execute([]string{"view", "-p"})
			}
		})

		return nil
	}

	if r.Quote {
		if r.Template == "" {
			r.Template = config.Templates.QuotedReply
		}

		if crypto.IsEncrypted(msg.BodyStructure) {
			provider := app.SelectedTabContent().(app.ProvidesMessage)
			mv, ok := provider.(*app.MessageViewer)
			if !ok {
				return fmt.Errorf("message is encrypted. can only quote reply while message is open")
			}
			p := provider.SelectedMessagePart()
			if p == nil {
				return fmt.Errorf("could not fetch message part")
			}
			mv.MessageView().FetchBodyPart(p.Index, func(reader io.Reader) {
				buf := new(bytes.Buffer)
				_, err := buf.ReadFrom(reader)
				if err != nil {
					log.Warnf("failed to fetch bodypart: %v", err)
				}
				original.Text = buf.String()
				err = addTab()
				if err != nil {
					log.Warnf("failed to add tab: %v", err)
				}
			})
			return nil
		}

		var part []int
		for _, mime := range config.Viewer.Alternatives {
			part = lib.FindMIMEPart(mime, msg.BodyStructure, nil)
			if part != nil {
				break
			}
		}

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

		store.FetchBodyPart(msg.Uid, part, func(reader io.Reader) {
			buf := new(bytes.Buffer)
			_, err := buf.ReadFrom(reader)
			if err != nil {
				log.Warnf("failed to fetch bodypart: %v", err)
			}
			original.Text = buf.String()
			err = addTab()
			if err != nil {
				log.Warnf("failed to add tab: %v", err)
			}
		})
		return nil
	} else {
		if r.Template == "" {
			r.Template = config.Templates.NewMessage
		}
		return addTab()
	}
}

func chooseFromAddr(conf *config.AccountConfig, msg *models.MessageInfo) *mail.Address {
	if len(conf.Aliases) == 0 {
		return conf.From
	}

	rec := newAddrSet()
	rec.AddList(msg.Envelope.To)
	rec.AddList(msg.Envelope.Cc)
	// test the from first, it has priority over any present alias
	if rec.Contains(conf.From) {
		// do nothing
	} else {
		for _, a := range conf.Aliases {
			if match := rec.FindMatch(a); match != "" {
				return &mail.Address{Name: a.Name, Address: match}
			}
		}
	}

	return conf.From
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
