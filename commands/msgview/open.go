package msgview

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"time"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/logging"
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
	mv := aerc.SelectedTab().(*widgets.MessageViewer)
	if mv != nil {
		if p := mv.SelectedMessagePart(); p != nil {
			return commands.CompletionFromList(aerc, p.Links, args)
		}
	}
	return nil
}

func (Open) Execute(aerc *widgets.Aerc, args []string) error {
	mv := aerc.SelectedTab().(*widgets.MessageViewer)
	p := mv.SelectedMessagePart()

	if args[0] == "open-link" && len(args) > 1 {
		if link := args[1]; link != "" {
			go func() {
				if err := lib.NewXDGOpen(link).Start(); err != nil {
					aerc.PushError(fmt.Sprintf("%s: %s", args[0], err.Error()))
				}
			}()
		}
		return nil
	}

	mv.MessageView().FetchBodyPart(p.Index, func(reader io.Reader) {
		extension := ""
		// try to determine the correct extension based on mimetype
		if part, err := mv.MessageView().BodyStructure().PartAtIndex(p.Index); err == nil {
			mimeType := fmt.Sprintf("%s/%s", part.MIMEType, part.MIMESubType)

			if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
				extension = exts[0]
			}
		}

		tmpFile, err := ioutil.TempFile(os.TempDir(), "aerc-*"+extension)
		if err != nil {
			aerc.PushError(err.Error())
			return
		}
		defer tmpFile.Close()

		_, err = io.Copy(tmpFile, reader)
		if err != nil {
			aerc.PushError(err.Error())
			return
		}

		xdg := lib.NewXDGOpen(tmpFile.Name())
		// pass through any arguments the user provided to the underlying handler
		if len(args) > 1 {
			xdg.SetArgs(args[1:])
		}
		err = xdg.Start()
		if err != nil {
			aerc.PushError(err.Error())
			return
		}
		go func() {
			defer logging.PanicHandler()

			err := xdg.Wait()
			if err != nil {
				aerc.PushError(err.Error())
			}
		}()

		aerc.PushStatus("Opened", 10*time.Second)
	})

	return nil
}
