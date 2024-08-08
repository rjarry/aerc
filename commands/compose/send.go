package compose

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/commands/mode"
	"git.sr.ht/~rjarry/aerc/commands/msg"
	"git.sr.ht/~rjarry/aerc/lib/hooks"
	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/lib/send"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-message/mail"
)

type Send struct {
	Archive string `opt:"-a" action:"ParseArchive" metavar:"flat|year|month" complete:"CompleteArchive"`
	CopyTo  string `opt:"-t" complete:"CompleteFolders"`

	CopyToReplied   bool `opt:"-r"`
	NoCopyToReplied bool `opt:"-R"`
}

func init() {
	commands.Register(Send{})
}

func (Send) Context() commands.CommandContext {
	return commands.COMPOSE
}

func (Send) Aliases() []string {
	return []string{"send"}
}

func (*Send) CompleteArchive(arg string) []string {
	return commands.FilterList(msg.ARCHIVE_TYPES, arg, nil)
}

func (*Send) CompleteFolders(arg string) []string {
	return commands.GetFolders(arg)
}

func (s *Send) ParseArchive(arg string) error {
	for _, a := range msg.ARCHIVE_TYPES {
		if a == arg {
			s.Archive = arg
			return nil
		}
	}
	return errors.New("unsupported archive type")
}

func (s Send) Execute(args []string) error {
	tab := app.SelectedTab()
	if tab == nil {
		return errors.New("No selected tab")
	}
	composer, _ := tab.Content.(*app.Composer)

	err := composer.CheckForMultipartErrors()
	if err != nil {
		return err
	}

	config := composer.Config()

	if s.CopyTo == "" {
		s.CopyTo = config.CopyTo
	}
	copyToReplied := config.CopyToReplied || (s.CopyToReplied && !s.NoCopyToReplied)

	outgoing, err := config.Outgoing.ConnectionString()
	if err != nil {
		return errors.Wrap(err, "ReadCredentials(outgoing)")
	}
	if outgoing == "" {
		return errors.New(
			"No outgoing mail transport configured for this account")
	}

	header, err := composer.PrepareHeader()
	if err != nil {
		return errors.Wrap(err, "PrepareHeader")
	}
	rcpts, err := listRecipients(header)
	if err != nil {
		return errors.Wrap(err, "listRecipients")
	}
	if len(rcpts) == 0 {
		return errors.New("Cannot send message with no recipients")
	}

	uri, err := url.Parse(outgoing)
	if err != nil {
		return errors.Wrap(err, "url.Parse(outgoing)")
	}

	var domain string
	if domain_, ok := config.Params["smtp-domain"]; ok {
		domain = domain_
	}
	from := config.From

	log.Debugf("send config uri: %s", uri.Redacted())
	log.Debugf("send config from: %s", from)
	log.Debugf("send config rcpts: %s", rcpts)
	log.Debugf("send config domain: %s", domain)

	warnSubject := composer.ShouldWarnSubject()
	warnAttachment := composer.ShouldWarnAttachment()
	if warnSubject || warnAttachment {
		var msg string
		switch {
		case warnSubject && warnAttachment:
			msg = "The subject is empty, and you may have forgotten an attachment."
		case warnSubject:
			msg = "The subject is empty."
		default:
			msg = "You may have forgotten an attachment."
		}

		prompt := app.NewPrompt(
			msg+" Abort send? [Y/n] ",
			func(text string) {
				if text == "n" || text == "N" {
					sendHelper(composer, header, uri, domain,
						from, rcpts, tab.Name, s.CopyTo,
						s.Archive, copyToReplied)
				}
			}, func(cmd string) ([]string, string) {
				if cmd == "" {
					return []string{"y", "n"}, ""
				}

				return nil, ""
			},
		)

		app.PushPrompt(prompt)
	} else {
		sendHelper(composer, header, uri, domain, from, rcpts, tab.Name,
			s.CopyTo, s.Archive, copyToReplied)
	}

	return nil
}

