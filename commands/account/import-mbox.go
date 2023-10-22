package account

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	mboxer "git.sr.ht/~rjarry/aerc/worker/mbox"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type ImportMbox struct {
	Filename string `opt:"filename" complete:"CompleteFilename"`
}

func init() {
	register(ImportMbox{})
}

func (ImportMbox) Aliases() []string {
	return []string{"import-mbox"}
}

func (*ImportMbox) CompleteFilename(arg string) []string {
	return commands.CompletePath(arg)
}

func (i ImportMbox) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("No message store selected")
	}

	folder := acct.SelectedDirectory()
	if folder == "" {
		return errors.New("No directory selected")
	}

	importFolder := func() {
		defer log.PanicHandler()
		statusInfo := fmt.Sprintln("Importing", i.Filename, "to folder", folder)
		app.PushStatus(statusInfo, 10*time.Second)
		log.Debugf(statusInfo)
		f, err := os.Open(i.Filename)
		if err != nil {
			app.PushError(err.Error())
			return
		}
		defer f.Close()

		messages, err := mboxer.Read(f)
		if err != nil {
			app.PushError(err.Error())
			return
		}
		worker := acct.Worker()

		var appended uint32
		for i, m := range messages {
			done := make(chan bool)
			var retries int = 4
			for retries > 0 {
				var buf bytes.Buffer
				r, err := m.NewReader()
				if err != nil {
					log.Errorf("could not get reader for uid %d", m.UID())
					break
				}
				nbytes, _ := io.Copy(&buf, r)
				worker.PostAction(&types.AppendMessage{
					Destination: folder,
					Flags:       models.SeenFlag,
					Date:        time.Now(),
					Reader:      &buf,
					Length:      int(nbytes),
				}, func(msg types.WorkerMessage) {
					switch msg := msg.(type) {
					case *types.Unsupported:
						errMsg := fmt.Sprintf("%s: AppendMessage is unsupported", args[0])
						log.Errorf(errMsg)
						app.PushError(errMsg)
						return
					case *types.Error:
						log.Errorf("AppendMessage failed: %v", msg.Error)
						done <- false
					case *types.Done:
						atomic.AddUint32(&appended, 1)
						done <- true
					}
				})

				select {
				case ok := <-done:
					if ok {
						retries = 0
					} else {
						// error encountered; try to append again after a quick nap
						retries -= 1
						sleeping := time.Duration((5 - retries) * 1e9)

						log.Debugf("sleeping for %s before append message %d again", sleeping, i)
						time.Sleep(sleeping)
					}
				case <-time.After(30 * time.Second):
					log.Warnf("timed-out; appended %d of %d", appended, len(messages))
					return
				}
			}
		}
		infoStr := fmt.Sprintf("%s: imported %d of %d successfully.", args[0], appended, len(messages))
		log.Debugf(infoStr)
		app.PushSuccess(infoStr)
	}

	if len(store.Uids()) > 0 {
		confirm := app.NewSelectorDialog(
			"Selected directory is not empty",
			fmt.Sprintf("Import mbox file to %s anyways?", folder),
			[]string{"No", "Yes"}, 0, app.SelectedAccountUiConfig(),
			func(option string, err error) {
				app.CloseDialog()
				if option == "Yes" {
					go importFolder()
				}
			},
		)
		app.AddDialog(confirm)
	} else {
		go importFolder()
	}

	return nil
}
