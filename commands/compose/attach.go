package compose

import (
	"fmt"
	"os"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/aerc/commands"
	"git.sr.ht/~sircmpwn/aerc/widgets"
	"github.com/mitchellh/go-homedir"
)

type Attach struct{}

func init() {
	register(Attach{})
}

func (Attach) Aliases() []string {
	return []string{"attach"}
}

func (Attach) Complete(aerc *widgets.Aerc, args []string) []string {
	path := strings.Join(args, " ")
	return commands.CompletePath(path)
}

func (Attach) Execute(aerc *widgets.Aerc, args []string) error {
	if len(args) == 1 {
		return fmt.Errorf("Usage: :attach <path>")
	}

	path := strings.Join(args[1:], " ")

	path, err := homedir.Expand(path)
	if err != nil {
		aerc.PushError(" "+err.Error(), 10*time.Second)
		return err
	}

	pathinfo, err := os.Stat(path)
	if err != nil {
		aerc.PushError(" "+err.Error(), 10*time.Second)
		return err
	} else if pathinfo.IsDir() {
		aerc.PushError("Attachment must be a file, not a directory", 10*time.Second)
		return nil
	}

	composer, _ := aerc.SelectedTab().(*widgets.Composer)
	composer.AddAttachment(path)

	aerc.PushSuccess(fmt.Sprintf("Attached %s", pathinfo.Name()), 10*time.Second)

	return nil
}
