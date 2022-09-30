package msgview

import (
	"fmt"
	"io"
	"mime"
	"os"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/widgets"
)

type Open struct{}

func init() {
	register(Open{})
}

func (Open) Aliases() []string {
	return []string{"open", "open-link"}
}

func (Open) Complete(aerc *widgets.Aerc, args []string) []string {
	mv := aerc.SelectedTabContent().(*widgets.MessageViewer)
	if mv != nil {
		if p := mv.SelectedMessagePart(); p != nil {
			return commands.CompletionFromList(aerc, p.Links, args)
		}
	}
	return nil
}

func (Open) Execute(aerc *widgets.Aerc, args []string) error {
	mv := aerc.SelectedTabContent().(*widgets.MessageViewer)
	p := mv.SelectedMessagePart()

	if args[0] == "open-link" && len(args) > 1 {
		if link := args[1]; link != "" {
			go func() {
				if err := lib.XDGOpen(link); err != nil {
					aerc.PushError("open: " + err.Error())
				}
			}()
		}
		return nil
	}

	mv.MessageView().FetchBodyPart(p.Index, func(reader io.Reader) {
		extension := ""
		mimeType := ""

		// try to determine the correct extension based on mimetype
		if part, err := mv.MessageView().BodyStructure().PartAtIndex(p.Index); err == nil {
			mimeType = fmt.Sprintf("%s/%s", part.MIMEType, part.MIMESubType)
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
