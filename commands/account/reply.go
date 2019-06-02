package account

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	gomail "net/mail"
	"regexp"
	"strings"

	"git.sr.ht/~sircmpwn/getopt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
	_ "github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("reply", Reply)
	register("forward", Reply)
}

var (
	atom *regexp.Regexp = regexp.MustCompile("^[a-z0-9!#$%7'*+-/=?^_`{}|~ ]+$")
)

func formatAddress(addr *imap.Address) string {
	if addr.PersonalName != "" {
		if atom.MatchString(addr.PersonalName) {
			return fmt.Sprintf("%s <%s@%s>",
				addr.PersonalName, addr.MailboxName, addr.HostName)
		} else {
			return fmt.Sprintf("\"%s\" <%s@%s>",
				strings.ReplaceAll(addr.PersonalName, "\"", "'"),
				addr.MailboxName, addr.HostName)
		}
	} else {
		return fmt.Sprintf("<%s@%s>", addr.MailboxName, addr.HostName)
	}
}

func Reply(aerc *widgets.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args[1:], "aq")
	if err != nil {
		return err
	}
	if optind != len(args)-1 {
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

	acct := aerc.SelectedAccount()
	conf := acct.AccountConfig()
	us, _ := gomail.ParseAddress(conf.From)
	store := acct.Messages().Store()
	msg := acct.Messages().Selected()
	acct.Logger().Println("Replying to email " + msg.Envelope.MessageId)

	var (
		to     []string
		cc     []string
		toList []*imap.Address
	)
	if args[0] == "reply" {
		if len(msg.Envelope.ReplyTo) != 0 {
			toList = msg.Envelope.ReplyTo
		} else {
			toList = msg.Envelope.From
		}
		for _, addr := range toList {
			if addr.PersonalName != "" {
				to = append(to, fmt.Sprintf("%s <%s@%s>",
					addr.PersonalName, addr.MailboxName, addr.HostName))
			} else {
				to = append(to, fmt.Sprintf("<%s@%s>",
					addr.MailboxName, addr.HostName))
			}
		}
		if replyAll {
			for _, addr := range msg.Envelope.Cc {
				cc = append(cc, formatAddress(addr))
			}
			for _, addr := range msg.Envelope.To {
				address := fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
				if address == us.Address {
					continue
				}
				to = append(to, formatAddress(addr))
			}
		}
	}

	var subject string
	if args[0] == "forward" {
		subject = "Fwd: " + msg.Envelope.Subject
	} else {
		if !strings.HasPrefix(msg.Envelope.Subject, "Re: ") {
			subject = "Re: " + msg.Envelope.Subject
		} else {
			subject = msg.Envelope.Subject
		}
	}

	composer := widgets.NewComposer(
		aerc.Config(), acct.AccountConfig(), acct.Worker()).
		Defaults(map[string]string{
			"To":          strings.Join(to, ", "),
			"Cc":          strings.Join(cc, ", "),
			"Subject":     subject,
			"In-Reply-To": msg.Envelope.MessageId,
		})

	if args[0] == "reply" {
		composer.FocusTerminal()
	}

	addTab := func() {
		tab := aerc.NewTab(composer, subject)
		composer.OnSubjectChange(func(subject string) {
			if subject == "" {
				tab.Name = "New email"
			} else {
				tab.Name = subject
			}
			tab.Content.Invalidate()
		})
	}

	if args[0] == "forward" {
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
				msg.Envelope.From[0].PersonalName,
				msg.Envelope.Date.Format("Mon Jan 2, 2006 at 3:04 PM")))
			for scanner.Scan() {
				io.WriteString(pipein, fmt.Sprintf("%s\n", scanner.Text()))
			}
			pipein.Close()
			pipeout.Close()
			addTab()
		})
	} else {
		if quote {
			var (
				path []int
				part *imap.BodyStructure
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
				go composer.SetContents(pipeout)
				// TODO: Let user customize the date format used here
				io.WriteString(pipein, fmt.Sprintf("On %s %s wrote:\n",
					msg.Envelope.Date.Format("Mon Jan 2, 2006 at 3:04 PM"),
					msg.Envelope.From[0].PersonalName))
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
	}

	return nil
}

func findPlaintext(bs *imap.BodyStructure,
	path []int) (*imap.BodyStructure, []int) {

	for i, part := range bs.Parts {
		cur := append(path, i+1)
		if part.MIMEType == "text" && part.MIMESubType == "plain" {
			return part, cur
		}
		if part.MIMEType == "multipart" {
			if part, path := findPlaintext(part, cur); path != nil {
				return part, path
			}
		}
	}

	return nil, nil
}
