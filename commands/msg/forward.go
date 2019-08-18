package msg

import (
	"bufio"
	"errors"
	"fmt"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"io"
)

type forward struct{}

func init() {
	register(forward{})
}

func (_ forward) Aliases() []string {
	return []string{"forward"}
}

func (_ forward) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ forward) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: forward")
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
		"Subject": subject,
	}
	composer := widgets.NewComposer(aerc.Config(), acct.AccountConfig(),
		acct.Worker(), defaults)

	addTab := func() {
		tab := aerc.NewTab(composer, subject)
		composer.OnHeaderChange("Subject", func(subject string) {
			if subject == "" {
				tab.Name = "New email"
			} else {
				tab.Name = subject
			}
			tab.Content.Invalidate()
		})
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

		pipeout, pipein := io.Pipe()
		scanner := bufio.NewScanner(part.Body)
		go composer.SetContents(pipeout)
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
	return nil
}
