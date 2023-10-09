package compose

import (
	"bytes"
	"time"

	"github.com/pkg/errors"

	"git.sr.ht/~sircmpwn/getopt"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

type Postpone struct{}

func init() {
	register(Postpone{})
}

func (Postpone) Aliases() []string {
	return []string{"postpone"}
}

func (Postpone) Options() string {
	return "t:"
}

func (Postpone) CompleteOption(aerc *app.Aerc, r rune, arg string) []string {
	var valid []string
	if r == 't' {
		valid = commands.GetFolders(aerc, []string{arg})
	}
	return commands.CompletionFromList(aerc, valid, []string{arg})
}

func (Postpone) Complete(aerc *app.Aerc, args []string) []string {
	return nil
}

func (p Postpone) Execute(aerc *app.Aerc, args []string) error {
	opts, optind, err := getopt.Getopts(args, p.Options())
	if err != nil {
		return err
	}

	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}
	tab := aerc.SelectedTab()
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
	for _, opt := range opts {
		if opt.Option == 't' {
			targetFolder = opt.Value
		}
	}
	args = args[optind:]

	if len(args) != 0 {
		return errors.New("Usage: postpone [-t <folder>]")
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
			aerc.PushError(errStr)
			return
		}

		handleErr := func(err error) {
			aerc.PushError(err.Error())
			log.Errorf("Postponing failed: %v", err)
			aerc.NewTab(composer, tabName)
		}

		aerc.RemoveTab(composer, false)
		buf := &bytes.Buffer{}

		err = composer.WriteMessage(header, buf)
		if err != nil {
			handleErr(errors.Wrap(err, "WriteMessage"))
			return
		}
		worker.PostAction(&types.AppendMessage{
			Destination: targetFolder,
			Flags:       models.SeenFlag,
			Date:        time.Now(),
			Reader:      buf,
			Length:      buf.Len(),
		}, func(msg types.WorkerMessage) {
			switch msg := msg.(type) {
			case *types.Done:
				aerc.PushStatus("Message postponed.", 10*time.Second)
				composer.SetPostponed()
				composer.Close()
			case *types.Error:
				handleErr(msg.Error)
			}
		})
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
