package compose

import (
	"fmt"
	"time"

	"git.sr.ht/~sircmpwn/aerc/widgets"
	"github.com/gdamore/tcell"
)

type Detach struct{}

func init() {
	register(Detach{})
}

func (_ Detach) Aliases() []string {
	return []string{"detach"}
}

func (_ Detach) Complete(aerc *widgets.Aerc, args []string) []string {
	composer, _ := aerc.SelectedTab().(*widgets.Composer)

	return composer.GetAttachments()
}

func (_ Detach) Execute(aerc *widgets.Aerc, args []string) error {
	var path string
	composer, _ := aerc.SelectedTab().(*widgets.Composer)

	if len(args) > 2 {
		return fmt.Errorf("Usage: :detach [path]")
	}

	if len(args) == 2 {
		path = args[1]
	} else {
		// if no attachment is specified, delete the first in the list
		atts := composer.GetAttachments()
		if len(atts) > 0 {
			path = atts[0]
		} else {
			return fmt.Errorf("No attachments to delete")
		}
	}

	if err := composer.DeleteAttachment(path); err != nil {
		return err
	}

	aerc.PushStatus(fmt.Sprintf("Detached %s", path), 10*time.Second).
		Color(tcell.ColorDefault, tcell.ColorGreen)

	return nil
}
