package msg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"

	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type forward struct{}

func init() {
	register(forward{})
}

func (forward) Aliases() []string {
	return []string{"forward"}
}

func (forward) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (forward) Execute(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, "AT:")
	if err != nil {
		return err
	}
	attach := false
	template := ""
	for _, opt := range opts {
		switch opt.Option {
		case 'A':
			attach = true
		case 'T':
			template = opt.Value
		}
	}

	to := ""
	if len(args) != 1 {
		to = strings.Join(args[optind:], ", ")
	}

	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	acct.Logger().Println("Forwarding email " + msg.Envelope.MessageId)

	subject := "Fwd: " + msg.Envelope.Subject
	defaults := map[string]string{
		"To":      to,
		"Subject": subject,
	}

	addTab := func() (*widgets.Composer, error) {
		if template != "" {
			defaults["OriginalFrom"] = models.FormatAddresses(msg.Envelope.From)
			defaults["OriginalDate"] = msg.Envelope.Date.Format("Mon Jan 2, 2006 at 3:04 PM")
		}

		composer, err := widgets.NewComposer(aerc, aerc.Config(), acct.AccountConfig(),
			acct.Worker(), template, defaults)
		if err != nil {
			aerc.PushError("Error: " + err.Error())
			return nil, err
		}

		tab := aerc.NewTab(composer, subject)
		if to == "" {
			composer.FocusRecipient()
		} else {
			composer.FocusTerminal()
		}
		composer.OnHeaderChange("Subject", func(subject string) {
			if subject == "" {
				tab.Name = "New email"
			} else {
				tab.Name = subject
			}
			tab.Content.Invalidate()
		})
		return composer, nil
	}

	if attach {
		tmpDir, err := ioutil.TempDir("", "aerc-tmp-attachment")
		if err != nil {
			return err
		}
		tmpFileName := path.Join(tmpDir,
			strings.ReplaceAll(fmt.Sprintf("%s.eml", msg.Envelope.Subject), "/", "-"))
		store.FetchFull([]uint32{msg.Uid}, func(reader io.Reader) {
			tmpFile, err := os.Create(tmpFileName)
			if err != nil {
				println(err)
				// TODO: Do something with the error
				addTab()
				return
			}

			defer tmpFile.Close()
			io.Copy(tmpFile, reader)
			composer, err := addTab()
			if err != nil {
				return
			}
			composer.AddAttachment(tmpFileName)
			composer.OnClose(func(composer *widgets.Composer) {
				os.RemoveAll(tmpDir)
			})
		})
	} else {
		if template == "" {
			template = aerc.Config().Templates.Forwards
		}

		// TODO: something more intelligent than fetching the 1st part
		// TODO: add attachments!
		store.FetchBodyPart(msg.Uid, []int{1}, func(reader io.Reader) {
			header := message.Header{}
			header.SetText(
				"Content-Transfer-Encoding", msg.BodyStructure.Encoding)
			header.SetContentType(
				msg.BodyStructure.MIMEType, msg.BodyStructure.Params)
			header.SetText("Content-Description", msg.BodyStructure.Description)
			entity, err := message.New(header, reader)
			if err != nil {
				// TODO: Do something with the error
				addTab()
				return
			}
			mreader := mail.NewReader(entity)
			part, err := mreader.NextPart()
			if err != nil {
				// TODO: Do something with the error
				addTab()
				return
			}
			buf := new(bytes.Buffer)
			buf.ReadFrom(part.Body)
			defaults["Original"] = buf.String()
			addTab()
		})
	}
	return nil
}