func sendHelper(composer *app.Composer, header *mail.Header, uri *url.URL, domain string,
	from *mail.Address, rcpts []*mail.Address, tabName string, copyTo string,
	archive string, copyToReplied bool,
) {
	// we don't want to block the UI thread while we are sending
	// so we do everything in a goroutine and hide the composer from the user
	app.RemoveTab(composer, false)
	app.PushStatus("Sending...", 10*time.Second)

	// enter no-quit mode
	mode.NoQuit()

	var shouldCopy bool = copyTo != "" && !strings.HasPrefix(uri.Scheme, "jmap")
	var copyBuf bytes.Buffer

	failCh := make(chan error)
	// writer
	go func() {
		defer log.PanicHandler()

		var parentDir string
		if copyToReplied && composer.Parent() != nil {
			parentDir = composer.Parent().Folder
		}
		sender, err := send.NewSender(
			composer.Worker(), uri, domain, from, rcpts, parentDir)
		if err != nil {
			failCh <- errors.Wrap(err, "send:")
			return
		}

		var writer io.Writer = sender

		if shouldCopy {
			writer = io.MultiWriter(writer, &copyBuf)
		}

		err = composer.WriteMessage(header, writer)
		if err != nil {
			failCh <- err
			return
		}
		failCh <- sender.Close()
	}()

	// cleanup + copy to sent
	go func() {
		defer log.PanicHandler()

		// leave no-quit mode
		defer mode.NoQuitDone()

		err := <-failCh
		if err != nil {
			app.PushError(strings.ReplaceAll(err.Error(), "\n", " "))
			app.NewTab(composer, tabName)
			return
		}
		if shouldCopy {
			app.PushStatus("Copying to "+copyTo, 10*time.Second)
			errch := copyToSent(copyTo, copyToReplied, copyBuf.Len(),
				&copyBuf, composer)
			err = <-errch
			if err != nil {
				errmsg := fmt.Sprintf(
					"message sent, but copying to %v failed: %v",
					copyTo, err.Error())
				app.PushError(errmsg)
				composer.SetSent(archive)
				composer.Close()
				return
			}
		}
		app.PushStatus("Message sent.", 10*time.Second)
		composer.SetSent(archive)
		err = hooks.RunHook(&hooks.MailSent{
			Account: composer.Account().Name(),
			Backend: composer.Account().AccountConfig().Backend,
			Header:  header,
		})
		if err != nil {
			log.Errorf("failed to trigger mail-sent hook: %v", err)
			composer.Account().PushError(fmt.Errorf("[hook.mail-sent] failed: %w", err))
		}
		composer.Close()
	}()
}

func listRecipients(h *mail.Header) ([]*mail.Address, error) {
	var rcpts []*mail.Address
	for _, key := range []string{"to", "cc", "bcc"} {
		list, err := h.AddressList(key)
		if err != nil {
			return nil, err
		}
		rcpts = append(rcpts, list...)
	}
	return rcpts, nil
}

func copyToSent(dest string, copyToReplied bool, n int, msg io.Reader, composer *app.Composer) <-chan error {
	errCh := make(chan error, 1)
	acct := composer.Account()
	if acct == nil {
		errCh <- errors.New("No account selected")
		return errCh
	}
	store := acct.Store()
	if store == nil {
		errCh <- errors.New("No message store selected")
		return errCh
	}
	store.Append(
		dest,
		models.SeenFlag,
		time.Now(),
		msg,
		n,
		func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				errCh <- nil
			case *types.Error:
				errCh <- msg.Error
			}
		},
	)
	if copyToReplied && composer.Parent() != nil {
		store.Append(
			composer.Parent().Folder,
			models.SeenFlag,
			time.Now(),
			msg,
			n,
			func(msg types.WorkerMessage) {
				switch msg := msg.(type) {
				case *types.Done:
					errCh <- nil
				case *types.Error:
					errCh <- msg.Error
				}
			},
		)
	}
	return errCh
}
