package account

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
		return []string{}
	}
	arg := strings.Join(args, " ")
	if arg != "" {
		for i, file := range files {
			files[i] = strings.Join([]string{arg, file}, " ")
		}
	}
	return files
}

func (Recover) Execute(aerc *widgets.Aerc, args []string) error {
	if len(Recover{}.Complete(aerc, args)) == 0 {
		return errors.New("No messages to recover.")
	}

	force := false

	opts, optind, err := getopt.Getopts(args, "f")
	if err != nil {
		return err
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'f':
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
		data, err := ioutil.ReadAll(recoverFile)
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
		tab.Content.Invalidate()
	})
	go composer.AppendContents(bytes.NewReader(data))

	// remove file if force flag is set
	if force {
		err = os.Remove(args[optind])
		if err != nil {
			return err
		}
	}

	return nil
}
