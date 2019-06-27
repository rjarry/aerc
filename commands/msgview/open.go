package msgview

import (
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"mime/quotedprintable"
	"os"
	"os/exec"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Open struct{}

func init() {
	register(Open{})
}

func (_ Open) Aliases() []string {
	return []string{"open"}
}

func (_ Open) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (_ Open) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: open")
	}

	mv := aerc.SelectedTab().(*widgets.MessageViewer)
	p := mv.CurrentPart()

	p.Store.FetchBodyPart(p.Msg.Uid, p.Index, func(reader io.Reader) {
		// email parts are encoded as 7bit (plaintext), quoted-printable, or base64

		if strings.EqualFold(p.Part.Encoding, "base64") {
			reader = base64.NewDecoder(base64.StdEncoding, reader)
		} else if strings.EqualFold(p.Part.Encoding, "quoted-printable") {
			reader = quotedprintable.NewReader(reader)
		}

		tmpFile, err := ioutil.TempFile(os.TempDir(), "aerc-")
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, reader)
		if err != nil {
			aerc.PushError(" " + err.Error())
			return
		}

		cmd := exec.Command("xdg-open", tmpFile.Name())
		err = cmd.Run()
		if err != nil {
			aerc.PushError(" " + err.Error())
		}

		aerc.PushStatus("Opened", 10*time.Second)
	})

	return nil
}
