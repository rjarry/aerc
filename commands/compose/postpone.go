package compose

import (
	"bytes"
	"time"

	"github.com/miolini/datacounter"
	"github.com/pkg/errors"

	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Postpone struct{}

func init() {
	register(Postpone{})
}

func (Postpone) Aliases() []string {
	return []string{"postpone"}
}

func (Postpone) Complete(aerc *widgets.Aerc, args []string) []string {
	return nil
}

func (Postpone) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: postpone")
	}
	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	tab := aerc.SelectedTab()
	if tab == nil {
		return errors.New("No tab selected")
	}
	composer, _ := tab.Content.(*widgets.Composer)
	config := composer.Config()
	tabName := tab.Name

	if config.Postpone == "" {
		return errors.New("No Postpone location configured")
	}

	logging.Infof("Postponing mail")

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
		if dir == config.Postpone {
			alreadyCreated = true
			break
		}
	}

	errChan := make(chan string)

	// run this as a goroutine so we can make other progress. The message
	// will be saved once the directory is created.
	go func() {
		defer logging.PanicHandler()

		errStr := <-errChan
		if errStr != "" {
			aerc.PushError(errStr)
			return
		}

		handleErr := func(err error) {
			aerc.PushError(err.Error())
			logging.Errorf("Postponing failed: %v", err)
			aerc.NewTab(composer, tabName)
		}

		aerc.RemoveTab(composer)
		var buf bytes.Buffer
		ctr := datacounter.NewWriterCounter(&buf)
		err = composer.WriteMessage(header, ctr)
		if err != nil {
			handleErr(errors.Wrap(err, "WriteMessage"))
			return
		}
		nbytes := int(ctr.Count())
		worker.PostAction(&types.AppendMessage{
			Destination: config.Postpone,
			Flags:       []models.Flag{models.SeenFlag},
			Date:        time.Now(),
			Reader:      &buf,
			Length:      int(nbytes),
		}, func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				aerc.PushStatus("Message postponed.", 10*time.Second)
				composer.Close()
			case *types.Error:
				handleErr(msg.Error)
			}
		})
	}()

	if !alreadyCreated {
		// to synchronise the creating of the directory
		worker.PostAction(&types.CreateDirectory{
			Directory: config.Postpone,
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
