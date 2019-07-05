package msg

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strings"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Pipe struct{}

func init() {
	register(Pipe{})
}

func (_ Pipe) Aliases() []string {
	return []string{"pipe"}
}

func (_ Pipe) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Pipe) Execute(aerc *widgets.Aerc, args []string) error {
	var (
		pipeFull bool
		pipePart bool
	)
	// TODO: let user specify part by index or preferred mimetype
	opts, optind, err := getopt.Getopts(args, "mp")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'm':
			if pipePart {
				return errors.New("-m and -p are mutually exclusive")
			}
			pipeFull = true
		case 'p':
			if pipeFull {
				return errors.New("-m and -p are mutually exclusive")
			}
			pipePart = true
		}
	}
	cmd := args[optind:]
	if len(cmd) == 0 {
		return errors.New("Usage: pipe [-mp] <cmd> [args...]")
	}

	provider := aerc.SelectedTab().(widgets.ProvidesMessage)
	if !pipeFull && !pipePart {
		if _, ok := provider.(*widgets.MessageViewer); ok {
			pipePart = true
		} else if _, ok := provider.(*widgets.AccountView); ok {
			pipeFull = true
		} else {
			return errors.New(
				"Neither -m nor -p specified and cannot infer default")
		}
	}

	if pipeFull {
		store := provider.Store()
		msg := provider.SelectedMessage()
		store.FetchFull([]uint32{msg.Uid}, func(reader io.Reader) {
			term, err := commands.QuickTerm(aerc, cmd, reader)
			if err != nil {
				aerc.PushError(" " + err.Error())
				return
			}
			name := cmd[0] + " <" + msg.Envelope.Subject
			aerc.NewTab(term, name)
		})
	} else if pipePart {
		p := provider.SelectedMessagePart()
		p.Store.FetchBodyPart(p.Msg.Uid, p.Index, func(reader io.Reader) {
			// email parts are encoded as 7bit (plaintext), quoted-printable, or base64
			if strings.EqualFold(p.Part.Encoding, "base64") {
				reader = base64.NewDecoder(base64.StdEncoding, reader)
			} else if strings.EqualFold(p.Part.Encoding, "quoted-printable") {
				reader = quotedprintable.NewReader(reader)
			}

			term, err := commands.QuickTerm(aerc, cmd, reader)
			if err != nil {
				aerc.PushError(" " + err.Error())
				return
			}
			name := fmt.Sprintf("%s <%s/[%d]", cmd[0], p.Msg.Envelope.Subject, p.Index)
			aerc.NewTab(term, name)
		})
	}

	return nil
}
