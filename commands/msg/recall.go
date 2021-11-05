package msg

import (
	"io"

	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Recall struct{}

func init() {
	register(Recall{})
}

func (Recall) Aliases() []string {
	return []string{"recall"}
}

func (Recall) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Recall) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: recall")
	}

	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	if acct.SelectedDirectory() != acct.AccountConfig().Postpone {
		return errors.New("Can only recall from the postpone directory: " +
			acct.AccountConfig().Postpone)
	}
	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}

	msgInfo, err := widget.SelectedMessage()
	if err != nil {
		return errors.Wrap(err, "Recall failed")
	}
	acct.Logger().Println("Recalling message " + msgInfo.Envelope.MessageId)

	composer, err := widgets.NewComposer(aerc, acct, aerc.Config(),
		acct.AccountConfig(), acct.Worker(), "", msgInfo.RFC822Headers,
		models.OriginalMail{})
	if err != nil {
		return errors.Wrap(err, "Cannot open a new composer")
	}

	// focus the terminal since the header fields are likely already done
	composer.FocusTerminal()

	addTab := func() {
		subject := msgInfo.Envelope.Subject
		if subject == "" {
			subject = "Recalled email"
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
		composer.OnClose(func(composer *widgets.Composer) {
			worker := composer.Worker()
			uids := []uint32{msgInfo.Uid}

			worker.PostAction(&types.DeleteMessages{
				Uids: uids,
			}, func(msg types.WorkerMessage) {
				switch msg := msg.(type) {
				case *types.Error:
					aerc.PushError(msg.Error.Error())
					composer.Close()
				}
			})

			return
		})
	}

	// find the main body part and add it to the editor
	// TODO: copy all parts of the message over?
	var (
		path []int
		part *models.BodyStructure
	)
	if len(msgInfo.BodyStructure.Parts) != 0 {
		path = lib.FindPlaintext(msgInfo.BodyStructure, path)
	}
	part, err = msgInfo.BodyStructure.PartAtIndex(path)
	if part == nil || err != nil {
		part = msgInfo.BodyStructure
	}

	store.FetchBodyPart(msgInfo.Uid, path, func(reader io.Reader) {
		header := message.Header{}
		header.SetText(
			"Content-Transfer-Encoding", part.Encoding)
		header.SetContentType(part.MIMEType, part.Params)
		header.SetText("Content-Description", part.Description)
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
		composer.SetContents(part.Body)
		addTab()
	})

	return nil
}
