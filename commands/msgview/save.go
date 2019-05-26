package msgview

import (
	"encoding/base64"
	"errors"
	"io"
	"mime/quotedprintable"
	"os"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"github.com/mitchellh/go-homedir"
)

func init() {
	register("save", Save)
}

func Save(aerc *widgets.Aerc, args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: :save <path>")
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

		target, err := homedir.Expand(args[1])
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}

		f, err := os.Create(target)
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}
		defer f.Close()

		_, err = io.Copy(f, reader)
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}

		aerc.PushStatus("Saved to "+target, 10*time.Second)
	})

	return nil
}
