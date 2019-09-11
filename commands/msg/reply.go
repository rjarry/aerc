package msg

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	gomail "net/mail"
	"strings"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"

	"git.sr.ht/~sircmpwn/aerc/models"
	"git.sr.ht/~sircmpwn/aerc/widgets"
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
	opts, optind, err := getopt.Getopts(args, "aq")
	if err != nil {
		return err
	}
	if optind != len(args) {
		return errors.New("Usage: reply [-aq]")
	}
	var (
		quote    bool
		replyAll bool
	)
	for _, opt := range opts {
		switch opt.Option {
		case 'a':
			replyAll = true
		case 'q':
			quote = true
		}
	}

	widget := aerc.SelectedTab().(widgets.ProvidesMessage)
	acct := widget.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	conf := acct.AccountConfig()
	us, _ := gomail.ParseAddress(conf.From)
	store := widget.Store()
	if store == nil {
		return errors.New("Cannot perform action. Messages still loading")
	}
	msg, err := widget.SelectedMessage()
	if err != nil {
		return err
	}
	acct.Logger().Println("Replying to email " + msg.Envelope.MessageId)

	var (
		to     []string
		cc     []string
		toList []*models.Address
	)
	if args[0] == "reply" {
		if len(msg.Envelope.ReplyTo) != 0 {
			toList = msg.Envelope.ReplyTo
		} else {
			toList = msg.Envelope.From
		}
		for _, addr := range toList {
			if addr.Name != "" {
				to = append(to, fmt.Sprintf("%s <%s@%s>",
					addr.Name, addr.Mailbox, addr.Host))
			} else {
				to = append(to, fmt.Sprintf("<%s@%s>", addr.Mailbox, addr.Host))
			}
		}
		if replyAll {
			for _, addr := range msg.Envelope.Cc {
				cc = append(cc, addr.Format())
			}
			for _, addr := range msg.Envelope.To {
				address := fmt.Sprintf("%s@%s", addr.Mailbox, addr.Host)
				if address == us.Address {
					continue
				}
				to = append(to, addr.Format())
			}
		}
	}

	var subject string
	if !strings.HasPrefix(strings.ToLower(msg.Envelope.Subject), "re: ") {
		subject = "Re: " + msg.Envelope.Subject
	} else {
		subject = msg.Envelope.Subject
	}

	defaults := map[string]string{
		"To":          strings.Join(to, ", "),
		"Cc":          strings.Join(cc, ", "),
		"Subject":     subject,
		"In-Reply-To": msg.Envelope.MessageId,
	}

	composer := widgets.NewComposer(aerc, aerc.Config(),
		acct.AccountConfig(), acct.Worker(), defaults)

	if args[0] == "reply" {
		composer.FocusTerminal()
	}

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

	if quote {
		var (
			path []int
			part *models.BodyStructure
		)
		if len(msg.BodyStructure.Parts) != 0 {
			part, path = findPlaintext(msg.BodyStructure, path)
		}
		if part == nil {
			part = msg.BodyStructure
			path = []int{1}
		}

		store.FetchBodyPart(msg.Uid, path, func(reader io.Reader) {
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

			pipeout, pipein := io.Pipe()
			scanner := bufio.NewScanner(part.Body)
			go composer.PrependContents(pipeout)
			// TODO: Let user customize the date format used here
			io.WriteString(pipein, fmt.Sprintf("On %s %s wrote:\n",
				msg.Envelope.Date.Format("Mon Jan 2, 2006 at 3:04 PM"),
				msg.Envelope.From[0].Name))
			for scanner.Scan() {
				io.WriteString(pipein, fmt.Sprintf("> %s\n", scanner.Text()))
			}
			pipein.Close()
			pipeout.Close()
			addTab()
		})
	} else {
		addTab()
	}

	return nil
}

func findPlaintext(bs *models.BodyStructure,
	path []int) (*models.BodyStructure, []int) {

	for i, part := range bs.Parts {
		cur := append(path, i+1)
		if strings.ToLower(part.MIMEType) == "text" &&
			strings.ToLower(part.MIMESubType) == "plain" {
			return part, cur
		}
		if strings.ToLower(part.MIMEType) == "multipart" {
			if part, path := findPlaintext(part, cur); path != nil {
				return part, path
			}
		}
	}

	return nil, nil
}
