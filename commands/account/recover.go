package account

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/lib/ui"
	"git.sr.ht/~rjarry/aerc/logging"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/widgets"
	"git.sr.ht/~sircmpwn/getopt"
)

type Recover struct{}

func init() {
	register(Recover{})
}

func (Recover) Aliases() []string {
	return []string{"recover"}
}

func (Recover) Complete(aerc *widgets.Aerc, args []string) []string {
	// file name of temp file is hard-coded in the NewComposer() function
	files, err := filepath.Glob(
		filepath.Join(os.TempDir(), "aerc-compose-*.eml"),
	)
	if err != nil {
		return make([]string, 0)
	}
	// if nothing is entered yet, return all files
	if len(args) == 0 {
		return files
	}
	switch args[0] {
	case "-":
		return []string{"-f"}
	case "-f":
		if len(args) == 1 {
			for i, file := range files {
				files[i] = args[0] + " " + file
			}
			return files
		} else {
			// only accepts one file to recover
			return commands.FilterList(files, args[1], args[0]+" ",
				aerc.SelectedAccountUiConfig().FuzzyComplete)
		}
	default:
		// only accepts one file to recover
		return commands.FilterList(files, args[0], "", aerc.SelectedAccountUiConfig().FuzzyComplete)
	}
}

func (Recover) Execute(aerc *widgets.Aerc, args []string) error {
	// Complete() expects to be passed only the arguments, not including the command name
	if len(Recover{}.Complete(aerc, args[1:])) == 0 {
		return errors.New("No messages to recover.")
	}

	force := false

	opts, optind, err := getopt.Getopts(args, "f")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		if opt.Option == 'f' {
			force = true
		}
	}

	if len(args) <= optind {
		return errors.New("Usage: recover [-f] <file>")
	}

	acct := aerc.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	readData := func() ([]byte, error) {
		recoverFile, err := os.Open(args[optind])
		if err != nil {
			return nil, err
		}
		defer recoverFile.Close()
		data, err := io.ReadAll(recoverFile)
		if err != nil {
			return nil, err
		}
		return data, nil
	}
	data, err := readData()
	if err != nil {
		return err
	}

	composer, err := widgets.NewComposer(aerc, acct,
		aerc.Config(), acct.AccountConfig(), acct.Worker(),
		"", nil, models.OriginalMail{})
	if err != nil {
		return err
	}

	tab := aerc.NewTab(composer, "Recovered")
	composer.OnHeaderChange("Subject", func(subject string) {
		tab.Name = subject
		ui.Invalidate()
	})
	go func() {
		defer logging.PanicHandler()

		composer.AppendContents(bytes.NewReader(data))
	}()

	// remove file if force flag is set
	if force {
		err = os.Remove(args[optind])
		if err != nil {
			return err
		}
	}

	return nil
}
