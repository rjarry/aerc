package account

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	mboxer "git.sr.ht/~rjarry/aerc/worker/mbox"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type ImportMbox struct{}

func init() {
	register(ImportMbox{})
}

func (ImportMbox) Aliases() []string {
	return []string{"import-mbox"}
}

func (ImportMbox) Complete(aerc *widgets.Aerc, args []string) []string {
	return commands.CompletePath(filepath.Join(args...))
}

func (ImportMbox) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 2 {
		return importFolderUsage(args[0])
	}
	filename := args[1]

	acct := aerc.SelectedAccount()
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
		statusInfo := fmt.Sprintln("Importing", filename, "to folder", folder)
		aerc.PushStatus(statusInfo, 10*time.Second)
		acct.Logger().Println(args[0], statusInfo)
		f, err := os.Open(filename)
		if err != nil {
			aerc.PushError(err.Error())
			return
		}
		defer f.Close()

		messages, err := mboxer.Read(f)
		if err != nil {
			aerc.PushError(err.Error())
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
					acct.Logger().Println(fmt.Sprintf("%s: could not get reader for uid %d", args[0], m.UID()))
					break
				}
				nbytes, _ := io.Copy(&buf, r)
				worker.PostAction(&types.AppendMessage{
					Destination: folder,
					Flags:       []models.Flag{models.SeenFlag},
					Date:        time.Now(),
					Reader:      &buf,
					Length:      int(nbytes),
				}, func(msg types.WorkerMessage) {
					switch msg := msg.(type) {
					case *types.Unsupported:
						errMsg := fmt.Sprintf("%s: AppendMessage is unsupported", args[0])
						acct.Logger().Println(errMsg)
						aerc.PushError(errMsg)
						return
					case *types.Error:
						acct.Logger().Println(args[0], msg.Error.Error())
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
						acct.Logger().Println(args[0], "sleeping for", sleeping, "before append message", i, "again")
						time.Sleep(sleeping)
					}
				case <-time.After(30 * time.Second):
					acct.Logger().Println(args[0], "timed-out; appended", appended, "of", len(messages))
					return
				}
			}
		}
		infoStr := fmt.Sprintf("%s: imported %d of %d sucessfully.", args[0], appended, len(messages))
		acct.Logger().Println(infoStr)
		aerc.SetStatus(infoStr)
	}

	if len(store.Uids()) > 0 {
		confirm := widgets.NewSelectorDialog(
			"Selected directory is not empty",
			fmt.Sprintf("Import mbox file to %s anyways?", folder),
			[]string{"No", "Yes"}, 0, aerc.SelectedAccountUiConfig(),
			func(option string, err error) {
				aerc.CloseDialog()
				aerc.Invalidate()
				switch option {
				case "Yes":
					go importFolder()
				}
				return
			},
		)
		aerc.AddDialog(confirm)
	} else {
		go importFolder()
	}

	return nil
}

func importFolderUsage(cmd string) error {
	return fmt.Errorf("Usage: %s <filename>", cmd)
}
