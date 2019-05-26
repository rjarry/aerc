package msgview

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/quotedprintable"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

func init() {
	register("pipe", Pipe)
}

func Pipe(aerc *widgets.Aerc, args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: :pipe <cmd> [args...]")
	}

	mv := aerc.SelectedTab().(*widgets.MessageViewer)
	p := mv.CurrentPart()

	p.Store.FetchBodyPart(p.Msg.Uid, p.Index, func(reader io.Reader) {
		// email parts are encoded as 7bit (plaintext), quoted-printable, or base64
		switch p.Part.Encoding {
		case "base64":
			reader = base64.NewDecoder(base64.StdEncoding, reader)
		case "quoted-printable":
			reader = quotedprintable.NewReader(reader)
		}

		term, err := commands.QuickTerm(aerc, args[1:], reader)
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}
		name := fmt.Sprintf("%s <%s/[%d]", args[1], p.Msg.Envelope.Subject, p.Index)
		aerc.NewTab(term, name)
	})

	return nil
}
