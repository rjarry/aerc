package account

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"git.sr.ht/~rjarry/aerc/commands"
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

func (Recover) Options() string {
	return "f"
}

func (r Recover) Complete(aerc *widgets.Aerc, args []string) []string {
	// file name of temp file is hard-coded in the NewComposer() function
	files, err := filepath.Glob(
		filepath.Join(os.TempDir(), "aerc-compose-*.eml"),
	)
	if err != nil {
		return nil
	}
	return commands.CompletionFromList(aerc, files,
		commands.Operands(args, r.Options()))
}

func (r Recover) Execute(aerc *widgets.Aerc, args []string) error {
	// Complete() expects to be passed only the arguments, not including the command name
	if len(Recover{}.Complete(aerc, args[1:])) == 0 {
		return errors.New("No messages to recover.")
	}

	force := false

	opts, optind, err := getopt.Getopts(args, r.Options())
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
		acct.AccountConfig(), acct.Worker(),
		"", nil, nil, bytes.NewReader(data))
	if err != nil {
		return err
	}
	composer.Tab = aerc.NewTab(composer, "Recovered")

	// remove file if force flag is set
	if force {
		err = os.Remove(args[optind])
		if err != nil {
			return err
		}
	}

	return nil
}
