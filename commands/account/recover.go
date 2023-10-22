package account

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"git.sr.ht/~rjarry/aerc/app"
	"git.sr.ht/~rjarry/aerc/commands"
	"git.sr.ht/~rjarry/aerc/config"
)

type Recover struct {
	Force  bool   `opt:"-f"`
	Edit   bool   `opt:"-e"`
	NoEdit bool   `opt:"-E"`
	File   string `opt:"file" complete:"CompleteFile"`
}

func init() {
	register(Recover{})
}

func (Recover) Aliases() []string {
	return []string{"recover"}
}

func (Recover) Options() string {
	return "feE"
}

func (*Recover) CompleteFile(arg string) []string {
	// file name of temp file is hard-coded in the NewComposer() function
	files, err := filepath.Glob(
		filepath.Join(os.TempDir(), "aerc-compose-*.eml"),
	)
	if err != nil {
		return nil
	}
	return commands.CompletionFromList(files, arg)
}

func (r Recover) Execute(args []string) error {
	file, err := os.Open(r.File)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	acct := app.SelectedAccount()
	if acct == nil {
		return errors.New("No account selected")
	}

	editHeaders := (config.Compose.EditHeaders || r.Edit) && !r.NoEdit

	composer, err := app.NewComposer(acct,
		acct.AccountConfig(), acct.Worker(), editHeaders,
		"", nil, nil, bytes.NewReader(data))
	if err != nil {
		return err
	}
	composer.Tab = app.NewTab(composer, "Recovered")

	// remove file if force flag is set
	if r.Force {
		err = os.Remove(r.File)
		if err != nil {
			return err
		}
	}

	return nil
}
