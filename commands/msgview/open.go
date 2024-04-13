package msgview

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/lib/log"
)

type Open struct {
	Delete bool   `opt:"-d"`
	Cmd    string `opt:"..." required:"false"`
}

func init() {
	commands.Register(Open{})
}

func (Open) Context() commands.CommandContext {
	return commands.MESSAGE_VIEWER
}

func (Open) Options() string {
	return "d"
}

func (Open) Aliases() []string {
	return []string{"open"}
}

func (o Open) Execute(args []string) error {
	mv := app.SelectedTabContent().(*app.MessageViewer)
	if mv == nil {
		return errors.New("open only supported selected message parts")
	}
	p := mv.SelectedMessagePart()

	mv.MessageView().FetchBodyPart(p.Index, func(reader io.Reader) {
		mimeType := ""

		part, err := mv.MessageView().BodyStructure().PartAtIndex(p.Index)
		if err != nil {
			app.PushError(err.Error())
			return
		}
		mimeType = part.FullMIMEType()

		tmpDir, err := os.MkdirTemp(os.TempDir(), "aerc-*")
		if err != nil {
			app.PushError(err.Error())
			return
		}
		tmpFile, err := os.Create(filepath.Join(tmpDir, part.FileName()))
		if err != nil {
			app.PushError(err.Error())
			return
		}

		_, err = io.Copy(tmpFile, reader)
		tmpFile.Close()
		if err != nil {
			app.PushError(err.Error())
			return
		}

		go func() {
			defer log.PanicHandler()
			if o.Delete {
				defer os.RemoveAll(tmpDir)
			}
			err = lib.XDGOpenMime(tmpFile.Name(), mimeType, o.Cmd)
			if err != nil {
				app.PushError("open: " + err.Error())
			}
		}()
	})

	return nil
}
