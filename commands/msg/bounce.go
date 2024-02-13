package msg

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/mode"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/send"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Bounce struct {
	Account string   `opt:"-A" complete:"CompleteAccount"`
	To      []string `opt:"..." required:"true" complete:"CompleteTo"`
}

func init() {
	commands.Register(Bounce{})
}

func (Bounce) Aliases() []string {
	return []string{"bounce", "resend"}
}

func (*Bounce) CompleteAccount(arg string) []string {
	return commands.FilterList(app.AccountNames(), arg, commands.QuoteSpace)
}

func (*Bounce) CompleteTo(arg string) []string {
	return commands.FilterList(commands.GetAddress(arg), arg, commands.QuoteSpace)
}

func (Bounce) Context() commands.CommandContext {
	return commands.MESSAGE
}

func (b Bounce) Execute(args []string) error {
	if len(b.To) == 0 {
		return errors.New("No recipients specified")
	}
	addresses := strings.Join(b.To, ", ")

	app.PushStatus("Bouncing to "+addresses, 10*time.Second)

	widget := app.SelectedTabContent().(app.ProvidesMessage)

	var err error
	acct := widget.SelectedAccount()
	if b.Account != "" {
		acct, err = app.Account(b.Account)
	}
	switch {
	case err != nil:
		return fmt.Errorf("Failed to select account %q: %w", b.Account, err)
	case acct == nil:
		return errors.New("No account selected")
	}

	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	config := acct.AccountConfig()

	outgoing, err := config.Outgoing.ConnectionString()
	if err != nil {
		return errors.Wrap(err, "ReadCredentials()")
	}
	if outgoing == "" {
		return errors.New("No outgoing mail transport configured for this account")
	}
	uri, err := url.Parse(outgoing)
	if err != nil {
		return errors.Wrap(err, "url.Parse()")
	}

	rcpts, err := mail.ParseAddressList(addresses)
	if err != nil {
		return errors.Wrap(err, "ParseAddressList()")
	}

	var domain string
	if domain_, ok := config.Params["smtp-domain"]; ok {
		domain = domain_
	}

	hostname, err := send.GetMessageIdHostname(config.SendWithHostname, config.From)
	if err != nil {
		return errors.Wrap(err, "GetMessageIdHostname()")
	}

	// According to RFC2822, all of the resent fields corresponding
	// to a particular resending of the message SHOULD be together.
	// Each new set of resent fields is prepended to the message;
	// that is, the most recent set of resent fields appear earlier in the
	// message.
	headers := fmt.Sprintf("Resent-From: %s\r\n", config.From)
	headers += "Resent-Date: %s\r\n"
	headers += "Resent-Message-ID: <%s>\r\n"
	headers += fmt.Sprintf("Resent-To: %s\r\n", addresses)

	helper := newHelper()
	uids, err := helper.markedOrSelectedUids()
	if err != nil {
		return err
	}

	mode.NoQuit()

	marker := store.Marker()
	marker.ClearVisualMark()

	errCh := make(chan error)
	store.FetchFull(uids, func(fm *types.FullMessage) {
		defer log.PanicHandler()

		var header mail.Header
		var msgId string
		var err, errClose error

		uid := fm.Content.Uid
		msg := store.Messages[uid]
		if msg == nil {
			errCh <- fmt.Errorf("no message info: %v", uid)
			return
		}
		if err = header.GenerateMessageIDWithHostname(hostname); err != nil {
			errCh <- errors.Wrap(err, "GenerateMessageIDWithHostname()")
			return
		}
		if msgId, err = header.MessageID(); err != nil {
			errCh <- errors.Wrap(err, "MessageID()")
			return
		}
		reader := strings.NewReader(fmt.Sprintf(headers,
			time.Now().Format(time.RFC1123Z), msgId))

		go func() {
			defer log.PanicHandler()
			defer func() { errCh <- err }()

			var sender io.WriteCloser

			log.Debugf("Bouncing email <%s> to %s",
				msg.Envelope.MessageId, addresses)

			if sender, err = send.NewSender(acct.Worker(), uri,
				domain, config.From, rcpts); err != nil {
				return
			}
			defer func() {
				errClose = sender.Close()
				// If there has already been an error,
				// we don't want to clobber it.
				if err == nil {
					err = errClose
				} else if errClose != nil {
					app.PushError(errClose.Error())
				}
			}()
			if _, err = io.Copy(sender, reader); err != nil {
				return
			}
			_, err = io.Copy(sender, fm.Content.Reader)
		}()
	})

	go func() {
		defer log.PanicHandler()
		defer mode.NoQuitDone()

		var total, success int

		for err = range errCh {
			if err != nil {
				app.PushError(err.Error())
			} else {
				success++
			}
			total++
			if total == len(uids) {
				break
			}
		}
		if success != total {
			marker.Remark()
			app.PushError(fmt.Sprintf("Failed to bounce %d of the messages",
				total-success))
		} else {
			plural := ""
			if success > 1 {
				plural = "s"
			}
			app.PushStatus(fmt.Sprintf("Bounced %d message%s",
				success, plural), 10*time.Second)
		}
	}()

	return nil
}
