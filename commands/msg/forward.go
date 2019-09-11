package msg

import (
	"bufio"
	"errors"
	"fmt"
	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
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
	opts, optind, err := getopt.Getopts(args, "A")
	if err != nil {
		return err
	}
	attach := false
	for _, opt := range opts {
		switch opt.Option {
		case 'A':
			attach = true
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
	composer := widgets.NewComposer(aerc, aerc.Config(), acct.AccountConfig(),
		acct.Worker(), defaults)

	addTab := func() {
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
	}

	if attach {
		forwardAttach(store, composer, msg, addTab)
	} else {
		forwardBodyPart(store, composer, msg, addTab)
	}
	return nil
}

func forwardAttach(store *lib.MessageStore, composer *widgets.Composer,
	msg *models.MessageInfo, addTab func()) {

	store.FetchFull([]uint32{msg.Uid}, func(reader io.Reader) {
		tmpDir, err := ioutil.TempDir("", "aerc-tmp-attachment")
		if err != nil {
			// TODO: Do something with the error
			addTab()
			return
		}
		tmpFileName := path.Join(tmpDir,
			strings.ReplaceAll(fmt.Sprintf("%s.eml", msg.Envelope.Subject), "/", "-"))
		tmpFile, err := os.Create(tmpFileName)
		if err != nil {
			println(err)
			// TODO: Do something with the error
			addTab()
			return
		}

		defer tmpFile.Close()
		io.Copy(tmpFile, reader)
		composer.AddAttachment(tmpFileName)
		composer.OnClose(func(composer *widgets.Composer) {
			os.RemoveAll(tmpDir)
		})
		addTab()
	})
}

func forwardBodyPart(store *lib.MessageStore, composer *widgets.Composer,
	msg *models.MessageInfo, addTab func()) {
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

		pipeout, pipein := io.Pipe()
		scanner := bufio.NewScanner(part.Body)
		go composer.PrependContents(pipeout)
		// TODO: Let user customize the date format used here
		io.WriteString(pipein, fmt.Sprintf("Forwarded message from %s on %s:\n\n",
			msg.Envelope.From[0].Name,
			msg.Envelope.Date.Format("Mon Jan 2, 2006 at 3:04 PM")))
		for scanner.Scan() {
			io.WriteString(pipein, fmt.Sprintf("%s\n", scanner.Text()))
		}
		pipein.Close()
		pipeout.Close()
		addTab()
	})
}
