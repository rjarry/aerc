package compose

import (
	"bytes"
	"time"

	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Postpone struct {
	Folder string `opt:"-t" complete:"CompleteFolder"`
}

func init() {
	register(Postpone{})
}

func (Postpone) Aliases() []string {
	return []string{"postpone"}
}

func (*Postpone) CompleteFolder(arg string) []string {
	return commands.GetFolders(arg)
}

func (p Postpone) Execute(args []string) error {
	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	store := acct.Store()
	if store == nil {
		return errors.New("No message store selected")
	}
	tab := app.SelectedTab()
	if tab == nil {
		return errors.New("No tab selected")
	}
	composer, _ := tab.Content.(*app.Composer)
	config := composer.Config()
	tabName := tab.Name

	targetFolder := config.Postpone
	if composer.RecalledFrom() != "" {
		targetFolder = composer.RecalledFrom()
	}
	if p.Folder != "" {
		targetFolder = p.Folder
	}
	if targetFolder == "" {
		return errors.New("No Postpone location configured")
	}

	log.Tracef("Postponing mail")

	header, err := composer.PrepareHeader()
	if err != nil {
		return errors.Wrap(err, "PrepareHeader")
	}
	header.SetContentType("text/plain", map[string]string{"charset": "UTF-8"})
	header.Set("Content-Transfer-Encoding", "quoted-printable")
	worker := composer.Worker()
	dirs := acct.Directories().List()
	alreadyCreated := false
	for _, dir := range dirs {
		if dir == targetFolder {
			alreadyCreated = true
			break
		}
	}

	errChan := make(chan string)

	// run this as a goroutine so we can make other progress. The message
	// will be saved once the directory is created.
	go func() {
		defer log.PanicHandler()

		errStr := <-errChan
		if errStr != "" {
			app.PushError(errStr)
			return
		}

		handleErr := func(err error) {
			app.PushError(err.Error())
			log.Errorf("Postponing failed: %v", err)
			app.NewTab(composer, tabName)
		}

		app.RemoveTab(composer, false)
		buf := &bytes.Buffer{}

		err = composer.WriteMessage(header, buf)
		if err != nil {
			handleErr(errors.Wrap(err, "WriteMessage"))
			return
		}
		store.Append(
			targetFolder,
			models.SeenFlag,
			time.Now(),
			buf,
			buf.Len(),
			func(msg types.WorkerMessage) {
				switch msg := msg.(type) {
				case *types.Done:
					app.PushStatus("Message postponed.", 10*time.Second)
					composer.SetPostponed()
					composer.Close()
				case *types.Error:
					handleErr(msg.Error)
				}
			},
		)
	}()

	if !alreadyCreated {
		// to synchronise the creating of the directory
		worker.PostAction(&types.CreateDirectory{
			Directory: targetFolder,
		}, func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				errChan <- ""
			case *types.Error:
				errChan <- msg.Error.Error()
			}
		})
	} else {
		errChan <- ""
	}

	return nil
}
