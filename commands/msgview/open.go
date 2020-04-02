package msgview

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"time"

	"git.sr.ht/~sircmpwn/aerc/lib"
	"git.sr.ht/~sircmpwn/aerc/widgets"
)

type Open struct{}

func init() {
	register(Open{})
}

func (Open) Aliases() []string {
	return []string{"open"}
}

func (Open) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Open) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: open")
	}

	mv := aerc.SelectedTab().(*widgets.MessageViewer)
	p := mv.SelectedMessagePart()

	store := mv.Store()
	store.FetchBodyPart(p.Msg.Uid, p.Msg.BodyStructure, p.Index, func(reader io.Reader) {
		extension := ""
		// try to determine the correct extension based on mimetype
		if part, err := p.Msg.BodyStructure.PartAtIndex(p.Index); err == nil {
			mimeType := fmt.Sprintf("%s/%s", part.MIMEType, part.MIMESubType)

			if exts, _ := mime.ExtensionsByType(mimeType); exts != nil && len(exts) > 0 {
				extension = exts[0]
			}
		}

		tmpFile, err := ioutil.TempFile(os.TempDir(), "aerc-*"+extension)
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

		err = lib.OpenFile(tmpFile.Name())
		if err != nil {
			aerc.PushError(" " + err.Error())
		}

		aerc.PushStatus("Opened", 10*time.Second)
	})

	return nil
}
