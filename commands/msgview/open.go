package msgview

import (
	"errors"
	"io"
	"mime"
	"os"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/widgets"
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
	mv := aerc.SelectedTabContent().(*widgets.MessageViewer)
	if mv == nil {
		return errors.New("open only supported selected message parts")
	}
	p := mv.SelectedMessagePart()

	mv.MessageView().FetchBodyPart(p.Index, func(reader io.Reader) {
		extension := ""
		mimeType := ""

		// try to determine the correct extension based on mimetype
		if part, err := mv.MessageView().BodyStructure().PartAtIndex(p.Index); err == nil {
			mimeType = part.FullMIMEType()
			if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
				extension = exts[0]
			}
		}

		tmpFile, err := os.CreateTemp(os.TempDir(), "aerc-*"+extension)
		if err != nil {
			aerc.PushError(err.Error())
			return
		}

		_, err = io.Copy(tmpFile, reader)
		tmpFile.Close()
		if err != nil {
			aerc.PushError(err.Error())
			return
		}

		go func() {
			openers := aerc.Config().Openers
			err = lib.XDGOpenMime(tmpFile.Name(), mimeType, openers, args[1:])
			if err != nil {
				aerc.PushError("open: " + err.Error())
			}
		}()
	})

	return nil
}
